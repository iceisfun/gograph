import type { Graph, Node, Connection } from './types.js';

// SSE event type constants
export const EVENT_START = 'event.start';
export const EVENT_UPDATE = 'event.update';
export const EVENT_END = 'event.end';
export const EVENT_CANCEL = 'event.cancel';
export const GRAPH_UPDATE = 'graph.update';
export const NODE_UPDATE = 'node.update';
export const NODE_ACTIVE = 'node.active';
export const CONNECTION_UPDATE = 'connection.update';

// Base envelope
export interface Envelope {
    v: number;
    ts: number;
}

export interface EventStartPayload extends Envelope {
    eventID: string;
    connectionID: string;
    color?: string;
    duration: number;
    metadata?: Record<string, string>;
}

export interface EventUpdatePayload extends Envelope {
    eventID: string;
    color?: string;
    intensity?: number;
    metadata?: Record<string, string>;
}

export interface EventEndPayload extends Envelope {
    eventID: string;
}

export interface EventCancelPayload extends Envelope {
    eventID: string;
    immediate: boolean;
}

export interface GraphUpdatePayload extends Envelope {
    graph: Graph;
}

export interface NodeUpdatePayload extends Envelope {
    node: Node;
}

export interface NodeActivePayload extends Envelope {
    nodeID: string;
    duration: number;
}

export interface ConnectionUpdatePayload extends Envelope {
    connection: Connection;
}
