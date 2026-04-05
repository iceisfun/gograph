import type { Theme } from './theme.js';

export const defaultTheme: Theme = {
    // Canvas
    background: '#1a1a2e',

    // Grid
    gridMinor: 'rgba(255, 255, 255, 0.063)',
    gridMajor: 'rgba(255, 255, 255, 0.125)',
    gridMinorWidth: 1,
    gridMajorWidth: 1.5,

    // Nodes
    nodeFill: '#16213e',
    nodeStroke: '#0f3460',
    nodeStrokeWidth: 1.5,
    nodeTitleBar: '#1a1a4e',
    nodeTitleText: '#ffffff',
    nodeTitleFont: '13px sans-serif',
    nodeCornerRadius: 6,
    nodeHoverStroke: '#4a4a8a',
    nodeSelectedStroke: '#e94560',
    nodeSelectedStrokeWidth: 2,

    // Slots
    slotOutputColor: '#e94560',
    slotInputColor: '#0f9b8e',
    slotStrokeWidth: 1.5,
    slotLabelFont: '11px sans-serif',
    slotLabelColor: '#cccccc',

    // Connections
    connectionStroke: '#4a4a8a',
    connectionStrokeWidth: 2,
    connectionSelectedStroke: '#e94560',
    connectionSelectedStrokeWidth: 3,
    connectionHoverStroke: '#6a6aaa',
    connectionPreviewStroke: 'rgba(233, 69, 96, 0.5)',
    connectionPreviewDash: [6, 4],

    // Events
    eventDotColor: '#e94560',
    eventGlowColor: 'rgba(233, 69, 96, 0.6)',
    eventGlowRadius: 12,
    eventTrailOpacity: 0.3,

    // Selection box
    selectionFill: 'rgba(233, 69, 96, 0.1)',
    selectionStroke: 'rgba(233, 69, 96, 0.5)',
    selectionStrokeWidth: 1,
    selectionDash: [5, 3],
};
