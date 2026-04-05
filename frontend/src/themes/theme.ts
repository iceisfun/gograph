export interface Theme {
    // Canvas
    background: string;

    // Grid
    gridMinor: string;
    gridMajor: string;
    gridMinorWidth: number;
    gridMajorWidth: number;

    // Nodes
    nodeFill: string;
    nodeStroke: string;
    nodeStrokeWidth: number;
    nodeTitleBar: string;
    nodeTitleText: string;
    nodeTitleFont: string;
    nodeCornerRadius: number;
    nodeHoverStroke: string;
    nodeSelectedStroke: string;
    nodeSelectedStrokeWidth: number;

    // Slots
    slotOutputColor: string;
    slotInputColor: string;
    slotStrokeWidth: number;
    slotLabelFont: string;
    slotLabelColor: string;

    // Connections
    connectionStroke: string;
    connectionStrokeWidth: number;
    connectionSelectedStroke: string;
    connectionSelectedStrokeWidth: number;
    connectionHoverStroke: string;
    connectionPreviewStroke: string;
    connectionPreviewDash: number[];

    // Events
    eventDotColor: string;
    eventGlowColor: string;
    eventGlowRadius: number;
    eventTrailOpacity: number;

    // Category-specific node colors
    nodeCategories: Record<string, {
        fill: string;
        stroke: string;
        titleBar: string;
    }>;

    // Active node border (emitting animation)
    nodeActiveBorderColor: string;
    nodeActiveBorderDash: number[];
    nodeActiveBorderWidth: number;

    // Active connection (event traversing)
    connectionActiveDash: number[];
    connectionActiveDashSpeed: number;
    connectionActiveStroke: string;
    connectionActiveGlowColor: string;
    connectionActiveGlowRadius: number;

    // Node config subtitle
    nodeSubtitleFont: string;
    nodeSubtitleColor: string;

    // Selection box
    selectionFill: string;
    selectionStroke: string;
    selectionStrokeWidth: number;
    selectionDash: number[];

    // Node shake animation
    nodeShakeDuration: number;
    nodeShakeIntensity: number;

    // Connection duration capsule
    connectionCapsuleVisibility: 'always' | 'hover' | 'related';
    connectionCapsuleMinDistance: number;
    connectionCapsuleFill: string;
    connectionCapsuleStroke: string;
    connectionCapsuleText: string;
    connectionCapsuleFont: string;

    // Node glow (delay/buffer holding)
    nodeGlowColor: string;
    nodeGlowRadius: number;

    // Compatible slot glow (during connection drag)
    slotCompatibleGlow: string;
    slotCompatibleGlowRadius: number;

    // Node content area
    nodeContentFont: string;
    nodeContentText: string;
}
