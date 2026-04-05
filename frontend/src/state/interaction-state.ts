import type { Vec2 } from '../core/geometry.js';

export interface DragNodeState {
    type: 'node';
    nodeId: string;
    startPos: Vec2;
    offset: Vec2;
}

export interface DragConnectionState {
    type: 'connection';
    fromNode: string;
    fromSlot: string;
    currentPos: Vec2;
}

export interface DragPanState {
    type: 'pan';
    startPos: Vec2;
}

export interface DragSelectState {
    type: 'select';
    startPos: Vec2;
    currentPos: Vec2;
}

export interface DragGroupState {
    type: 'group';
    startPos: Vec2;
    nodeOffsets: Map<string, Vec2>;
}

export type DragState = DragNodeState | DragConnectionState | DragPanState | DragSelectState | DragGroupState;

export interface HoveredSlot {
    nodeId: string;
    slotId: string;
}

export class InteractionState {
    mode: 'edit' | 'view' = 'edit';
    selectedNodes: Set<string> = new Set();
    selectedConnections: Set<string> = new Set();
    hoveredNode: string | null = null;
    hoveredSlot: HoveredSlot | null = null;
    hoveredConnection: string | null = null;
    dragState: DragState | null = null;

    clearSelection(): void {
        this.selectedNodes.clear();
        this.selectedConnections.clear();
    }

    compatibleSlots: Set<string> = new Set();

    clearHover(): void {
        this.hoveredNode = null;
        this.hoveredSlot = null;
        this.hoveredConnection = null;
    }

    setCompatibleSlots(slots: Set<string>): void {
        this.compatibleSlots = slots;
    }

    clearCompatibleSlots(): void {
        this.compatibleSlots.clear();
    }
}
