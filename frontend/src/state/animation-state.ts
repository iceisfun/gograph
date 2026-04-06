import type { EventStartPayload, EventUpdatePayload } from '../core/protocol.js';

export interface ActiveEvent {
    eventID: string;
    connectionID: string;
    startTime: number;
    duration: number;
    color: string;
    intensity: number;
    progress: number;
}

export interface TextSlotAnimation {
    nodeID: string;
    slotName: string;
    type: 'flash' | 'pulse';
    color: string;
    startTime: number;
    duration: number;
}

export class AnimationState {
    activeEvents: Map<string, ActiveEvent> = new Map();
    activeNodes: Map<string, { startTime: number; endTime: number }> = new Map();
    activeConnections: Map<string, { startTime: number; endTime: number; color: string }> = new Map();
    shakingNodes: Map<string, { startTime: number; duration: number; intensity: number }> = new Map();
    glowingNodes: Map<string, { startTime: number; endTime: number }> = new Map();
    /** Steady state for instant (wiretype) connections. Persists until next update. */
    stateConnections: Map<string, { active: boolean; value: string }> = new Map();
    /** Text slot animations (flash, pulse). Key: "nodeID:slotName" */
    textSlotAnimations: Map<string, TextSlotAnimation> = new Map();

    activateNode(nodeId: string, durationMs: number): void {
        const now = performance.now();
        this.activeNodes.set(nodeId, { startTime: now, endTime: now + durationMs });
    }

    activateConnection(connectionId: string, durationMs: number, color: string): void {
        const now = performance.now();
        this.activeConnections.set(connectionId, { startTime: now, endTime: now + durationMs, color });
    }

    setConnectionState(connectionId: string, active: boolean, value: string): void {
        this.stateConnections.set(connectionId, { active, value });
    }

    glowNode(nodeId: string, durationMs: number): void {
        const now = performance.now();
        this.glowingNodes.set(nodeId, { startTime: now, endTime: now + durationMs });
    }

    animateTextSlot(nodeID: string, slotName: string, type: string, color: string, duration: number): void {
        if (type !== 'flash' && type !== 'pulse') return;
        const key = `${nodeID}:${slotName}`;
        this.textSlotAnimations.set(key, {
            nodeID,
            slotName,
            type: type as 'flash' | 'pulse',
            color,
            startTime: performance.now(),
            duration,
        });
    }

    shakeNode(nodeId: string, duration: number = 300, intensity: number = 3): void {
        this.shakingNodes.set(nodeId, {
            startTime: performance.now(),
            duration,
            intensity,
        });
    }

    startEvent(payload: EventStartPayload): void {
        if (payload.duration > 0) {
            // Timed: animate dot along curve
            this.activeEvents.set(payload.eventID, {
                eventID: payload.eventID,
                connectionID: payload.connectionID,
                startTime: performance.now(),
                duration: payload.duration,
                color: payload.color || '',
                intensity: 1,
                progress: 0,
            });
            this.activateConnection(payload.connectionID, payload.duration, payload.color || '');
        } else {
            // Instant: brief flash on connection (200ms dash), no dot
            this.activateConnection(payload.connectionID, 200, payload.color || '');
        }
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

        // Clean up expired glows
        for (const [id, state] of this.glowingNodes) {
            if (now >= state.endTime) this.glowingNodes.delete(id);
        }

        // Clean up expired shakes
        for (const [id, shake] of this.shakingNodes) {
            if (now >= shake.startTime + shake.duration) {
                this.shakingNodes.delete(id);
            }
        }

        // Clean up expired text slot animations
        for (const [key, anim] of this.textSlotAnimations) {
            if (now >= anim.startTime + anim.duration) {
                this.textSlotAnimations.delete(key);
            }
        }
    }
}
