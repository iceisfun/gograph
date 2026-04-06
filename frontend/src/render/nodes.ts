import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import type { Node, NodeType } from '../core/types.js';
import type { Rect } from '../core/geometry.js';
import {
    NODE_WIDTH,
    NODE_TITLE_HEIGHT,
    SLOT_SPACING,
    SLOT_RADIUS,
    MIN_NODE_HEIGHT,
} from '../core/constants.js';

export function getNodeBounds(node: Node, nodeType: NodeType | undefined): Rect {
    const inputs = nodeType ? nodeType.slots.filter(s => s.direction === 'input').length : 0;
    const outputs = nodeType ? nodeType.slots.filter(s => s.direction === 'output').length : 0;
    const slotCount = Math.max(inputs, outputs);
    const bodyHeight = SLOT_SPACING * Math.max(slotCount, 1);
    const contentHeight = nodeType?.contentHeight || 0;
    const height = Math.max(MIN_NODE_HEIGHT, NODE_TITLE_HEIGHT + bodyHeight + contentHeight);

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
        const bounds = store.graph.getCachedNodeBounds(node.id) ?? getNodeBounds(node, nodeType);
        const isSelected = store.interaction.selectedNodes.has(node.id);
        const isHovered = store.interaction.hoveredNode === node.id;

        // Resolve category colors
        const category = nodeType?.category;
        const catColors = category ? theme.nodeCategories[category] : undefined;
        const nodeFill = catColors?.fill ?? theme.nodeFill;
        const nodeStroke = catColors?.stroke ?? theme.nodeStroke;
        const nodeTitleBar = catColors?.titleBar ?? theme.nodeTitleBar;

        // Check for shake animation
        const shake = store.animation.shakingNodes.get(node.id);
        let shaking = false;
        if (shake) {
            const elapsed = now - shake.startTime;
            const t = elapsed / shake.duration;
            if (t < 1) {
                shaking = true;
                const decay = Math.max(0, 1 - t);
                const dx = Math.sin(elapsed * 0.05) * shake.intensity * decay;
                const dy = Math.cos(elapsed * 0.07) * shake.intensity * decay;
                ctx.save();
                ctx.translate(dx, dy);
            }
        }

        // Glow effect for nodes holding/buffering (e.g. delay nodes)
        const glowState = store.animation.glowingNodes.get(node.id);
        if (glowState) {
            ctx.save();
            const elapsed = now - glowState.startTime;
            const pulse = 0.5 + 0.5 * Math.sin(elapsed * 0.005);

            // Strong shadow glow
            ctx.shadowBlur = (theme.nodeGlowRadius + 10) * pulse;
            ctx.shadowColor = theme.nodeGlowColor;
            drawRoundedRect(ctx, bounds.x, bounds.y, bounds.width, bounds.height, theme.nodeCornerRadius);
            ctx.fillStyle = nodeFill;
            ctx.fill();

            // Additive radial gradient overlay for extra intensity
            ctx.globalCompositeOperation = 'lighter';
            ctx.globalAlpha = 0.15 * pulse;
            const cx = bounds.x + bounds.width / 2;
            const cy = bounds.y + bounds.height / 2;
            const r = Math.max(bounds.width, bounds.height) * 0.7;
            const grad = ctx.createRadialGradient(cx, cy, 0, cx, cy, r);
            grad.addColorStop(0, theme.nodeGlowColor);
            grad.addColorStop(0.4, theme.nodeGlowColor);
            grad.addColorStop(1, 'rgba(0,0,0,0)');
            ctx.fillStyle = grad;
            ctx.fillRect(bounds.x - r, bounds.y - r, bounds.width + r * 2, bounds.height + r * 2);
            ctx.globalCompositeOperation = 'source-over';
            ctx.globalAlpha = 1;
            ctx.restore();
        }

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

        // Undo shake translation before drawing slots
        if (shaking) {
            ctx.restore();
        }

        // Draw slots from layout cache
        const slotLayouts = store.graph.getSlotLayouts(node.id);
        if (slotLayouts && nodeType) {
            for (const [slotId, slotLayout] of slotLayouts) {
                const slotDef = nodeType.slots.find(s => s.id === slotId);
                if (!slotDef) continue;

                const sx = slotLayout.position.x;
                const sy = slotLayout.position.y;
                const isInput = slotDef.direction === 'input';
                const slotColor = isInput ? theme.slotInputColor : theme.slotOutputColor;
                const isConnected = store.graph.isSlotConnected(node.id, slotId);
                const isSlotHovered = store.interaction.hoveredSlot?.nodeId === node.id &&
                    store.interaction.hoveredSlot?.slotId === slotId;

                // Compatible slot glow during connection drag
                const slotKey = `${node.id}:${slotLayout.slotId}`;
                const isDraggingConnection = store.interaction.dragState?.type === 'connection';
                const isCompatible = isDraggingConnection && store.interaction.compatibleSlots.has(slotKey);

                if (isCompatible) {
                    ctx.save();
                    ctx.shadowBlur = theme.slotCompatibleGlowRadius;
                    ctx.shadowColor = theme.slotCompatibleGlow;
                    const pulseR = SLOT_RADIUS + 4 + Math.sin(now * 0.008) * 2;
                    ctx.beginPath();
                    ctx.arc(sx, sy, pulseR, 0, Math.PI * 2);
                    ctx.strokeStyle = theme.slotCompatibleGlow;
                    ctx.lineWidth = 1.5;
                    ctx.stroke();
                    ctx.restore();
                }

                ctx.beginPath();
                ctx.arc(sx, sy, SLOT_RADIUS, 0, Math.PI * 2);

                if (isConnected) {
                    ctx.fillStyle = slotColor;
                    ctx.fill();
                } else {
                    ctx.fillStyle = nodeFill;
                    ctx.fill();
                    ctx.strokeStyle = slotColor;
                    ctx.lineWidth = theme.slotStrokeWidth;
                    ctx.stroke();
                }

                if (isSlotHovered) {
                    ctx.beginPath();
                    ctx.arc(sx, sy, SLOT_RADIUS + 2, 0, Math.PI * 2);
                    ctx.strokeStyle = slotColor;
                    ctx.lineWidth = 1;
                    ctx.stroke();
                }

                // Slot label positioned based on side
                ctx.fillStyle = theme.slotLabelColor;
                ctx.font = theme.slotLabelFont;
                ctx.textBaseline = 'middle';

                switch (slotLayout.side) {
                    case 'left':
                        ctx.textAlign = 'left';
                        ctx.fillText(slotDef.name, sx + SLOT_RADIUS + 4, sy);
                        break;
                    case 'right':
                        ctx.textAlign = 'right';
                        ctx.fillText(slotDef.name, sx - SLOT_RADIUS - 4, sy);
                        break;
                    case 'bottom':
                        ctx.textAlign = 'center';
                        ctx.textBaseline = 'top';
                        ctx.fillText(slotDef.name, sx, sy + SLOT_RADIUS + 2);
                        break;
                }
            }
        }

        // Draw content area (multi-slot)
        const content = store.graph.nodeContent.get(node.id);
        const contentH = nodeType?.contentHeight || 0;
        if (content && contentH > 0 && content.slots.size > 0) {
            const contentSlotLayouts = store.graph.getSlotLayouts(node.id);
            let leftCount = 0, rightCount = 0;
            if (contentSlotLayouts) {
                for (const sl of contentSlotLayouts.values()) {
                    if (sl.side === 'left') leftCount++;
                    else if (sl.side === 'right') rightCount++;
                }
            }
            const slotsH = SLOT_SPACING * Math.max(leftCount, rightCount, 1);
            const contentY = bounds.y + NODE_TITLE_HEIGHT + slotsH;

            // Separator
            ctx.beginPath();
            ctx.moveTo(bounds.x + 8, contentY);
            ctx.lineTo(bounds.x + bounds.width - 8, contentY);
            ctx.strokeStyle = nodeStroke;
            ctx.lineWidth = 0.5;
            ctx.stroke();

            // Clip to content area
            ctx.save();
            ctx.beginPath();
            ctx.rect(bounds.x + 4, contentY + 2, bounds.width - 8, contentH - 4);
            ctx.clip();

            const now = performance.now();
            const slotCount = content.slots.size;
            const lineHeight = Math.min(contentH - 4, Math.max(14, (contentH - 4) / slotCount));
            let yOffset = 0;

            for (const [slotName, slot] of content.slots) {
                const drawY = contentY + 4 + yOffset;

                // Font
                const fontSize = slot.size || 11;
                const fontFamily = slot.font || 'monospace';
                ctx.font = `${fontSize}px ${fontFamily}`;

                // Alignment
                const align = (slot.align || 'left') as CanvasTextAlign;
                ctx.textAlign = align;
                let textX: number;
                if (align === 'center') {
                    textX = bounds.x + bounds.width / 2;
                } else if (align === 'right') {
                    textX = bounds.x + bounds.width - 8;
                } else {
                    textX = bounds.x + 8;
                }
                ctx.textBaseline = 'top';

                // Color (may be animated)
                let textColor = slot.color || theme.nodeContentText;
                const animKey = `${node.id}:${slotName}`;
                const anim = store.animation.textSlotAnimations.get(animKey);

                if (anim) {
                    const elapsed = now - anim.startTime;
                    const t = Math.min(1, elapsed / anim.duration);

                    if (anim.type === 'flash') {
                        // Show animation color, fade back
                        textColor = t < 0.5 ? anim.color : (slot.color || theme.nodeContentText);
                    } else if (anim.type === 'pulse') {
                        const alpha = 0.3 + 0.7 * Math.abs(Math.sin(elapsed * 0.01));
                        ctx.globalAlpha = alpha;
                        textColor = anim.color;
                    }
                }

                ctx.fillStyle = textColor;
                ctx.fillText(slot.text, textX, drawY);

                if (anim?.type === 'pulse') {
                    ctx.globalAlpha = 1;
                }

                yOffset += lineHeight;
            }

            ctx.restore();
        }

        // Interactive node button
        if (nodeType?.interactive && contentH > 0) {
            const slotLayouts2 = store.graph.getSlotLayouts(node.id);
            let lc2 = 0, rc2 = 0;
            if (slotLayouts2) {
                for (const sl of slotLayouts2.values()) {
                    if (sl.side === 'left') lc2++;
                    else if (sl.side === 'right') rc2++;
                }
            }
            const slotsH2 = SLOT_SPACING * Math.max(lc2, rc2, 1);
            const interactiveY = bounds.y + NODE_TITLE_HEIGHT + slotsH2;

            const state = node.config?.state || 'off';
            const isOn = state === 'on';

            // Draw pill button centered in content area
            const btnW = 60;
            const btnH = 22;
            const btnR = btnH / 2;
            const btnX = bounds.x + bounds.width / 2 - btnW / 2;
            const btnY = interactiveY + (contentH - btnH) / 2;

            ctx.beginPath();
            ctx.moveTo(btnX + btnR, btnY);
            ctx.lineTo(btnX + btnW - btnR, btnY);
            ctx.arc(btnX + btnW - btnR, btnY + btnR, btnR, -Math.PI / 2, Math.PI / 2);
            ctx.lineTo(btnX + btnR, btnY + btnH);
            ctx.arc(btnX + btnR, btnY + btnR, btnR, Math.PI / 2, -Math.PI / 2);
            ctx.closePath();
            ctx.fillStyle = isOn ? theme.nodeInteractiveOnColor : theme.nodeInteractiveOffColor;
            ctx.fill();

            // Button label
            ctx.fillStyle = '#ffffff';
            ctx.font = '11px bold sans-serif';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(isOn ? 'ON' : 'OFF', bounds.x + bounds.width / 2, btnY + btnR);
        }
    }
}
