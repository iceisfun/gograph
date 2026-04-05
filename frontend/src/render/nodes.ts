import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import type { Node, NodeType } from '../core/types.js';
import type { Rect } from '../core/geometry.js';
import {
    NODE_WIDTH,
    NODE_TITLE_HEIGHT,
    SLOT_SPACING,
    SLOT_RADIUS,
    SLOT_OFFSET_X,
    MIN_NODE_HEIGHT,
} from '../core/constants.js';

export function getNodeBounds(node: Node, nodeType: NodeType | undefined): Rect {
    const inputs = nodeType ? nodeType.slots.filter(s => s.direction === 'input').length : 0;
    const outputs = nodeType ? nodeType.slots.filter(s => s.direction === 'output').length : 0;
    const slotCount = Math.max(inputs, outputs);
    const bodyHeight = SLOT_SPACING * Math.max(slotCount, 1);
    const height = Math.max(MIN_NODE_HEIGHT, NODE_TITLE_HEIGHT + bodyHeight);

    return {
        x: node.position.x,
        y: node.position.y,
        width: NODE_WIDTH,
        height,
    };
}

function drawRoundedRect(
    ctx: CanvasRenderingContext2D,
    x: number,
    y: number,
    w: number,
    h: number,
    r: number,
): void {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.lineTo(x + w - r, y);
    ctx.arcTo(x + w, y, x + w, y + r, r);
    ctx.lineTo(x + w, y + h - r);
    ctx.arcTo(x + w, y + h, x + w - r, y + h, r);
    ctx.lineTo(x + r, y + h);
    ctx.arcTo(x, y + h, x, y + h - r, r);
    ctx.lineTo(x, y + r);
    ctx.arcTo(x, y, x + r, y, r);
    ctx.closePath();
}

export function drawNodes(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
    now: number = 0,
): void {
    const graph = store.graph.current;
    if (!graph) return;

    for (const node of Object.values(graph.nodes)) {
        const nodeType = store.graph.getNodeType(node.type);
        const bounds = getNodeBounds(node, nodeType);
        const isSelected = store.interaction.selectedNodes.has(node.id);
        const isHovered = store.interaction.hoveredNode === node.id;

        // Resolve category colors
        const category = nodeType?.category;
        const catColors = category ? theme.nodeCategories[category] : undefined;
        const nodeFill = catColors?.fill ?? theme.nodeFill;
        const nodeStroke = catColors?.stroke ?? theme.nodeStroke;
        const nodeTitleBar = catColors?.titleBar ?? theme.nodeTitleBar;

        // Draw node body
        drawRoundedRect(ctx, bounds.x, bounds.y, bounds.width, bounds.height, theme.nodeCornerRadius);
        ctx.fillStyle = nodeFill;
        ctx.fill();

        // Draw stroke
        if (isSelected) {
            ctx.strokeStyle = theme.nodeSelectedStroke;
            ctx.lineWidth = theme.nodeSelectedStrokeWidth;
        } else if (isHovered) {
            ctx.strokeStyle = theme.nodeHoverStroke;
            ctx.lineWidth = theme.nodeStrokeWidth;
        } else {
            ctx.strokeStyle = nodeStroke;
            ctx.lineWidth = theme.nodeStrokeWidth;
        }
        ctx.stroke();

        // Active node border animation
        const activeState = store.animation.activeNodes.get(node.id);
        if (activeState) {
            ctx.save();
            ctx.setLineDash(theme.nodeActiveBorderDash);
            ctx.lineDashOffset = -((now - activeState.startTime) / 1000) * 60;
            ctx.strokeStyle = theme.nodeActiveBorderColor;
            ctx.lineWidth = theme.nodeActiveBorderWidth;
            drawRoundedRect(ctx, bounds.x, bounds.y, bounds.width, bounds.height, theme.nodeCornerRadius);
            ctx.stroke();
            ctx.restore();
        }

        // Draw title bar (clip to top rounded corners)
        ctx.save();
        drawRoundedRect(ctx, bounds.x, bounds.y, bounds.width, bounds.height, theme.nodeCornerRadius);
        ctx.clip();

        ctx.fillStyle = nodeTitleBar;
        ctx.fillRect(bounds.x, bounds.y, bounds.width, NODE_TITLE_HEIGHT);

        // Draw title separator line
        ctx.beginPath();
        ctx.moveTo(bounds.x, bounds.y + NODE_TITLE_HEIGHT);
        ctx.lineTo(bounds.x + bounds.width, bounds.y + NODE_TITLE_HEIGHT);
        ctx.strokeStyle = theme.nodeStroke;
        ctx.lineWidth = 1;
        ctx.stroke();

        ctx.restore();

        // Draw title text
        ctx.fillStyle = theme.nodeTitleText;
        ctx.font = theme.nodeTitleFont;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        const label = node.label || (nodeType ? nodeType.label : node.type);
        ctx.fillText(label, bounds.x + bounds.width / 2, bounds.y + NODE_TITLE_HEIGHT / 2);

        // Draw config subtitle (e.g., duration for delay nodes)
        if (node.config?.duration) {
            ctx.fillStyle = theme.nodeSubtitleColor;
            ctx.font = theme.nodeSubtitleFont;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'top';
            ctx.fillText(
                `\u23F1 ${node.config.duration}ms`,
                bounds.x + bounds.width / 2,
                bounds.y + NODE_TITLE_HEIGHT + 4,
            );
        }

        // Draw slots
        if (nodeType) {
            const inputs = nodeType.slots.filter(s => s.direction === 'input');
            const outputs = nodeType.slots.filter(s => s.direction === 'output');

            // Input slots (left edge)
            for (let i = 0; i < inputs.length; i++) {
                const slot = inputs[i];
                const sx = bounds.x + SLOT_OFFSET_X;
                const sy = bounds.y + NODE_TITLE_HEIGHT + SLOT_SPACING * (i + 0.5);
                const isConnected = store.graph.isSlotConnected(node.id, slot.id);
                const isSlotHovered = store.interaction.hoveredSlot?.nodeId === node.id &&
                    store.interaction.hoveredSlot?.slotId === slot.id;

                ctx.beginPath();
                ctx.arc(sx, sy, SLOT_RADIUS, 0, Math.PI * 2);

                if (isConnected) {
                    ctx.fillStyle = theme.slotInputColor;
                    ctx.fill();
                } else {
                    ctx.fillStyle = nodeFill;
                    ctx.fill();
                    ctx.strokeStyle = theme.slotInputColor;
                    ctx.lineWidth = theme.slotStrokeWidth;
                    ctx.stroke();
                }

                if (isSlotHovered) {
                    ctx.beginPath();
                    ctx.arc(sx, sy, SLOT_RADIUS + 2, 0, Math.PI * 2);
                    ctx.strokeStyle = theme.slotInputColor;
                    ctx.lineWidth = 1;
                    ctx.stroke();
                }

                // Slot label
                ctx.fillStyle = theme.slotLabelColor;
                ctx.font = theme.slotLabelFont;
                ctx.textAlign = 'left';
                ctx.textBaseline = 'middle';
                ctx.fillText(slot.name, sx + SLOT_RADIUS + 4, sy);
            }

            // Output slots (right edge)
            for (let i = 0; i < outputs.length; i++) {
                const slot = outputs[i];
                const sx = bounds.x + NODE_WIDTH - SLOT_OFFSET_X;
                const sy = bounds.y + NODE_TITLE_HEIGHT + SLOT_SPACING * (i + 0.5);
                const isConnected = store.graph.isSlotConnected(node.id, slot.id);
                const isSlotHovered = store.interaction.hoveredSlot?.nodeId === node.id &&
                    store.interaction.hoveredSlot?.slotId === slot.id;

                ctx.beginPath();
                ctx.arc(sx, sy, SLOT_RADIUS, 0, Math.PI * 2);

                if (isConnected) {
                    ctx.fillStyle = theme.slotOutputColor;
                    ctx.fill();
                } else {
                    ctx.fillStyle = nodeFill;
                    ctx.fill();
                    ctx.strokeStyle = theme.slotOutputColor;
                    ctx.lineWidth = theme.slotStrokeWidth;
                    ctx.stroke();
                }

                if (isSlotHovered) {
                    ctx.beginPath();
                    ctx.arc(sx, sy, SLOT_RADIUS + 2, 0, Math.PI * 2);
                    ctx.strokeStyle = theme.slotOutputColor;
                    ctx.lineWidth = 1;
                    ctx.stroke();
                }

                // Slot label
                ctx.fillStyle = theme.slotLabelColor;
                ctx.font = theme.slotLabelFont;
                ctx.textAlign = 'right';
                ctx.textBaseline = 'middle';
                ctx.fillText(slot.name, sx - SLOT_RADIUS - 4, sy);
            }
        }
    }
}
