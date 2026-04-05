import type { Vec2 } from '../core/geometry.js';
import { distance } from '../core/geometry.js';
import type { AppStore } from '../state/store.js';
import { getNodeBounds } from '../render/nodes.js';
import { rectContains } from '../core/geometry.js';
import { computeControlPoints, bezierPoint } from '../core/bezier.js';
import { SLOT_RADIUS } from '../core/constants.js';

/**
 * Hit-test nodes. Returns the node ID under the point, or null.
 */
export function hitTestNode(worldPos: Vec2, store: AppStore): string | null {
    const graph = store.graph.current;
    if (!graph) return null;

    // Iterate in reverse for z-order (last drawn = on top)
    const nodes = Object.values(graph.nodes);
    for (let i = nodes.length - 1; i >= 0; i--) {
        const node = nodes[i];
        const bounds = store.graph.getCachedNodeBounds(node.id)
            ?? getNodeBounds(node, store.graph.getNodeType(node.type));
        if (rectContains(bounds, worldPos)) {
            return node.id;
        }
    }
    return null;
}

/**
 * Hit-test slots. Returns the node and slot IDs, or null.
 */
export function hitTestSlot(
    worldPos: Vec2,
    store: AppStore,
): { nodeId: string; slotId: string } | null {
    const graph = store.graph.current;
    if (!graph) return null;

    const hitRadius = SLOT_RADIUS + 4; // slightly larger target

    for (const node of Object.values(graph.nodes)) {
        const nodeType = store.graph.getNodeType(node.type);
        if (!nodeType) continue;

        for (const slot of nodeType.slots) {
            const pos = store.graph.getSlotPosition(node.id, slot.id);
            if (pos.x === 0 && pos.y === 0) continue;
            if (distance(worldPos, pos) <= hitRadius) {
                return { nodeId: node.id, slotId: slot.id };
            }
        }
    }

    return null;
}

/**
 * Hit-test connections using distance to Bezier curve.
 */
export function hitTestConnection(worldPos: Vec2, store: AppStore): string | null {
    const graph = store.graph.current;
    if (!graph) return null;

    const threshold = 8;

    for (const conn of graph.connections) {
        const from = store.graph.getSlotPosition(conn.fromNode, conn.fromSlot);
        const to = store.graph.getSlotPosition(conn.toNode, conn.toSlot);
        const fromDir = store.graph.getSlotDirection(conn.fromNode, conn.fromSlot);
        const toDir = store.graph.getSlotDirection(conn.toNode, conn.toSlot);
        const [cp1, cp2] = computeControlPoints(from, to, fromDir, toDir);

        const dist = distanceToBezier(worldPos, from, cp1, cp2, to);
        if (dist <= threshold) {
            return conn.id;
        }
    }

    return null;
}

/**
 * Approximate minimum distance from a point to a cubic Bezier curve by sampling.
 */
function distanceToBezier(
    point: Vec2,
    p0: Vec2,
    p1: Vec2,
    p2: Vec2,
    p3: Vec2,
): number {
    const samples = 32;
    let minDist = Infinity;

    for (let i = 0; i <= samples; i++) {
        const t = i / samples;
        const bp = bezierPoint(p0, p1, p2, p3, t);
        const d = distance(point, bp);
        if (d < minDist) {
            minDist = d;
        }
    }

    return minDist;
}
