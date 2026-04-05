import type {
    EventStartPayload,
    EventUpdatePayload,
    EventEndPayload,
    EventCancelPayload,
    GraphUpdatePayload,
    NodeUpdatePayload,
    NodeActivePayload,
    ConnectionUpdatePayload,
    NodeContentPayload,
    ConnectionStatePayload,
} from '../core/protocol.js';
import {
    EVENT_START,
    EVENT_UPDATE,
    EVENT_END,
    EVENT_CANCEL,
    GRAPH_UPDATE,
    NODE_UPDATE,
    NODE_ACTIVE,
    CONNECTION_UPDATE,
    NODE_CONTENT,
    CONNECTION_STATE,
} from '../core/protocol.js';

export interface SSEHandlers {
    onEventStart(payload: EventStartPayload): void;
    onEventUpdate(payload: EventUpdatePayload): void;
    onEventEnd(payload: EventEndPayload): void;
    onEventCancel(payload: EventCancelPayload): void;
    onGraphUpdate(payload: GraphUpdatePayload): void;
    onNodeUpdate(payload: NodeUpdatePayload): void;
    onNodeActive?(payload: NodeActivePayload): void;
    onConnectionUpdate(payload: ConnectionUpdatePayload): void;
    onNodeContent?(payload: NodeContentPayload): void;
    onConnectionState?(payload: ConnectionStatePayload): void;
    onConnect?(): void;
    onDisconnect?(): void;
}

export class SSEClient {
    private graphID: string;
    private baseUrl: string;
    private handlers: SSEHandlers;
    private eventSource: EventSource | null = null;
    private reconnectDelay = 1000;
    private maxReconnectDelay = 30000;
    private shouldReconnect = false;

    constructor(graphID: string, baseUrl: string, handlers: SSEHandlers) {
        this.graphID = graphID;
        this.baseUrl = baseUrl;
        this.handlers = handlers;
    }

    connect(): void {
        this.shouldReconnect = true;
        this.doConnect();
    }

    disconnect(): void {
        this.shouldReconnect = false;
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
    }

    private doConnect(): void {
        if (this.eventSource) {
            this.eventSource.close();
        }

        const url = `${this.baseUrl}/graphs/${this.graphID}/events`;
        this.eventSource = new EventSource(url);

        this.eventSource.onopen = () => {
            this.reconnectDelay = 1000;
            this.handlers.onConnect?.();
        };

        this.eventSource.onerror = () => {
            this.handlers.onDisconnect?.();
            this.eventSource?.close();
            this.eventSource = null;

            if (this.shouldReconnect) {
                setTimeout(() => this.doConnect(), this.reconnectDelay);
                this.reconnectDelay = Math.min(
                    this.reconnectDelay * 2,
                    this.maxReconnectDelay,
                );
            }
        };

        // Register typed event listeners
        this.eventSource.addEventListener(EVENT_START, (e) => {
            this.handlers.onEventStart(JSON.parse((e as MessageEvent).data) as EventStartPayload);
        });

        this.eventSource.addEventListener(EVENT_UPDATE, (e) => {
            this.handlers.onEventUpdate(JSON.parse((e as MessageEvent).data) as EventUpdatePayload);
        });

        this.eventSource.addEventListener(EVENT_END, (e) => {
            this.handlers.onEventEnd(JSON.parse((e as MessageEvent).data) as EventEndPayload);
        });

        this.eventSource.addEventListener(EVENT_CANCEL, (e) => {
            this.handlers.onEventCancel(JSON.parse((e as MessageEvent).data) as EventCancelPayload);
        });

        this.eventSource.addEventListener(GRAPH_UPDATE, (e) => {
            this.handlers.onGraphUpdate(JSON.parse((e as MessageEvent).data) as GraphUpdatePayload);
        });

        this.eventSource.addEventListener(NODE_UPDATE, (e) => {
            this.handlers.onNodeUpdate(JSON.parse((e as MessageEvent).data) as NodeUpdatePayload);
        });

        this.eventSource.addEventListener(NODE_ACTIVE, (e) => {
            this.handlers.onNodeActive?.(JSON.parse((e as MessageEvent).data) as NodeActivePayload);
        });

        this.eventSource.addEventListener(CONNECTION_UPDATE, (e) => {
            this.handlers.onConnectionUpdate(JSON.parse((e as MessageEvent).data) as ConnectionUpdatePayload);
        });

        this.eventSource.addEventListener(NODE_CONTENT, (e) => {
            this.handlers.onNodeContent?.(JSON.parse((e as MessageEvent).data) as NodeContentPayload);
        });

        this.eventSource.addEventListener(CONNECTION_STATE, (e) => {
            this.handlers.onConnectionState?.(JSON.parse((e as MessageEvent).data) as ConnectionStatePayload);
        });
    }
}
