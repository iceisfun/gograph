export interface Graph {
    id: string;
    version: number;
    nodes: Record<string, Node>;
    connections: Connection[];
    metadata?: Record<string, string>;
}

export interface Node {
    id: string;
    type: string;
    label: string;
    position: Position;
    config?: Record<string, string>;
    content?: Record<string, import('../core/protocol.js').ContentSlot>;
}

export interface Position {
    x: number;
    y: number;
}

export interface Slot {
    id: string;
    name: string;
    direction: 'input' | 'output';
    dataType: string;
}

export interface Connection {
    id: string;
    kind?: 'event' | 'state';
    fromNode: string;
    fromSlot: string;
    toNode: string;
    toSlot: string;
    config?: Record<string, string>;
    duration?: number;
    stateDataType?: string;
}

export interface NodeType {
    name: string;
    label: string;
    slots: Slot[];
    scriptName?: string;
    category?: string;
    contentHeight?: number;
    interactive?: boolean;
}
