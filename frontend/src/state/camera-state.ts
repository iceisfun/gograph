import type { Vec2 } from '../core/geometry.js';
import { vec2 } from '../core/geometry.js';

export class CameraState {
    pan: Vec2 = vec2(0, 0);
    zoom = 1;

    private readonly minZoom = 0.1;
    private readonly maxZoom = 5;

    screenToWorld(screen: Vec2): Vec2 {
        return vec2(
            (screen.x - this.pan.x) / this.zoom,
            (screen.y - this.pan.y) / this.zoom,
        );
    }

    worldToScreen(world: Vec2): Vec2 {
        return vec2(
            world.x * this.zoom + this.pan.x,
            world.y * this.zoom + this.pan.y,
        );
    }

    applyTransform(ctx: CanvasRenderingContext2D): void {
        ctx.translate(this.pan.x, this.pan.y);
        ctx.scale(this.zoom, this.zoom);
    }

    panBy(delta: Vec2): void {
        this.pan = vec2(this.pan.x + delta.x, this.pan.y + delta.y);
    }

    zoomAt(center: Vec2, factor: number): void {
        const newZoom = Math.min(this.maxZoom, Math.max(this.minZoom, this.zoom * factor));
        const ratio = newZoom / this.zoom;

        // Adjust pan so the point under the cursor stays in place
        this.pan = vec2(
            center.x - (center.x - this.pan.x) * ratio,
            center.y - (center.y - this.pan.y) * ratio,
        );
        this.zoom = newZoom;
    }
}
