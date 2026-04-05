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
            onEventStart: (p) => { store.animation.startEvent(p); },
            onEventUpdate: (p) => { store.animation.updateEvent(p); },
            onEventEnd: (p) => { store.animation.endEvent(p.eventID); },
            onEventCancel: (p) => { store.animation.cancelEvent(p.eventID, p.immediate); },
        });
        sse.connect();
    }

    renderer.start();
    handler.start();

    // Expose for debugging
    Object.assign(window, { store, api, sse, renderer });
}

main().catch(console.error);
