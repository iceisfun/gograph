import type { Graph, Node, Connection, NodeType } from '../core/types.js';
import type { Vec2, Rect } from '../core/geometry.js';
import { vec2 } from '../core/geometry.js';
import { computeLayout, type NodeLayout, type SlotLayout, type Side } from './layout-cache.js';

export class GraphState {
    current: Graph | null = null;
    private _nodeTypes: NodeType[] = [];
    private _nodeTypeMap: Map<string, NodeType> = new Map();
    private _layoutCache: Map<string, NodeLayout> = new Map();
    nodeContent: Map<string, { text?: string; image?: string }> = new Map();

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

    setNodeContent(nodeId: string, content: { text?: string; image?: string }): void {
        this.nodeContent.set(nodeId, content);
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
     * Recompute the layout cache for all nodes. Call once per render frame.
     */
    computeLayout(): void {
        this._layoutCache = computeLayout(this.current, this._nodeTypeMap);
    }

    /**
     * Compute the world-space position of a slot on a node.
     * Reads from the layout cache (must call computeLayout() first).
     */
    getSlotPosition(nodeId: string, slotId: string): Vec2 {
        const nodeLayout = this._layoutCache.get(nodeId);
        if (!nodeLayout) return vec2(0, 0);
        const slotLayout = nodeLayout.slots.get(slotId);
        if (!slotLayout) return vec2(0, 0);
        return slotLayout.position;
    }

    /**
     * Get the outward direction vector for a slot.
     */
    getSlotDirection(nodeId: string, slotId: string): Vec2 {
        const nodeLayout = this._layoutCache.get(nodeId);
        if (!nodeLayout) return vec2(1, 0);
        const slotLayout = nodeLayout.slots.get(slotId);
        if (!slotLayout) return vec2(1, 0);
        return slotLayout.direction;
    }

    /**
     * Get which side a slot is on.
     */
    getSlotSide(nodeId: string, slotId: string): Side | null {
        const nodeLayout = this._layoutCache.get(nodeId);
        if (!nodeLayout) return null;
        const slotLayout = nodeLayout.slots.get(slotId);
        if (!slotLayout) return null;
        return slotLayout.side;
    }

    /**
     * Get the cached node bounds from the layout cache.
     */
    getCachedNodeBounds(nodeId: string): Rect | null {
        const nodeLayout = this._layoutCache.get(nodeId);
        if (!nodeLayout) return null;
        return nodeLayout.bounds;
    }

    /**
     * Get all slot layouts for a node (for the renderer).
     */
    getSlotLayouts(nodeId: string): Map<string, SlotLayout> | null {
        const nodeLayout = this._layoutCache.get(nodeId);
        if (!nodeLayout) return null;
        return nodeLayout.slots;
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
