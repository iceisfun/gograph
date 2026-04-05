import type { Vec2 } from '../core/geometry.js';
import { vec2 } from '../core/geometry.js';
import type { CameraState } from '../state/camera-state.js';
import type { DragPanState } from '../state/interaction-state.js';

export function handleWheel(
    screenPos: Vec2,
    deltaY: number,
    camera: CameraState,
): void {
    const factor = deltaY < 0 ? 1.1 : 0.9;
    camera.zoomAt(screenPos, factor);
}

export function startPan(screenPos: Vec2): DragPanState {
    return { type: 'pan', startPos: vec2(screenPos.x, screenPos.y) };
}

export function updatePan(
    screenPos: Vec2,
    camera: CameraState,
    dragState: DragPanState,
): void {
    const dx = screenPos.x - dragState.startPos.x;
    const dy = screenPos.y - dragState.startPos.y;
    camera.panBy(vec2(dx, dy));
    dragState.startPos = vec2(screenPos.x, screenPos.y);
}
