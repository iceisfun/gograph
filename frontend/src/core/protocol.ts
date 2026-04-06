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

export const NODE_CONTENT = 'node.content';
export const CONNECTION_STATE = 'connection.state';

// Base fields shared by all slot types
interface BaseSlot {
    type: string;
    color?: string;
    animate?: string;   // flash|pulse|none
    duration?: number;  // animation ms
}

export interface TextSlot extends BaseSlot {
    type: 'text';
    text: string;
    size?: number;
    align?: string;
    font?: string;
}

export interface ProgressSlot extends BaseSlot {
    type: 'progress';
    value: number;      // 0.0..1.0
}

export interface LedSlot extends BaseSlot {
    type: 'led';
    states: boolean[];
}

export interface SpinnerSlot extends BaseSlot {
    type: 'spinner';
    visible: boolean;
}

export interface BadgeSlot extends BaseSlot {
    type: 'badge';
    text?: string;
    background?: string;
}

export interface SparklineSlot extends BaseSlot {
    type: 'sparkline';
    values: number[];
    min?: number;
    max?: number;
}

export interface ImageSlot extends BaseSlot {
    type: 'image';
    src: string;
    width?: number;
    height?: number;
}

export interface SvgSlot extends BaseSlot {
    type: 'svg';
    markup: string;
    width?: number;
    height?: number;
}

export type ContentSlot = TextSlot | ProgressSlot | LedSlot | SpinnerSlot | BadgeSlot | SparklineSlot | ImageSlot | SvgSlot;

export interface NodeContentPayload extends Envelope {
    nodeID: string;
    text?: string;
    image?: string;
    slots?: Record<string, ContentSlot>;
}

export interface ConnectionStatePayload extends Envelope {
    connectionID: string;
    active: boolean;
    value?: string;
}
