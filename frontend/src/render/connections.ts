import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import { computeControlPoints, bezierPoint, bezierTangent } from '../core/bezier.js';
import { distance } from '../core/geometry.js';

export function drawConnections(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
    now: number = 0,
): void {
    const graph = store.graph.current;
    if (!graph) return;

    for (const conn of graph.connections) {
        const from = store.graph.getSlotPosition(conn.fromNode, conn.fromSlot);
        const to = store.graph.getSlotPosition(conn.toNode, conn.toSlot);
        const fromDir = store.graph.getSlotDirection(conn.fromNode, conn.fromSlot);
        const toDir = store.graph.getSlotDirection(conn.toNode, conn.toSlot);
        const [cp1, cp2] = computeControlPoints(from, to, fromDir, toDir);

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

        // Duration capsule at midpoint for timed connections
        const duration = conn.config?.duration;
        const connDist = distance(from, to);
        const capsuleMode = theme.connectionCapsuleVisibility;
        const capsuleVisible = duration && parseInt(duration) > 0
            && connDist >= theme.connectionCapsuleMinDistance
            && (capsuleMode === 'always'
                || (capsuleMode === 'hover' && (isHovered || isSelected))
                || (capsuleMode === 'related' && (isHovered || isSelected
                    || store.interaction.hoveredNode === conn.fromNode
                    || store.interaction.hoveredNode === conn.toNode
                    || store.interaction.selectedNodes.has(conn.fromNode)
                    || store.interaction.selectedNodes.has(conn.toNode))));

        if (capsuleVisible) {
            const mid = bezierPoint(from, cp1, cp2, to, 0.5);
            const tan = bezierTangent(from, cp1, cp2, to, 0.5);
            const angle = Math.atan2(tan.y, tan.x);

            const label = `${duration}ms`;
            ctx.save();
            ctx.font = theme.connectionCapsuleFont;
            const textW = ctx.measureText(label).width;
            const padX = 6;
            const padY = 3;
            const capsuleW = textW + padX * 2;
            const capsuleH = 14 + padY * 2;
            const r = capsuleH / 2;

            ctx.translate(mid.x, mid.y);
            ctx.rotate(angle);

            // Draw capsule shape (rounded rect centered at origin)
            ctx.beginPath();
            ctx.moveTo(-capsuleW / 2 + r, -capsuleH / 2);
            ctx.lineTo(capsuleW / 2 - r, -capsuleH / 2);
            ctx.arc(capsuleW / 2 - r, 0, r, -Math.PI / 2, Math.PI / 2);
            ctx.lineTo(-capsuleW / 2 + r, capsuleH / 2);
            ctx.arc(-capsuleW / 2 + r, 0, r, Math.PI / 2, -Math.PI / 2);
            ctx.closePath();

            ctx.fillStyle = theme.connectionCapsuleFill;
            ctx.fill();
            ctx.strokeStyle = theme.connectionCapsuleStroke;
            ctx.lineWidth = 1;
            ctx.stroke();

            // Draw text (flip if upside-down so text is always readable)
            const flipped = angle > Math.PI / 2 || angle < -Math.PI / 2;
            if (flipped) {
                ctx.rotate(Math.PI);
            }
            ctx.fillStyle = theme.connectionCapsuleText;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(label, 0, 0);

            ctx.restore();
        }

        // Active connection dashing animation
        const activeConn = store.animation.activeConnections.get(conn.id);
        if (activeConn) {
            ctx.save();
            ctx.beginPath();
            ctx.moveTo(from.x, from.y);
            ctx.bezierCurveTo(cp1.x, cp1.y, cp2.x, cp2.y, to.x, to.y);
            ctx.setLineDash(theme.connectionActiveDash);
            ctx.lineDashOffset = -((now - activeConn.startTime) / 1000) * theme.connectionActiveDashSpeed;
            ctx.strokeStyle = theme.connectionActiveStroke;
            ctx.lineWidth = theme.connectionStrokeWidth + 1;
            ctx.shadowBlur = theme.connectionActiveGlowRadius;
            ctx.shadowColor = theme.connectionActiveGlowColor;
            ctx.stroke();
            ctx.restore();
        }
    }

    // Draw in-progress connection preview
    const drag = store.interaction.dragState;
    if (drag && drag.type === 'connection') {
        const from = store.graph.getSlotPosition(drag.fromNode, drag.fromSlot);
        const fromDir = store.graph.getSlotDirection(drag.fromNode, drag.fromSlot);
        const to = drag.currentPos;
        // Guess toDir: opposite of fromDir for visual consistency
        const toDir = { x: -fromDir.x, y: -fromDir.y };
        const [cp1, cp2] = computeControlPoints(from, to, fromDir, toDir);

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
