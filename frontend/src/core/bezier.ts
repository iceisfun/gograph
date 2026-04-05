import { type Vec2, vec2, lerp } from './geometry.js';
import { CONNECTION_CURVE_OFFSET } from './constants.js';

/**
 * Compute cubic Bezier control points for a connection between two points.
 * Direction vectors control which way the curve leaves/enters each endpoint.
 * Defaults preserve backward compatibility (right-to-left horizontal).
 */
export function computeControlPoints(
    from: Vec2,
    to: Vec2,
    fromDir: Vec2 = { x: 1, y: 0 },
    toDir: Vec2 = { x: -1, y: 0 },
): [Vec2, Vec2] {
    const dx = Math.abs(to.x - from.x);
    const dy = Math.abs(to.y - from.y);
    const dist = Math.max(dx, dy);
    const offset = Math.max(dist * CONNECTION_CURVE_OFFSET, 50);
    const cp1 = vec2(from.x + fromDir.x * offset, from.y + fromDir.y * offset);
    const cp2 = vec2(to.x + toDir.x * offset, to.y + toDir.y * offset);
    return [cp1, cp2];
}

/**
 * Evaluate a point on a cubic Bezier curve using De Casteljau's algorithm.
 */
export function bezierPoint(p0: Vec2, p1: Vec2, p2: Vec2, p3: Vec2, t: number): Vec2 {
    const a = lerp(p0, p1, t);
    const b = lerp(p1, p2, t);
    const c = lerp(p2, p3, t);
    const d = lerp(a, b, t);
    const e = lerp(b, c, t);
    return lerp(d, e, t);
}

/**
 * A cached Bezier path that stores endpoints and control points.
 */
export class BezierPath {
    from: Vec2;
    to: Vec2;
    cp1: Vec2;
    cp2: Vec2;
    private _dirty = true;

    constructor(from: Vec2, to: Vec2, fromDir?: Vec2, toDir?: Vec2) {
        this.from = from;
        this.to = to;
        const [cp1, cp2] = computeControlPoints(from, to, fromDir, toDir);
        this.cp1 = cp1;
        this.cp2 = cp2;
    }

    pointAt(t: number): Vec2 {
        return bezierPoint(this.from, this.cp1, this.cp2, this.to, t);
    }

    invalidate(): void {
        this._dirty = true;
    }

    update(from: Vec2, to: Vec2, fromDir?: Vec2, toDir?: Vec2): void {
        this.from = from;
        this.to = to;
        const [cp1, cp2] = computeControlPoints(from, to, fromDir, toDir);
        this.cp1 = cp1;
        this.cp2 = cp2;
        this._dirty = false;
    }

    get dirty(): boolean {
        return this._dirty;
    }

    draw(ctx: CanvasRenderingContext2D): void {
        ctx.beginPath();
        ctx.moveTo(this.from.x, this.from.y);
        ctx.bezierCurveTo(
            this.cp1.x, this.cp1.y,
            this.cp2.x, this.cp2.y,
            this.to.x, this.to.y,
        );
    }
}
