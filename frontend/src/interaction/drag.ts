import type { Vec2 } from '../core/geometry.js';
import { vec2, sub } from '../core/geometry.js';
import type { AppStore } from '../state/store.js';
import type { ApiClient } from '../net/api.js';
import type { DragNodeState } from '../state/interaction-state.js';

export function startDrag(nodeId: string, worldPos: Vec2, store: AppStore): DragNodeState {
    const graph = store.graph.current;
    if (!graph) {
        return { type: 'node', nodeId, startPos: worldPos, offset: vec2(0, 0) };
    }

    const node = graph.nodes[nodeId];
    if (!node) {
        return { type: 'node', nodeId, startPos: worldPos, offset: vec2(0, 0) };
    }

    const offset = sub(worldPos, { x: node.position.x, y: node.position.y });
    return { type: 'node', nodeId, startPos: worldPos, offset };
}

export function updateDrag(worldPos: Vec2, store: AppStore, dragState: DragNodeState): void {
    const graph = store.graph.current;
    if (!graph) return;

    const node = graph.nodes[dragState.nodeId];
    if (!node) return;

    node.position.x = worldPos.x - dragState.offset.x;
    node.position.y = worldPos.y - dragState.offset.y;
}

export async function endDrag(store: AppStore, api: ApiClient): Promise<void> {
    const graph = store.graph.current;
    if (!graph) return;

    try {
        await api.updateGraph(graph.id, graph);
    } catch (err) {
        console.error('Failed to persist node position:', err);
    }
}
