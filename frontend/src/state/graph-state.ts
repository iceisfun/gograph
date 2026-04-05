import type { Graph, Node, Connection, NodeType } from '../core/types.js';
import type { Vec2 } from '../core/geometry.js';
import { vec2 } from '../core/geometry.js';
import {
    NODE_WIDTH,
    NODE_TITLE_HEIGHT,
    SLOT_SPACING,
    SLOT_OFFSET_X,
} from '../core/constants.js';

export class GraphState {
    current: Graph | null = null;
    private _nodeTypes: NodeType[] = [];
    private _nodeTypeMap: Map<string, NodeType> = new Map();

    setGraph(graph: Graph): void {
        this.current = graph;
    }

    setNodeTypes(types: NodeType[]): void {
        this._nodeTypes = types;
        this._nodeTypeMap.clear();
        for (const t of types) {
            this._nodeTypeMap.set(t.name, t);
        }
    }

    get nodeTypes(): NodeType[] {
        return this._nodeTypes;
    }

    getNodeType(name: string): NodeType | undefined {
        return this._nodeTypeMap.get(name);
    }

    updateNode(node: Node): void {
        if (!this.current) return;
        this.current.nodes[node.id] = node;
    }

    updateConnection(connection: Connection): void {
        if (!this.current) return;
        const idx = this.current.connections.findIndex(c => c.id === connection.id);
        if (idx >= 0) {
            this.current.connections[idx] = connection;
        } else {
            this.current.connections.push(connection);
        }
    }

    removeConnection(connectionId: string): void {
        if (!this.current) return;
        this.current.connections = this.current.connections.filter(c => c.id !== connectionId);
    }

    /**
     * Compute the world-space position of a slot on a node.
     * Input slots appear on the left edge, output slots on the right edge.
     */
    getSlotPosition(nodeId: string, slotId: string): Vec2 {
        if (!this.current) return vec2(0, 0);

        const node = this.current.nodes[nodeId];
        if (!node) return vec2(0, 0);

        const nodeType = this._nodeTypeMap.get(node.type);
        if (!nodeType) return vec2(0, 0);

        const inputs = nodeType.slots.filter(s => s.direction === 'input');
        const outputs = nodeType.slots.filter(s => s.direction === 'output');

        // Check inputs
        const inputIdx = inputs.findIndex(s => s.id === slotId);
        if (inputIdx >= 0) {
            return vec2(
                node.position.x + SLOT_OFFSET_X,
                node.position.y + NODE_TITLE_HEIGHT + SLOT_SPACING * (inputIdx + 0.5),
            );
        }

        // Check outputs
        const outputIdx = outputs.findIndex(s => s.id === slotId);
        if (outputIdx >= 0) {
            return vec2(
                node.position.x + NODE_WIDTH - SLOT_OFFSET_X,
                node.position.y + NODE_TITLE_HEIGHT + SLOT_SPACING * (outputIdx + 0.5),
            );
        }

        return vec2(0, 0);
    }

    /**
     * Get the connection object for a given connection ID.
     */
    getConnection(connectionId: string): Connection | undefined {
        if (!this.current) return undefined;
        return this.current.connections.find(c => c.id === connectionId);
    }

    /**
     * Check if a slot is connected to any connection.
     */
    isSlotConnected(nodeId: string, slotId: string): boolean {
        if (!this.current) return false;
        return this.current.connections.some(
            c => (c.fromNode === nodeId && c.fromSlot === slotId) ||
                 (c.toNode === nodeId && c.toSlot === slotId),
        );
    }
}
