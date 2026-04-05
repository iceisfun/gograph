import type { EventStartPayload, EventUpdatePayload } from '../core/protocol.js';
import { DEFAULT_EVENT_DURATION } from '../core/constants.js';

export interface ActiveEvent {
    eventID: string;
    connectionID: string;
    startTime: number;
    duration: number;
    color: string;
    intensity: number;
    progress: number;
}

export class AnimationState {
    activeEvents: Map<string, ActiveEvent> = new Map();
    activeNodes: Map<string, { startTime: number; endTime: number }> = new Map();
    activeConnections: Map<string, { startTime: number; endTime: number; color: string }> = new Map();

    activateNode(nodeId: string, durationMs: number): void {
        const now = performance.now();
        this.activeNodes.set(nodeId, { startTime: now, endTime: now + durationMs });
    }

    activateConnection(connectionId: string, durationMs: number, color: string): void {
        const now = performance.now();
        this.activeConnections.set(connectionId, { startTime: now, endTime: now + durationMs, color });
    }

    startEvent(payload: EventStartPayload): void {
        this.activeEvents.set(payload.eventID, {
            eventID: payload.eventID,
            connectionID: payload.connectionID,
            startTime: performance.now(),
            duration: payload.duration || DEFAULT_EVENT_DURATION,
            color: payload.color || '#e94560',
            intensity: 1,
            progress: 0,
        });
        this.activateConnection(payload.connectionID, payload.duration || DEFAULT_EVENT_DURATION, payload.color || '');
    }

    updateEvent(payload: EventUpdatePayload): void {
        const event = this.activeEvents.get(payload.eventID);
        if (!event) return;
        if (payload.color !== undefined) event.color = payload.color;
        if (payload.intensity !== undefined) event.intensity = payload.intensity;
    }

    endEvent(eventID: string): void {
        this.activeEvents.delete(eventID);
    }

    cancelEvent(eventID: string, immediate: boolean): void {
        if (immediate) {
            this.activeEvents.delete(eventID);
        } else {
            // Let it fade out on next tick
            const event = this.activeEvents.get(eventID);
            if (event) {
                event.intensity = 0;
            }
        }
    }

    tick(now: number): void {
        const toRemove: string[] = [];

        for (const [id, event] of this.activeEvents) {
            const elapsed = now - event.startTime;
            event.progress = Math.min(1, elapsed / event.duration);

            if (event.progress >= 1) {
                toRemove.push(id);
            }

            // Remove fading events that hit zero intensity
            if (event.intensity <= 0 && event.progress > 0) {
                toRemove.push(id);
            }
        }

        for (const id of toRemove) {
            this.activeEvents.delete(id);
        }

        // Clean up expired active nodes
        for (const [id, state] of this.activeNodes) {
            if (now >= state.endTime) this.activeNodes.delete(id);
        }
        for (const [id, state] of this.activeConnections) {
            if (now >= state.endTime) this.activeConnections.delete(id);
        }
    }
}
