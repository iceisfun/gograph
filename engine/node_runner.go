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
	outputWires map[string][]WireRunner

	tickTimer    *time.Timer
	tickInterval time.Duration

	broker   EventBroker
	inputs      map[string]any    // Go-side mirror of self.inputs
	prevInputs  map[string]any    // previous input values for on_change
	stateOutputs map[string]any   // change detection for set()
	config   map[string]string // Go-side mirror of self.config
	nodeType graph.NodeType

	lastSlots   map[string]graph.ContentSlot // per-slot change detection
	lastLabel   string
	updateLabel func(nodeID, label string)

	ctx    context.Context
	cancel context.CancelFunc
}

func newNodeRunner(
	ctx context.Context,
	nodeID, graphID string,
	nt graph.NodeType,
	config map[string]string,
	broker EventBroker,
	updateLabel func(nodeID, label string),
) (*nodeRunner, error) {
	ctx, cancel := context.WithCancel(ctx)

	nr := &nodeRunner{
		nodeID:        nodeID,
		graphID:       graphID,
		inputChan:     make(chan Arrival, 16),
		controlChan:   make(chan ControlMsg, 8),
		scheduledTick: make(chan struct{}, 1),
		outputWires:   make(map[string][]WireRunner),
		broker:        broker,
		updateLabel:   updateLabel,
		inputs:        make(map[string]any),
		prevInputs:    make(map[string]any),
		stateOutputs:  make(map[string]any),
		config:        config,
		nodeType:      nt,
		lastSlots:     make(map[string]graph.ContentSlot),
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
		Set:          nr.set,
		Display:      nr.display,
		Glow:         nr.glow,
		Log:          nr.log,
		SetConfig:    nr.setConfig,
		SetLabel:     nr.setLabel,
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
	if a.Kind == graph.StateKind {
		nr.handleStateArrival(a)
		return
	}

	// Event arrival — update inputs and call on_event.
	nr.inputs[a.Slot] = a.Value
	nr.nvm.InputsTbl.SetString(a.Slot, golua.GoToLuaValue(a.Value))

	eventTbl := golua.BuildEventTable("arrival", a.Slot, a.Value, a.Source, nil)
	if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_event", eventTbl); err != nil {
		nr.log(fmt.Sprintf("on_event error: %v", err))
	}
}

func (nr *nodeRunner) handleStateArrival(a Arrival) {
	prev := nr.prevInputs[a.Slot]
	nr.prevInputs[a.Slot] = a.Value
	nr.inputs[a.Slot] = a.Value
	nr.nvm.InputsTbl.SetString(a.Slot, golua.GoToLuaValue(a.Value))

	changeTbl := golua.BuildChangeEventTable(a.Slot, a.Value, prev, a.Source)

	// Edge detection for boolean states.
	prevTruthy := isTruthy(prev)
	newTruthy := isTruthy(a.Value)

	if newTruthy && !prevTruthy {
		if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_high", changeTbl); err != nil {
			nr.log(fmt.Sprintf("on_high error: %v", err))
		}
	} else if !newTruthy && prevTruthy {
		if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_low", changeTbl); err != nil {
			nr.log(fmt.Sprintf("on_low error: %v", err))
		}
	}

	// Always fire on_change.
	if err := golua.CallHandler(nr.nvm.VM, nr.nvm.NodeTbl, "on_change", changeTbl); err != nil {
		nr.log(fmt.Sprintf("on_change error: %v", err))
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
			slot := msg.Wire.GetFromSlot()
			nr.outputWires[slot] = append(nr.outputWires[slot], msg.Wire)
		}

	case CtrlRemoveWire:
		if msg.Wire != nil {
			slot := msg.Wire.GetFromSlot()
			wires := nr.outputWires[slot]
			for i, w := range wires {
				if w.GetConnID() == msg.Wire.GetConnID() {
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
		case w.GetIntake() <- WireMessage{Value: value, Source: nr.nodeID}:
		case <-nr.ctx.Done():
			return
		}
	}
}

// set updates a state output, only propagating if the value changed.
func (nr *nodeRunner) set(slot string, value any) {
	prevStr := fmt.Sprintf("%v", nr.stateOutputs[slot])
	newStr := fmt.Sprintf("%v", value)
	if prevStr == newStr {
		return
	}
	nr.stateOutputs[slot] = value
	nr.emit(slot, value) // routes through wires (StateWire does SSE + change detect)
}

func (nr *nodeRunner) display(slotName string, slot graph.ContentSlot) {
	// Skip change detection if animation requested (always re-trigger).
	if slot.Animate == "" {
		if prev, ok := nr.lastSlots[slotName]; ok && prev == slot {
			return
		}
	}
	nr.lastSlots[slotName] = slot

	payload := graph.NodeContentPayload{
		Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
		NodeID:   nr.nodeID,
		Slots:    map[string]graph.ContentSlot{slotName: slot},
	}
	if slotName == "default" {
		payload.Text = slot.Text // backward compat
	}
	nr.broker.Publish(nr.graphID, graph.TypeNodeContent, payload)
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

func (nr *nodeRunner) setLabel(label string) {
	if nr.lastLabel == label {
		return
	}
	nr.lastLabel = label
	nr.updateLabel(nr.nodeID, label)
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
