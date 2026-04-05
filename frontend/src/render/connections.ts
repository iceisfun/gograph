import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import { computeControlPoints } from '../core/bezier.js';

export function drawConnections(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
): void {
    const graph = store.graph.current;
    if (!graph) return;

    for (const conn of graph.connections) {
        const from = store.graph.getSlotPosition(conn.fromNode, conn.fromSlot);
        const to = store.graph.getSlotPosition(conn.toNode, conn.toSlot);
        const [cp1, cp2] = computeControlPoints(from, to);

        const isSelected = store.interaction.selectedConnections.has(conn.id);
        const isHovered = store.interaction.hoveredConnection === conn.id;

        ctx.beginPath();
        ctx.moveTo(from.x, from.y);
        ctx.bezierCurveTo(cp1.x, cp1.y, cp2.x, cp2.y, to.x, to.y);

        if (isSelected) {
            ctx.strokeStyle = theme.connectionSelectedStroke;
            ctx.lineWidth = theme.connectionSelectedStrokeWidth;
        } else if (isHovered) {
            ctx.strokeStyle = theme.connectionHoverStroke;
            ctx.lineWidth = theme.connectionStrokeWidth + 1;
        } else {
            ctx.strokeStyle = theme.connectionStroke;
            ctx.lineWidth = theme.connectionStrokeWidth;
        }

        ctx.stroke();
    }

    // Draw in-progress connection preview
    const drag = store.interaction.dragState;
    if (drag && drag.type === 'connection') {
        const from = store.graph.getSlotPosition(drag.fromNode, drag.fromSlot);
        const to = drag.currentPos;
        const [cp1, cp2] = computeControlPoints(from, to);

        ctx.beginPath();
        ctx.moveTo(from.x, from.y);
        ctx.bezierCurveTo(cp1.x, cp1.y, cp2.x, cp2.y, to.x, to.y);
        ctx.strokeStyle = theme.connectionPreviewStroke;
        ctx.lineWidth = theme.connectionStrokeWidth;
        ctx.setLineDash(theme.connectionPreviewDash);
        ctx.stroke();
        ctx.setLineDash([]);
    }
}
