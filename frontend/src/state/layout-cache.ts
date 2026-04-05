import type { Vec2, Rect } from '../core/geometry.js';
import { vec2 } from '../core/geometry.js';
import type { Graph, NodeType } from '../core/types.js';
import {
    NODE_WIDTH,
    NODE_TITLE_HEIGHT,
    SLOT_SPACING,
    MIN_NODE_HEIGHT,
} from '../core/constants.js';

export type Side = 'left' | 'right' | 'bottom';

export interface SlotLayout {
    position: Vec2;
    direction: Vec2;  // outward unit vector
    side: Side;
    slotId: string;
}

export interface NodeLayout {
    bounds: Rect;
    slots: Map<string, SlotLayout>;
}

const DIR_LEFT: Vec2 = { x: -1, y: 0 };
const DIR_RIGHT: Vec2 = { x: 1, y: 0 };
const DIR_BOTTOM: Vec2 = { x: 0, y: 1 };

export function sideDirection(side: Side): Vec2 {
    switch (side) {
        case 'left': return DIR_LEFT;
        case 'right': return DIR_RIGHT;
        case 'bottom': return DIR_BOTTOM;
    }
}

/**
 * Determine which side of a node a peer is on.
 * Never returns 'top' to avoid title bar conflicts.
 */
function determineSide(nodeCenter: Vec2, peerCenter: Vec2, defaultSide: Side): Side {
    const dx = peerCenter.x - nodeCenter.x;
    const dy = peerCenter.y - nodeCenter.y;

    // If peer is very close (< 1px), keep default
    if (Math.abs(dx) < 1 && Math.abs(dy) < 1) return defaultSide;

    const projLeft = -dx;
    const projRight = dx;
    const projBottom = dy;

    const best = Math.max(projLeft, projRight, projBottom);

    // Only go bottom if peer is clearly below (projBottom > 0 and is the max)
    if (best === projBottom && projBottom > 0) return 'bottom';
    if (best === projLeft) return 'left';
    return 'right';
}

/**
 * Compute the full layout for all nodes in the graph.
 * Call this once per render frame. All slot positions, sides, directions,
 * and node bounds are cached in the returned map.
 */
export function computeLayout(
    graph: Graph | null,
    nodeTypeMap: Map<string, NodeType>,
): Map<string, NodeLayout> {
    const result = new Map<string, NodeLayout>();
    if (!graph) return result;

    const nodes = graph.nodes;
    const connections = graph.connections;

    // Pre-compute node centers for side determination
    const nodeCenters = new Map<string, Vec2>();
    for (const [id, node] of Object.entries(nodes)) {
        const nt = nodeTypeMap.get(node.type);
        const inputs = nt ? nt.slots.filter(s => s.direction === 'input').length : 0;
        const outputs = nt ? nt.slots.filter(s => s.direction === 'output').length : 0;
        const slotCount = Math.max(inputs, outputs, 1);
        const height = Math.max(MIN_NODE_HEIGHT, NODE_TITLE_HEIGHT + SLOT_SPACING * slotCount);
        nodeCenters.set(id, vec2(
            node.position.x + NODE_WIDTH / 2,
            node.position.y + height / 2,
        ));
    }

    // Build connection lookup: for each (nodeId, slotId) -> list of peer node IDs
    const slotPeers = new Map<string, string[]>(); // key: "nodeId:slotId" -> peer node IDs
    for (const conn of connections) {
        const fromKey = `${conn.fromNode}:${conn.fromSlot}`;
        const toKey = `${conn.toNode}:${conn.toSlot}`;
        if (!slotPeers.has(fromKey)) slotPeers.set(fromKey, []);
        if (!slotPeers.has(toKey)) slotPeers.set(toKey, []);
        slotPeers.get(fromKey)!.push(conn.toNode);
        slotPeers.get(toKey)!.push(conn.fromNode);
    }

    // Compute layout for each node
    for (const [nodeId, node] of Object.entries(nodes)) {
        const nt = nodeTypeMap.get(node.type);
        if (!nt) {
            // No type info - create minimal layout
            result.set(nodeId, {
                bounds: { x: node.position.x, y: node.position.y, width: NODE_WIDTH, height: MIN_NODE_HEIGHT },
                slots: new Map(),
            });
            continue;
        }

        const nodeCenter = nodeCenters.get(nodeId)!;

        // Determine side for each slot
        interface SlotAssignment {
            slotId: string;
            name: string;
            direction: 'input' | 'output';
            side: Side;
        }

        const assignments: SlotAssignment[] = [];

        for (const slot of nt.slots) {
            const defaultSide: Side = slot.direction === 'input' ? 'left' : 'right';
            const key = `${nodeId}:${slot.id}`;
            const peers = slotPeers.get(key);

            if (!peers || peers.length === 0) {
                // Unconnected: use default
                assignments.push({ slotId: slot.id, name: slot.name, direction: slot.direction, side: defaultSide });
                continue;
            }

            // Compute centroid of peer node centers
            let cx = 0, cy = 0;
            let count = 0;
            for (const peerId of peers) {
                const pc = nodeCenters.get(peerId);
                if (pc) { cx += pc.x; cy += pc.y; count++; }
            }

            if (count === 0) {
                assignments.push({ slotId: slot.id, name: slot.name, direction: slot.direction, side: defaultSide });
                continue;
            }

            const centroid = vec2(cx / count, cy / count);
            const side = determineSide(nodeCenter, centroid, defaultSide);
            assignments.push({ slotId: slot.id, name: slot.name, direction: slot.direction, side });
        }

        // Group slots by side
        const bySide: Record<Side, SlotAssignment[]> = { left: [], right: [], bottom: [] };
        for (const a of assignments) {
            bySide[a.side].push(a);
        }

        // Compute node bounds based on left/right slot counts
        const verticalSlotCount = Math.max(bySide.left.length, bySide.right.length, 1);
        const bodyHeight = SLOT_SPACING * verticalSlotCount;
        const height = Math.max(MIN_NODE_HEIGHT, NODE_TITLE_HEIGHT + bodyHeight);
        const bounds: Rect = { x: node.position.x, y: node.position.y, width: NODE_WIDTH, height };

        // Position slots on each side
        const slots = new Map<string, SlotLayout>();

        // Left side
        for (let i = 0; i < bySide.left.length; i++) {
            const a = bySide.left[i];
            slots.set(a.slotId, {
                position: vec2(node.position.x, node.position.y + NODE_TITLE_HEIGHT + SLOT_SPACING * (i + 0.5)),
                direction: DIR_LEFT,
                side: 'left',
                slotId: a.slotId,
            });
        }

        // Right side
        for (let i = 0; i < bySide.right.length; i++) {
            const a = bySide.right[i];
            slots.set(a.slotId, {
                position: vec2(node.position.x + NODE_WIDTH, node.position.y + NODE_TITLE_HEIGHT + SLOT_SPACING * (i + 0.5)),
                direction: DIR_RIGHT,
                side: 'right',
                slotId: a.slotId,
            });
        }

        // Bottom side -- evenly spaced horizontally
        if (bySide.bottom.length > 0) {
            const spacing = NODE_WIDTH / (bySide.bottom.length + 1);
            for (let i = 0; i < bySide.bottom.length; i++) {
                const a = bySide.bottom[i];
                slots.set(a.slotId, {
                    position: vec2(node.position.x + spacing * (i + 1), node.position.y + height),
                    direction: DIR_BOTTOM,
                    side: 'bottom',
                    slotId: a.slotId,
                });
            }
        }

        result.set(nodeId, { bounds, slots });
    }

    return result;
}

