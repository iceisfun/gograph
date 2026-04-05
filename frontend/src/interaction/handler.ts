import { vec2 } from '../core/geometry.js';
import type { AppStore } from '../state/store.js';
import type { ApiClient } from '../net/api.js';
import { hitTestNode, hitTestSlot, hitTestConnection } from './hit-test.js';
import { handleWheel, startPan, updatePan } from './camera.js';
import { startDrag, updateDrag, endDrag } from './drag.js';
import { startConnect, updateConnect, endConnect } from './connect.js';
import { handleClick, startBoxSelect, updateBoxSelect, endBoxSelect } from './select.js';
import { ContextMenu } from './context-menu.js';
import { ConfigModal } from './config-modal.js';

export class InteractionHandler {
    private canvas: HTMLCanvasElement;
    private store: AppStore;
    private api: ApiClient;
    private contextMenu: ContextMenu;
    private configModal: ConfigModal;
    private boundMouseDown: (e: MouseEvent) => void;
    private boundMouseMove: (e: MouseEvent) => void;
    private boundMouseUp: (e: MouseEvent) => void;
    private boundWheel: (e: WheelEvent) => void;
    private boundKeyDown: (e: KeyboardEvent) => void;
    private boundContextMenu: (e: MouseEvent) => void;
    private boundDblClick: (e: MouseEvent) => void;

    constructor(canvas: HTMLCanvasElement, store: AppStore, api: ApiClient) {
        this.canvas = canvas;
        this.store = store;
        this.api = api;

        const container = canvas.parentElement || document.body;
        this.contextMenu = new ContextMenu(container);
        this.configModal = new ConfigModal(container);

        this.boundMouseDown = this.onMouseDown.bind(this);
        this.boundMouseMove = this.onMouseMove.bind(this);
        this.boundMouseUp = this.onMouseUp.bind(this);
        this.boundWheel = this.onWheel.bind(this);
        this.boundKeyDown = this.onKeyDown.bind(this);
        this.boundContextMenu = this.onContextMenu.bind(this);
        this.boundDblClick = this.onDblClick.bind(this);
    }

    start(): void {
        this.canvas.addEventListener('mousedown', this.boundMouseDown);
        this.canvas.addEventListener('mousemove', this.boundMouseMove);
        this.canvas.addEventListener('mouseup', this.boundMouseUp);
        this.canvas.addEventListener('wheel', this.boundWheel, { passive: false });
        this.canvas.addEventListener('contextmenu', this.boundContextMenu);
        this.canvas.addEventListener('dblclick', this.boundDblClick);
        window.addEventListener('keydown', this.boundKeyDown);
    }

    stop(): void {
        this.canvas.removeEventListener('mousedown', this.boundMouseDown);
        this.canvas.removeEventListener('mousemove', this.boundMouseMove);
        this.canvas.removeEventListener('mouseup', this.boundMouseUp);
        this.canvas.removeEventListener('wheel', this.boundWheel);
        this.canvas.removeEventListener('contextmenu', this.boundContextMenu);
        this.canvas.removeEventListener('dblclick', this.boundDblClick);
        window.removeEventListener('keydown', this.boundKeyDown);
    }

    private getScreenPos(e: MouseEvent): { x: number; y: number } {
        const rect = this.canvas.getBoundingClientRect();
        return vec2(e.clientX - rect.left, e.clientY - rect.top);
    }

    private onMouseDown(e: MouseEvent): void {
        const screenPos = this.getScreenPos(e);
        const worldPos = this.store.camera.screenToWorld(screenPos);
        const interaction = this.store.interaction;

        // Middle mouse button or space+left click - pan
        if (e.button === 1) {
            interaction.dragState = startPan(screenPos);
            return;
        }

        if (e.button !== 0) return;

        // In view mode, only allow panning
        if (interaction.mode === 'view') {
            interaction.dragState = startPan(screenPos);
            return;
        }

        // Check slot hit first (higher priority than node)
        const slotHit = hitTestSlot(worldPos, this.store);
        if (slotHit) {
            // Verify it's an output slot to start a connection
            const node = this.store.graph.current?.nodes[slotHit.nodeId];
            if (node) {
                const nodeType = this.store.graph.getNodeType(node.type);
                const slot = nodeType?.slots.find(s => s.id === slotHit.slotId);
                if (slot?.direction === 'output') {
                    interaction.dragState = startConnect(slotHit.nodeId, slotHit.slotId, worldPos);
                    return;
                }
            }
        }

        // Check node hit
        const nodeHit = hitTestNode(worldPos, this.store);
        if (nodeHit) {
            handleClick(worldPos, this.store, e.shiftKey);

            if (e.shiftKey) {
                if (interaction.selectedNodes.has(nodeHit)) {
                    interaction.selectedNodes.delete(nodeHit);
                } else {
                    interaction.selectedNodes.add(nodeHit);
                }
            } else {
                interaction.selectedNodes.clear();
                interaction.selectedNodes.add(nodeHit);
            }

            interaction.dragState = startDrag(nodeHit, worldPos, this.store);
            return;
        }

        // Check connection hit
        const connHit = hitTestConnection(worldPos, this.store);
        if (connHit) {
            handleClick(worldPos, this.store, e.shiftKey);

            if (e.shiftKey) {
                if (interaction.selectedConnections.has(connHit)) {
                    interaction.selectedConnections.delete(connHit);
                } else {
                    interaction.selectedConnections.add(connHit);
                }
            } else {
                interaction.clearSelection();
                interaction.selectedConnections.add(connHit);
            }
            return;
        }

        // Empty space - start box select or pan
        handleClick(worldPos, this.store, e.shiftKey);

        if (e.altKey) {
            interaction.dragState = startPan(screenPos);
        } else {
            interaction.dragState = startBoxSelect(worldPos);
        }
    }

    private onMouseMove(e: MouseEvent): void {
        const screenPos = this.getScreenPos(e);
        const worldPos = this.store.camera.screenToWorld(screenPos);
        const interaction = this.store.interaction;
        const drag = interaction.dragState;

        if (drag) {
            switch (drag.type) {
                case 'pan':
                    updatePan(screenPos, this.store.camera, drag);
                    break;
                case 'node':
                    updateDrag(worldPos, this.store, drag);
                    break;
                case 'connection':
                    updateConnect(worldPos, drag);
                    break;
                case 'select':
                    updateBoxSelect(worldPos, drag);
                    break;
            }
            return;
        }

        // Update hover state
        interaction.clearHover();

        const slotHit = hitTestSlot(worldPos, this.store);
        if (slotHit) {
            interaction.hoveredSlot = slotHit;
            interaction.hoveredNode = slotHit.nodeId;
            return;
        }

        const nodeHit = hitTestNode(worldPos, this.store);
        if (nodeHit) {
            interaction.hoveredNode = nodeHit;
            return;
        }

        const connHit = hitTestConnection(worldPos, this.store);
        if (connHit) {
            interaction.hoveredConnection = connHit;
        }
    }

    private onMouseUp(e: MouseEvent): void {
        const screenPos = this.getScreenPos(e);
        const worldPos = this.store.camera.screenToWorld(screenPos);
        const interaction = this.store.interaction;
        const drag = interaction.dragState;

        if (drag) {
            switch (drag.type) {
                case 'node':
                    void endDrag(this.store, this.api);
                    break;
                case 'connection':
                    void endConnect(worldPos, this.store, this.api, drag);
                    break;
                case 'select':
                    endBoxSelect(this.store, drag);
                    break;
                case 'pan':
                    // Nothing to finalize
                    break;
            }
            interaction.dragState = null;
        }
    }

    private onWheel(e: WheelEvent): void {
        e.preventDefault();
        const screenPos = this.getScreenPos(e);
        handleWheel(screenPos, e.deltaY, this.store.camera);
    }

    private onKeyDown(e: KeyboardEvent): void {
        // Don't process shortcuts when modal is open
        if (this.configModal.visible) return;

        // Delete selected nodes/connections
        if (e.key === 'Delete' || e.key === 'Backspace') {
            if (this.store.interaction.mode !== 'edit') return;

            const graph = this.store.graph.current;
            if (!graph) return;

            // Remove selected connections
            for (const connId of this.store.interaction.selectedConnections) {
                this.store.graph.removeConnection(connId);
            }

            // Remove selected nodes and their connections
            for (const nodeId of this.store.interaction.selectedNodes) {
                graph.connections = graph.connections.filter(
                    c => c.fromNode !== nodeId && c.toNode !== nodeId,
                );
                delete graph.nodes[nodeId];
            }

            this.store.interaction.clearSelection();

            void this.api.updateGraph(graph.id, graph).catch(err => {
                console.error('Failed to persist deletion:', err);
            });
        }

        // Escape to deselect
        if (e.key === 'Escape') {
            this.store.interaction.clearSelection();
            this.store.interaction.dragState = null;
        }
    }

    private onContextMenu(e: MouseEvent): void {
        e.preventDefault();
        if (this.store.interaction.mode === 'view') return;

        const graph = this.store.graph.current;
        if (!graph) return;

        const screenPos = this.getScreenPos(e);
        const worldPos = this.store.camera.screenToWorld(screenPos);
        const nodeTypes = this.store.graph.nodeTypes;

        if (nodeTypes.length === 0) return;

        const items = nodeTypes.map(nt => ({
            label: nt.label,
            category: nt.category || 'other',
            onClick: () => {
                const id = typeof crypto !== 'undefined' && crypto.randomUUID
                    ? crypto.randomUUID()
                    : `node-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

                const node = {
                    id,
                    type: nt.name,
                    label: nt.label,
                    position: { x: worldPos.x, y: worldPos.y },
                };

                graph.nodes[id] = node;

                void this.api.addNode(graph.id, node).catch(err => {
                    console.error('Failed to persist new node:', err);
                });
            },
        }));

        this.contextMenu.show(screenPos.x, screenPos.y, items);
    }

    private onDblClick(e: MouseEvent): void {
        const screenPos = this.getScreenPos(e);
        const worldPos = this.store.camera.screenToWorld(screenPos);

        const nodeHit = hitTestNode(worldPos, this.store);
        if (!nodeHit) return;

        const graph = this.store.graph.current;
        if (!graph) return;

        const node = graph.nodes[nodeHit];
        if (!node) return;

        const nodeType = this.store.graph.getNodeType(node.type);
        if (!nodeType) return;

        // Show config modal for nodes with config or delay-category nodes
        if (node.config || nodeType.category === 'delay') {
            this.configModal.show(node, nodeType, (config) => {
                node.config = config;
                void this.api.updateGraph(graph.id, graph).catch(err => {
                    console.error('Failed to persist config update:', err);
                });
            });
        }
    }
}
