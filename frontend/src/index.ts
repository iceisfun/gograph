import { GoGraph } from './gograph.js';
export { GoGraph };
export type { GoGraphOptions } from './gograph.js';

// Auto-init from data attributes
function autoInit() {
    const elements = document.querySelectorAll('[data-gograph]');

    if (elements.length > 0) {
        for (const el of elements) {
            GoGraph.create(el as HTMLElement, {
                graphId: el.getAttribute('data-graph-id') || undefined,
                apiBase: el.getAttribute('data-api') || undefined,
                readOnly: el.hasAttribute('data-read-only'),
            });
        }
    } else {
        // Backward compat: full-page mode with existing canvas
        const canvas = document.getElementById('graph-canvas');
        if (canvas && canvas.parentElement) {
            GoGraph.create(canvas.parentElement);
        }
    }
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', autoInit);
} else {
    autoInit();
}

// Global export for <script> usage
(window as any).GoGraph = GoGraph;
