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

    // Selection box
    selectionFill: string;
    selectionStroke: string;
    selectionStrokeWidth: number;
    selectionDash: number[];
}
