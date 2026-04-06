package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/iceisfun/gograph/graph"
	golua "github.com/iceisfun/gograph/lua"
)

// nodeRunner is a single node's runtime — a goroutine with a persistent
// Lua VM. All VM access happens on this goroutine (thread safety by
// confinement).
type nodeRunner struct {
	nodeID  string
	graphID string
	nvm     *golua.NodeVM

	inputChan     chan Arrival
	controlChan   chan ControlMsg
	scheduledTick chan struct{} // one-shot tick signal from schedule_tick

	// Output wires, keyed by slot. Only accessed from this goroutine.
	outputWires map[string][]*Wire

	tickTimer    *time.Timer
	tickInterval time.Duration

	broker   EventBroker
	inputs   map[string]any    // Go-side mirror of self.inputs
	config   map[string]string // Go-side mirror of self.config
	nodeType graph.NodeType

	lastDisplay string // change detection

	ctx    context.Context
	cancel context.CancelFunc
}

func newNodeRunner(
	ctx context.Context,
	nodeID, graphID string,
	nt graph.NodeType,
	config map[string]string,
	broker EventBroker,
) (*nodeRunner, error) {
	ctx, cancel := context.WithCancel(ctx)

	nr := &nodeRunner{
		nodeID:        nodeID,
		graphID:       graphID,
		inputChan:     make(chan Arrival, 16),
		controlChan:   make(chan ControlMsg, 8),
		scheduledTick: make(chan struct{}, 1),
		outputWires:   make(map[string][]*Wire),
		broker:        broker,
		inputs:        make(map[string]any),
		config:        config,
		nodeType:      nt,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Stopped timer — init_tick will reset it.
	nr.tickTimer = time.NewTimer(0)
	if !nr.tickTimer.Stop() {
		<-nr.tickTimer.C
	}

	// Create persistent VM with callbacks that route to this runner.
	nvm, err := golua.CreateNodeVM(ctx, nodeID, graphID, nt, config, golua.NodeCallbacks{
		Emit:         nr.emit,
		Display:      nr.display,
		Glow:         nr.glow,
		Log:          nr.log,
		SetConfig:    nr.setConfig,
		InitTick:     nr.initTick,
		ScheduleTick: nr.scheduleTick,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	nr.nvm = nvm

	return nr, nil
}

// run is the node's main loop. Called as a goroutine.
func (nr *nodeRunner) run() {
	defer nr.cancel()

	// Fire on_init.
	if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_init", nil); err != nil {
		nr.log(fmt.Sprintf("on_init error: %v", err))
	}

	for {
		select {
		case a := <-nr.inputChan:
			nr.handleArrival(a)

		case msg := <-nr.controlChan:
			if nr.handleControl(msg) {
				return // shutdown
			}

		case <-nr.tickTimer.C:
			nr.handleTick()
			if nr.tickInterval > 0 {
				nr.tickTimer.Reset(nr.tickInterval)
			}

		case <-nr.scheduledTick:
			nr.handleTick()

		case <-nr.ctx.Done():
			golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_shutdown", nil)
			return
		}
	}
}

func (nr *nodeRunner) handleArrival(a Arrival) {
	// Update Go-side inputs.
	nr.inputs[a.Slot] = a.Value

	// Update Lua-side self.inputs in-place.
	nr.nvm.InputsTbl.SetString(a.Slot, golua.GoToLuaValue(a.Value))

	// Build event table and call on_event.
	eventTbl := golua.BuildEventTable("arrival", a.Slot, a.Value, a.Source, nil)
	if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_event", eventTbl); err != nil {
		nr.log(fmt.Sprintf("on_event error: %v", err))
	}
}

func (nr *nodeRunner) handleTick() {
	if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_tick", nil); err != nil {
		nr.log(fmt.Sprintf("on_tick error: %v", err))
	}
}

// handleControl processes a control message. Returns true if shutdown.
func (nr *nodeRunner) handleControl(msg ControlMsg) bool {
	defer func() {
		if msg.Done != nil {
			close(msg.Done)
		}
	}()

	switch msg.Kind {
	case CtrlClick:
		if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_click", nil); err != nil {
			nr.log(fmt.Sprintf("on_click error: %v", err))
		}

	case CtrlUpdateConfig:
		if msg.Config != nil {
			for k, v := range msg.Config {
				nr.config[k] = v
				nr.nvm.ConfigTbl.SetString(k, golua.GoToLuaValue(v))
			}
			golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_config", nil)
		}

	case CtrlConnect:
		eventTbl := golua.BuildConnectEventTable(msg.Conn)
		if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_connect", eventTbl); err != nil {
			nr.log(fmt.Sprintf("on_connect error: %v", err))
		}

	case CtrlDisconnect:
		eventTbl := golua.BuildConnectEventTable(msg.Conn)
		if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_disconnect", eventTbl); err != nil {
			nr.log(fmt.Sprintf("on_disconnect error: %v", err))
		}

	case CtrlAddWire:
		if msg.Wire != nil {
			slot := msg.Wire.FromSlot
			nr.outputWires[slot] = append(nr.outputWires[slot], msg.Wire)
		}

	case CtrlRemoveWire:
		if msg.Wire != nil {
			slot := msg.Wire.FromSlot
			wires := nr.outputWires[slot]
			for i, w := range wires {
				if w.ConnID == msg.Wire.ConnID {
					nr.outputWires[slot] = append(wires[:i], wires[i+1:]...)
					break
				}
			}
		}

	case CtrlShutdown:
		golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_shutdown", nil)
		return true
	}

	return false
}

// ---------------------------------------------------------------------------
// Callbacks — called from Lua during handler execution
// ---------------------------------------------------------------------------

func (nr *nodeRunner) emit(slot string, value any) {
	wires := nr.outputWires[slot]
	for _, w := range wires {
		select {
		case w.Intake <- WireMessage{Value: value, Source: nr.nodeID}:
		case <-w.ctx.Done():
			// Wire is dead (closed/cancelled) — skip it.
			continue
		case <-nr.ctx.Done():
			return
		}
	}
}

func (nr *nodeRunner) display(text string) {
	if text == nr.lastDisplay {
		return // no change
	}
	nr.lastDisplay = text
	nr.broker.Publish(nr.graphID, graph.TypeNodeContent, graph.NodeContentPayload{
		Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
		NodeID:   nr.nodeID,
		Text:     text,
	})
}

func (nr *nodeRunner) glow(durationMs int) {
	nr.broker.Publish(nr.graphID, graph.TypeNodeActive, graph.NodeActivePayload{
		Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
		NodeID:   nr.nodeID,
		Duration: durationMs,
	})
}

func (nr *nodeRunner) log(msg string) {
	fmt.Printf("[%s] %s\n", nr.nodeID, msg)
}

func (nr *nodeRunner) setConfig(key, value string) {
	nr.config[key] = value
}

func (nr *nodeRunner) initTick(ms int) {
	nr.tickInterval = time.Duration(ms) * time.Millisecond
	nr.tickTimer.Reset(nr.tickInterval)
}

func (nr *nodeRunner) scheduleTick(ms int) {
	go func() {
		if ms > 0 {
			timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
			select {
			case <-nr.ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
		select {
		case nr.scheduledTick <- struct{}{}:
		case <-nr.ctx.Done():
		}
	}()
}
