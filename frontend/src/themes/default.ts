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

    // Category-specific node colors
    nodeCategories: {
        source: { fill: '#1a3a2e', stroke: '#0f6040', titleBar: '#1a4a3e' },
        transform: { fill: '#2a1a3e', stroke: '#5f3460', titleBar: '#3a1a4e' },
        output: { fill: '#3e2a1a', stroke: '#b05020', titleBar: '#4e3a1a' },
        delay: { fill: '#3e3a1a', stroke: '#a09020', titleBar: '#4e4a1a' },
    },

    // Active node border
    nodeActiveBorderColor: '#e94560',
    nodeActiveBorderDash: [8, 4],
    nodeActiveBorderWidth: 2.5,

    // Active connection
    connectionActiveDash: [10, 6],
    connectionActiveDashSpeed: 80,
    connectionActiveStroke: '#e94560',
    connectionActiveGlowColor: 'rgba(233, 69, 96, 0.4)',
    connectionActiveGlowRadius: 8,

    // Node config subtitle
    nodeSubtitleFont: '10px sans-serif',
    nodeSubtitleColor: '#999999',

    // Selection box
    selectionFill: 'rgba(233, 69, 96, 0.1)',
    selectionStroke: 'rgba(233, 69, 96, 0.5)',
    selectionStrokeWidth: 1,
    selectionDash: [5, 3],

    // Node shake animation
    nodeShakeDuration: 300,
    nodeShakeIntensity: 3,

    // Connection duration capsule
    // 'always' = show all, 'hover' = only when connection hovered/selected,
    // 'related' = when connection or either endpoint node hovered/selected
    connectionCapsuleVisibility: 'related',
    connectionCapsuleMinDistance: 100,
    connectionCapsuleFill: '#1a1a2e',
    connectionCapsuleStroke: '#4a4a8a',
    connectionCapsuleText: '#999999',
    connectionCapsuleFont: '10px sans-serif',

    // Node glow (delay/buffer holding)
    nodeGlowColor: 'rgba(160, 144, 32, 0.85)',
    nodeGlowRadius: 28,

    // Compatible slot glow (during connection drag)
    slotCompatibleGlow: 'rgba(15, 155, 142, 0.8)',
    slotCompatibleGlowRadius: 10,
};
