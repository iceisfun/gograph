import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import { computeControlPoints, bezierPoint } from '../core/bezier.js';
import { EVENT_DOT_RADIUS } from '../core/constants.js';

export function drawConnectionEvents(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
): void {
    for (const [_id, event] of store.animation.activeEvents) {
        const conn = store.graph.getConnection(event.connectionID);
        if (!conn) continue;

        const from = store.graph.getSlotPosition(conn.fromNode, conn.fromSlot);
        const to = store.graph.getSlotPosition(conn.toNode, conn.toSlot);
        const fromDir = store.graph.getSlotDirection(conn.fromNode, conn.fromSlot);
        const toDir = store.graph.getSlotDirection(conn.toNode, conn.toSlot);
        const [cp1, cp2] = computeControlPoints(from, to, fromDir, toDir);

        // Draw trail (several dots behind the main dot)
        const trailCount = 5;
        const trailSpacing = 0.03;
        for (let i = trailCount; i > 0; i--) {
            const trailT = event.progress - i * trailSpacing;
            if (trailT < 0) continue;

            const trailPos = bezierPoint(from, cp1, cp2, to, trailT);
            const trailOpacity = theme.eventTrailOpacity * (1 - i / (trailCount + 1)) * event.intensity;

            ctx.beginPath();
            ctx.arc(trailPos.x, trailPos.y, EVENT_DOT_RADIUS * 0.7, 0, Math.PI * 2);
            ctx.fillStyle = hexWithAlpha(event.color, trailOpacity);
            ctx.fill();
        }

        // Main dot position
        const pos = bezierPoint(from, cp1, cp2, to, event.progress);

        // Glow effect
        ctx.save();
        ctx.shadowBlur = theme.eventGlowRadius * event.intensity;
        ctx.shadowColor = event.color || theme.eventGlowColor;

        ctx.beginPath();
        ctx.arc(pos.x, pos.y, EVENT_DOT_RADIUS, 0, Math.PI * 2);
        ctx.fillStyle = event.color || theme.eventDotColor;
        ctx.fill();

        ctx.restore();
    }
}

function hexWithAlpha(hex: string, alpha: number): string {
    // Parse hex color and return rgba
    const r = parseInt(hex.slice(1, 3), 16) || 0;
    const g = parseInt(hex.slice(3, 5), 16) || 0;
    const b = parseInt(hex.slice(5, 7), 16) || 0;
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}
