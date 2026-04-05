import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import { getNodeBounds } from './nodes.js';

const HANDLE_HEIGHT = 22;

export function drawOverlays(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
): void {
    const drag = store.interaction.dragState;

    // Selection rectangle (while dragging to select)
    if (drag && drag.type === 'select') {
        const x = Math.min(drag.startPos.x, drag.currentPos.x);
        const y = Math.min(drag.startPos.y, drag.currentPos.y);
        const w = Math.abs(drag.currentPos.x - drag.startPos.x);
        const h = Math.abs(drag.currentPos.y - drag.startPos.y);

        ctx.fillStyle = theme.selectionFill;
        ctx.fillRect(x, y, w, h);

        ctx.strokeStyle = theme.selectionStroke;
        ctx.lineWidth = theme.selectionStrokeWidth;
        ctx.setLineDash(theme.selectionDash);
        ctx.strokeRect(x, y, w, h);
        ctx.setLineDash([]);
    }

    // Persistent selection bounding box with drag handle
    const selected = store.interaction.selectedNodes;
    if (selected.size > 1) {
        const graph = store.graph.current;
        if (!graph) return;

        // Compute bounding box of all selected nodes
        let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
        for (const nodeId of selected) {
            const bounds = store.graph.getCachedNodeBounds(nodeId)
                ?? getNodeBounds(graph.nodes[nodeId], store.graph.getNodeType(graph.nodes[nodeId]?.type));
            if (!bounds) continue;
            minX = Math.min(minX, bounds.x);
            minY = Math.min(minY, bounds.y);
            maxX = Math.max(maxX, bounds.x + bounds.width);
            maxY = Math.max(maxY, bounds.y + bounds.height);
        }

        if (!isFinite(minX)) return;

        const pad = 12;
        const bx = minX - pad;
        const by = minY - pad - HANDLE_HEIGHT;
        const bw = maxX - minX + pad * 2;
        const bh = maxY - minY + pad * 2 + HANDLE_HEIGHT;

        // Draw bounding region
        ctx.fillStyle = theme.selectionFill;
        ctx.fillRect(bx, by + HANDLE_HEIGHT, bw, bh - HANDLE_HEIGHT);

        ctx.strokeStyle = theme.selectionStroke;
        ctx.lineWidth = theme.selectionStrokeWidth;
        ctx.setLineDash(theme.selectionDash);
        ctx.strokeRect(bx, by + HANDLE_HEIGHT, bw, bh - HANDLE_HEIGHT);
        ctx.setLineDash([]);

        // Draw handle bar
        const handleR = 4;
        ctx.beginPath();
        ctx.moveTo(bx + handleR, by);
        ctx.lineTo(bx + bw - handleR, by);
        ctx.arcTo(bx + bw, by, bx + bw, by + handleR, handleR);
        ctx.lineTo(bx + bw, by + HANDLE_HEIGHT);
        ctx.lineTo(bx, by + HANDLE_HEIGHT);
        ctx.lineTo(bx, by + handleR);
        ctx.arcTo(bx, by, bx + handleR, by, handleR);
        ctx.closePath();

        ctx.fillStyle = 'rgba(233, 69, 96, 0.25)';
        ctx.fill();
        ctx.strokeStyle = theme.selectionStroke;
        ctx.lineWidth = 1;
        ctx.setLineDash([]);
        ctx.stroke();

        // Handle text
        ctx.fillStyle = theme.selectionStroke;
        ctx.font = '11px sans-serif';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(
            `${selected.size} Nodes Selected`,
            bx + bw / 2,
            by + HANDLE_HEIGHT / 2,
        );
    }
}

/**
 * Hit-test the selection group handle. Returns true if the point is
 * within the handle bar of the selection bounding box.
 */
export function hitTestGroupHandle(
    worldPos: { x: number; y: number },
    store: AppStore,
): boolean {
    const selected = store.interaction.selectedNodes;
    if (selected.size <= 1) return false;

    const graph = store.graph.current;
    if (!graph) return false;

    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    for (const nodeId of selected) {
        const bounds = store.graph.getCachedNodeBounds(nodeId)
            ?? getNodeBounds(graph.nodes[nodeId], store.graph.getNodeType(graph.nodes[nodeId]?.type));
        if (!bounds) continue;
        minX = Math.min(minX, bounds.x);
        minY = Math.min(minY, bounds.y);
        maxX = Math.max(maxX, bounds.x + bounds.width);
        maxY = Math.max(maxY, bounds.y + bounds.height);
    }

    if (!isFinite(minX)) return false;

    const pad = 12;
    const bx = minX - pad;
    const by = minY - pad - HANDLE_HEIGHT;
    const bw = maxX - minX + pad * 2;

    return worldPos.x >= bx && worldPos.x <= bx + bw &&
           worldPos.y >= by && worldPos.y <= by + HANDLE_HEIGHT;
}
