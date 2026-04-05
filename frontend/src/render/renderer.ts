import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import { drawGrid } from './grid.js';
import { drawConnections } from './connections.js';
import { drawConnectionEvents } from './connection-events.js';
import { drawNodes } from './nodes.js';
import { drawOverlays } from './overlays.js';

export class Renderer {
    private canvas: HTMLCanvasElement;
    private ctx: CanvasRenderingContext2D;
    private store: AppStore;
    private theme: Theme;
    private animFrameId: number | null = null;
    private _canvasWidth = 0;
    private _canvasHeight = 0;

    constructor(canvas: HTMLCanvasElement, store: AppStore, theme: Theme) {
        this.canvas = canvas;
        const ctx = canvas.getContext('2d');
        if (!ctx) throw new Error('Failed to get 2D rendering context');
        this.ctx = ctx;
        this.store = store;
        this.theme = theme;
    }

    start(): void {
        const loop = (): void => {
            this.render();
            this.animFrameId = requestAnimationFrame(loop);
        };
        this.animFrameId = requestAnimationFrame(loop);
    }

    stop(): void {
        if (this.animFrameId !== null) {
            cancelAnimationFrame(this.animFrameId);
            this.animFrameId = null;
        }
    }

    private handleResize(): void {
        const dpr = window.devicePixelRatio || 1;
        const displayWidth = this.canvas.clientWidth;
        const displayHeight = this.canvas.clientHeight;

        const targetWidth = Math.round(displayWidth * dpr);
        const targetHeight = Math.round(displayHeight * dpr);

        if (this.canvas.width !== targetWidth || this.canvas.height !== targetHeight) {
            this.canvas.width = targetWidth;
            this.canvas.height = targetHeight;
        }

        this._canvasWidth = targetWidth;
        this._canvasHeight = targetHeight;
    }

    private render(): void {
        const ctx = this.ctx;
        const store = this.store;
        const theme = this.theme;
        const now = performance.now();

        // 1. Resize canvas to match container (handle DPR)
        this.handleResize();

        // 1.5. Recompute layout cache for this frame
        store.graph.computeLayout();

        // 2. Clear
        ctx.clearRect(0, 0, this._canvasWidth, this._canvasHeight);

        // 3. Save state
        ctx.save();

        // Scale for DPR, then apply camera
        const dpr = window.devicePixelRatio || 1;
        ctx.scale(dpr, dpr);

        // 4. Apply camera transform
        store.camera.applyTransform(ctx);

        // Get logical canvas size for grid computation
        const logicalWidth = this.canvas.clientWidth;
        const logicalHeight = this.canvas.clientHeight;

        // 5. Draw grid
        drawGrid(ctx, store.camera, logicalWidth, logicalHeight, theme);

        // 6. Draw connections
        drawConnections(ctx, store, theme, now);

        // 7. Draw connection events (animated dots)
        drawConnectionEvents(ctx, store, theme);

        // 8. Draw nodes
        drawNodes(ctx, store, theme, now);

        // 9. Draw overlays (selection box, etc.)
        drawOverlays(ctx, store, theme);

        // 10. Restore
        ctx.restore();

        // 11. Update animation state
        store.animation.tick(now);
    }
}
