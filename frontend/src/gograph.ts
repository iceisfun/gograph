import { Renderer } from './render/renderer.js';
import { AppStore } from './state/store.js';
import { ApiClient } from './net/api.js';
import { SSEClient } from './net/sse.js';
import { InteractionHandler } from './interaction/handler.js';
import { defaultTheme } from './themes/default.js';
import type { Theme } from './themes/theme.js';

export interface GoGraphOptions {
    graphId?: string;
    apiBase?: string;
    readOnly?: boolean;
    darkMode?: boolean;
    theme?: Partial<Theme>;
}

export class GoGraph {
    private _store: AppStore;
    private _api: ApiClient;
    private _sse: SSEClient | null = null;
    private _renderer: Renderer;
    private _handler: InteractionHandler;
    private _canvas: HTMLCanvasElement;

    private constructor(
        canvas: HTMLCanvasElement,
        store: AppStore,
        api: ApiClient,
        renderer: Renderer,
        handler: InteractionHandler,
    ) {
        this._canvas = canvas;
        this._store = store;
        this._api = api;
        this._renderer = renderer;
        this._handler = handler;
    }

    static async create(element: HTMLElement, options: GoGraphOptions = {}): Promise<GoGraph> {
        // 1. Create canvas, append to element
        const canvas = document.createElement('canvas');
        canvas.style.width = '100%';
        canvas.style.height = '100%';
        canvas.style.display = 'block';
        element.style.overflow = 'hidden';
        element.appendChild(canvas);

        // 2. Determine apiBase
        let apiBase = options.apiBase || '/api';
        if (!options.apiBase) {
            try {
                const resp = await fetch('/config');
                if (resp.ok) {
                    const cfg = await resp.json();
                    if (cfg.apiBase) apiBase = cfg.apiBase;
                }
            } catch { /* use default */ }
        }

        // 3. Init ApiClient
        const api = new ApiClient(apiBase);

        // Fetch config for mode (unless overridden by readOnly option)
        let mode: 'edit' | 'view' = options.readOnly ? 'view' : 'edit';
        if (options.readOnly === undefined) {
            try {
                const config = await api.getConfig();
                mode = config.mode as 'edit' | 'view';
            } catch { /* default to edit */ }
        }

        const nodeTypes = await api.getNodeTypes();

        // 4. Init store
        const store = new AppStore({ mode });
        store.graph.setNodeTypes(nodeTypes);

        // 5. Load graph
        if (options.graphId) {
            const graph = await api.getGraph(options.graphId);
            store.graph.setGraph(graph);
        } else {
            const ids = await api.listGraphs();
            if (ids.length > 0) {
                const graph = await api.getGraph(ids[0]);
                store.graph.setGraph(graph);
            }
        }

        // 6. Build theme (deep merge nodeCategories so users can add/override)
        const theme: Theme = options.theme
            ? {
                ...defaultTheme,
                ...options.theme,
                nodeCategories: {
                    ...defaultTheme.nodeCategories,
                    ...(options.theme.nodeCategories || {}),
                },
            }
            : { ...defaultTheme };

        // 7. Create renderer and handler
        const renderer = new Renderer(canvas, store, theme);
        const handler = new InteractionHandler(canvas, store, api);

        // 8. Create instance
        const instance = new GoGraph(canvas, store, api, renderer, handler);

        // 9. Start SSE
        if (store.graph.current) {
            instance._sse = new SSEClient(store.graph.current.id, apiBase, {
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
            instance._sse.connect();
        }

        // 10. Start renderer and interaction
        renderer.start();
        handler.start();

        // Expose for debugging
        Object.assign(window, { store, api, sse: instance._sse, renderer });

        return instance;
    }

    destroy(): void {
        this._sse?.disconnect();
        this._renderer.stop();
        this._handler.stop();
        this._canvas.remove();
    }

    get store(): AppStore { return this._store; }
    get api(): ApiClient { return this._api; }
}
