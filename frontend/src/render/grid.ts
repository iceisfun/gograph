import type { CameraState } from '../state/camera-state.js';
import type { Theme } from '../themes/theme.js';
import { GRID_SIZE, GRID_MAJOR_EVERY } from '../core/constants.js';

export function drawGrid(
    ctx: CanvasRenderingContext2D,
    camera: CameraState,
    canvasWidth: number,
    canvasHeight: number,
    theme: Theme,
): void {
    // Compute visible world-space bounds
    const topLeft = camera.screenToWorld({ x: 0, y: 0 });
    const bottomRight = camera.screenToWorld({ x: canvasWidth, y: canvasHeight });

    const startX = Math.floor(topLeft.x / GRID_SIZE) * GRID_SIZE;
    const startY = Math.floor(topLeft.y / GRID_SIZE) * GRID_SIZE;
    const endX = Math.ceil(bottomRight.x / GRID_SIZE) * GRID_SIZE;
    const endY = Math.ceil(bottomRight.y / GRID_SIZE) * GRID_SIZE;

    // Draw minor grid lines
    ctx.beginPath();
    ctx.strokeStyle = theme.gridMinor;
    ctx.lineWidth = theme.gridMinorWidth / camera.zoom;

    for (let x = startX; x <= endX; x += GRID_SIZE) {
        const gridIndex = Math.round(x / GRID_SIZE);
        if (gridIndex % GRID_MAJOR_EVERY === 0) continue;
        ctx.moveTo(x, startY);
        ctx.lineTo(x, endY);
    }

    for (let y = startY; y <= endY; y += GRID_SIZE) {
        const gridIndex = Math.round(y / GRID_SIZE);
        if (gridIndex % GRID_MAJOR_EVERY === 0) continue;
        ctx.moveTo(startX, y);
        ctx.lineTo(endX, y);
    }

    ctx.stroke();

    // Draw major grid lines
    ctx.beginPath();
    ctx.strokeStyle = theme.gridMajor;
    ctx.lineWidth = theme.gridMajorWidth / camera.zoom;

    const majorSize = GRID_SIZE * GRID_MAJOR_EVERY;
    const majorStartX = Math.floor(topLeft.x / majorSize) * majorSize;
    const majorStartY = Math.floor(topLeft.y / majorSize) * majorSize;

    for (let x = majorStartX; x <= endX; x += majorSize) {
        ctx.moveTo(x, startY);
        ctx.lineTo(x, endY);
    }

    for (let y = majorStartY; y <= endY; y += majorSize) {
        ctx.moveTo(startX, y);
        ctx.lineTo(endX, y);
    }

    ctx.stroke();
}
