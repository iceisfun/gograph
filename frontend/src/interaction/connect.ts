import type { Vec2 } from '../core/geometry.js';
import type { AppStore } from '../state/store.js';
import type { ApiClient } from '../net/api.js';
import type { DragConnectionState } from '../state/interaction-state.js';
import { hitTestSlot } from './hit-test.js';

export function startConnect(
    nodeId: string,
    slotId: string,
    worldPos: Vec2,
): DragConnectionState {
    return {
        type: 'connection',
        fromNode: nodeId,
        fromSlot: slotId,
        currentPos: worldPos,
    };
}

export function updateConnect(worldPos: Vec2, dragState: DragConnectionState): void {
    dragState.currentPos = worldPos;
}

export async function endConnect(
    worldPos: Vec2,
    store: AppStore,
    api: ApiClient,
    dragState: DragConnectionState,
): Promise<void> {
    const target = hitTestSlot(worldPos, store);
    if (!target) return;

    // Don't connect to the same node
    if (target.nodeId === dragState.fromNode) return;

    // Verify that the target is an input slot
    const nodeType = store.graph.getNodeType(
        store.graph.current?.nodes[target.nodeId]?.type || '',
    );
    if (!nodeType) return;

    const targetSlot = nodeType.slots.find(s => s.id === target.slotId);
    if (!targetSlot || targetSlot.direction !== 'input') return;

    const graph = store.graph.current;
    if (!graph) return;

    // Create a new connection
    const connection = {
        id: `conn_${Date.now()}`,
        fromNode: dragState.fromNode,
        fromSlot: dragState.fromSlot,
        toNode: target.nodeId,
        toSlot: target.slotId,
    };

    graph.connections.push(connection);

    try {
        await api.updateGraph(graph.id, graph);
    } catch (err) {
        console.error('Failed to create connection:', err);
        // Rollback
        graph.connections = graph.connections.filter(c => c.id !== connection.id);
    }
}
