import { Renderer } from './render/renderer.js';
import { AppStore } from './state/store.js';
import { ApiClient } from './net/api.js';
import { SSEClient } from './net/sse.js';
import { InteractionHandler } from './interaction/handler.js';
import { defaultTheme } from './themes/default.js';

async function main() {
    const canvas = document.getElementById('graph-canvas') as HTMLCanvasElement;
    if (!canvas) throw new Error('Canvas not found');

    const api = new ApiClient();
    const config = await api.getConfig();
    const nodeTypes = await api.getNodeTypes();

    const store = new AppStore({ mode: config.mode as 'edit' | 'view' });
    store.graph.setNodeTypes(nodeTypes);

    // Try to load existing graphs
    const graphIds = await api.listGraphs();
    if (graphIds.length > 0) {
        const graph = await api.getGraph(graphIds[0]);
        store.graph.setGraph(graph);
    }

    const renderer = new Renderer(canvas, store, defaultTheme);
    const handler = new InteractionHandler(canvas, store, api);

    // SSE connection
    let sse: SSEClient | null = null;
    if (store.graph.current) {
        sse = new SSEClient(store.graph.current.id, '/api', {
            onGraphUpdate: (p) => { store.graph.setGraph(p.graph); },
            onNodeUpdate: (p) => { store.graph.updateNode(p.node); },
            onConnectionUpdate: (p) => { store.graph.updateConnection(p.connection); },
            onEventStart: (p) => {
                store.animation.startEvent(p);
                const conn = store.graph.getConnection(p.connectionID);
                if (conn) {
                    store.animation.activateNode(conn.fromNode, p.duration > 0 ? p.duration : 200);
                    store.animation.shakeNode(conn.fromNode);
                }
            },
            onNodeActive: (p) => {
                store.animation.glowNode(p.nodeID, p.duration);
            },
            onEventUpdate: (p) => { store.animation.updateEvent(p); },
            onEventEnd: (p) => { store.animation.endEvent(p.eventID); },
            onEventCancel: (p) => { store.animation.cancelEvent(p.eventID, p.immediate); },
            onNodeContent: (p) => {
                // Capture previous progress values before updating state
                const prevEntry = store.graph.nodeContent.get(p.nodeID);
                if (p.slots) {
                    for (const [name, slot] of Object.entries(p.slots)) {
                        // Trigger progress animation when duration > 0
                        if (slot.type === 'progress' && slot.duration && slot.duration > 0) {
                            let prevValue = 0;
                            if (prevEntry) {
                                const prev = prevEntry.slots.get(name);
                                if (prev && prev.type === 'progress') {
                                    prevValue = prev.value;
                                }
                            }
                            store.animation.animateProgress(
                                p.nodeID, name, prevValue, slot.value, slot.duration,
                            );
                        }
                        // Trigger text/badge animations
                        if (slot.animate && slot.animate !== 'none') {
                            store.animation.animateTextSlot(
                                p.nodeID, name, slot.animate,
                                slot.color || '#ffffff', slot.duration || 300,
                            );
                        }
                    }
                }
                store.graph.setNodeContent(p.nodeID, { text: p.text, image: p.image, slots: p.slots });
            },
            onConnectionState: (p) => {
                store.animation.setConnectionState(p.connectionID, p.active, p.value || '');
            },
        });
        sse.connect();
    }

    renderer.start();
    handler.start();

    // Expose for debugging
    Object.assign(window, { store, api, sse, renderer });
}

main().catch(console.error);
