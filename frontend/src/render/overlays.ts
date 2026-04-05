import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';

export function drawOverlays(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
): void {
    const drag = store.interaction.dragState;

    // Selection rectangle
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
}
