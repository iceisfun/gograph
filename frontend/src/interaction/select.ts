import type { Vec2 } from '../core/geometry.js';
import { rectsOverlap } from '../core/geometry.js';
import type { AppStore } from '../state/store.js';
import type { DragSelectState } from '../state/interaction-state.js';
import { getNodeBounds } from '../render/nodes.js';

export function handleClick(worldPos: Vec2, store: AppStore, shiftKey: boolean): void {
    // Hit-test is handled externally; this is about toggling selection
    // If shift is held, toggle; otherwise replace selection
    void worldPos;

    if (!shiftKey) {
        store.interaction.clearSelection();
    }
}

export function startBoxSelect(worldPos: Vec2): DragSelectState {
    return {
        type: 'select',
        startPos: worldPos,
        currentPos: worldPos,
    };
}

export function updateBoxSelect(worldPos: Vec2, dragState: DragSelectState): void {
    dragState.currentPos = worldPos;
}

export function endBoxSelect(store: AppStore, dragState: DragSelectState): void {
    const graph = store.graph.current;
    if (!graph) return;

    const x = Math.min(dragState.startPos.x, dragState.currentPos.x);
    const y = Math.min(dragState.startPos.y, dragState.currentPos.y);
    const w = Math.abs(dragState.currentPos.x - dragState.startPos.x);
    const h = Math.abs(dragState.currentPos.y - dragState.startPos.y);

    const selRect = { x, y, width: w, height: h };

    store.interaction.selectedNodes.clear();

    for (const node of Object.values(graph.nodes)) {
        const nodeType = store.graph.getNodeType(node.type);
        const bounds = getNodeBounds(node, nodeType);
        if (rectsOverlap(selRect, bounds)) {
            store.interaction.selectedNodes.add(node.id);
        }
    }
}
