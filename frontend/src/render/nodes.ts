import type { AppStore } from '../state/store.js';
import type { Theme } from '../themes/theme.js';
import type { Node, NodeType } from '../core/types.js';
import type { Rect } from '../core/geometry.js';
import type {
    TextSlot, ProgressSlot, LedSlot,
    SpinnerSlot, BadgeSlot, SparklineSlot, ImageSlot, SvgSlot,
} from '../core/protocol.js';
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

// Module-level image cache for ImageSlot
const imageCache: Map<string, HTMLImageElement> = new Map();

// Module-level SVG blob URL cache for SvgSlot (keyed by markup hash)
const svgCache: Map<string, HTMLImageElement> = new Map();

function renderTextSlot(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
    nodeId: string,
    slotName: string,
    slot: TextSlot,
    bounds: Rect,
    _slotX: number,
    drawY: number,
    lineHeight: number,
    now: number,
): void {
    const fontSize = slot.size || 11;
    const fontFamily = slot.font || 'monospace';
    ctx.font = `${fontSize}px ${fontFamily}`;

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

    let textColor = slot.color || theme.nodeContentText;
    const animKey = `${nodeId}:${slotName}`;
    const anim = store.animation.textSlotAnimations.get(animKey);

    if (anim) {
        const elapsed = now - anim.startTime;
        const t = Math.min(1, elapsed / anim.duration);

        if (anim.type === 'flash') {
            textColor = t < 0.5 ? anim.color : (slot.color || theme.nodeContentText);
        } else if (anim.type === 'pulse') {
            const alpha = 0.3 + 0.7 * Math.abs(Math.sin(elapsed * 0.01));
            ctx.globalAlpha = alpha;
            textColor = anim.color;
        }
    }

    ctx.fillStyle = textColor;
    ctx.fillText(slot.text || '', textX, drawY);

    if (anim?.type === 'pulse') {
        ctx.globalAlpha = 1;
    }
}

function renderProgressSlot(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
    nodeId: string,
    slotName: string,
    slot: ProgressSlot,
    slotX: number,
    drawY: number,
    slotW: number,
    lineHeight: number,
): void {
    const barH = 8;
    const barY = drawY + (lineHeight - barH) / 2;
    const r = 3;

    // Track
    ctx.beginPath();
    ctx.moveTo(slotX + r, barY);
    ctx.lineTo(slotX + slotW - r, barY);
    ctx.arcTo(slotX + slotW, barY, slotX + slotW, barY + r, r);
    ctx.lineTo(slotX + slotW, barY + barH - r);
    ctx.arcTo(slotX + slotW, barY + barH, slotX + slotW - r, barY + barH, r);
    ctx.lineTo(slotX + r, barY + barH);
    ctx.arcTo(slotX, barY + barH, slotX, barY + barH - r, r);
    ctx.lineTo(slotX, barY + r);
    ctx.arcTo(slotX, barY, slotX + r, barY, r);
    ctx.closePath();
    ctx.fillStyle = theme.slotProgressTrack;
    ctx.fill();

    // Fill (animated or static)
    const value = store.animation.getProgressValue(nodeId, slotName, slot.value);
    const fillW = Math.max(0, value * slotW);
    if (fillW > 0) {
        ctx.beginPath();
        const fw = Math.max(r * 2, fillW);
        ctx.moveTo(slotX + r, barY);
        ctx.lineTo(slotX + fw - r, barY);
        ctx.arcTo(slotX + fw, barY, slotX + fw, barY + r, Math.min(r, fw / 2));
        ctx.lineTo(slotX + fw, barY + barH - r);
        ctx.arcTo(slotX + fw, barY + barH, slotX + fw - r, barY + barH, Math.min(r, fw / 2));
        ctx.lineTo(slotX + r, barY + barH);
        ctx.arcTo(slotX, barY + barH, slotX, barY + barH - r, r);
        ctx.lineTo(slotX, barY + r);
        ctx.arcTo(slotX, barY, slotX + r, barY, r);
        ctx.closePath();
        ctx.fillStyle = slot.color || theme.slotProgressFill;
        ctx.fill();
    }
}

function renderLedSlot(
    ctx: CanvasRenderingContext2D,
    theme: Theme,
    slot: LedSlot,
    slotX: number,
    drawY: number,
    slotW: number,
    lineHeight: number,
): void {
    const count = slot.states.length;
    if (count === 0) return;
    const radius = 5;
    const cy = drawY + lineHeight / 2;
    const spacing = slotW / count;

    for (let i = 0; i < count; i++) {
        const cx = slotX + spacing * (i + 0.5);
        const isOn = slot.states[i];

        ctx.beginPath();
        ctx.arc(cx, cy, radius, 0, Math.PI * 2);

        if (isOn) {
            ctx.save();
            ctx.shadowBlur = 6;
            ctx.shadowColor = theme.slotLedGlow;
            ctx.fillStyle = slot.color || theme.slotLedOn;
            ctx.fill();
            ctx.restore();
        } else {
            ctx.fillStyle = theme.slotLedOff;
            ctx.fill();
        }
    }
}

function renderSpinnerSlot(
    ctx: CanvasRenderingContext2D,
    theme: Theme,
    slot: SpinnerSlot,
    slotX: number,
    drawY: number,
    slotW: number,
    lineHeight: number,
): void {
    if (!slot.visible) return;
    const cx = slotX + slotW / 2;
    const cy = drawY + lineHeight / 2;
    const radius = 5;
    const startAngle = performance.now() * 0.006;

    ctx.beginPath();
    ctx.arc(cx, cy, radius, startAngle, startAngle + 4.5);
    ctx.strokeStyle = slot.color || theme.slotSpinnerColor;
    ctx.lineWidth = 2;
    ctx.stroke();
}

function renderBadgeSlot(
    ctx: CanvasRenderingContext2D,
    store: AppStore,
    theme: Theme,
    nodeId: string,
    slotName: string,
    slot: BadgeSlot,
    slotX: number,
    drawY: number,
    slotW: number,
    lineHeight: number,
    now: number,
): void {
    const pillH = lineHeight - 4;
    const pillR = pillH / 2;
    const pillY = drawY + 2;
    const text = slot.text || '';

    // Measure text to size pill
    ctx.font = theme.slotBadgeFont;
    const textW = ctx.measureText(text).width;
    const pillW = Math.min(slotW, textW + pillR * 2 + 4);
    const pillX = slotX + (slotW - pillW) / 2;

    // Pill background
    ctx.beginPath();
    ctx.moveTo(pillX + pillR, pillY);
    ctx.lineTo(pillX + pillW - pillR, pillY);
    ctx.arc(pillX + pillW - pillR, pillY + pillR, pillR, -Math.PI / 2, Math.PI / 2);
    ctx.lineTo(pillX + pillR, pillY + pillH);
    ctx.arc(pillX + pillR, pillY + pillR, pillR, Math.PI / 2, -Math.PI / 2);
    ctx.closePath();
    ctx.fillStyle = slot.background || '#555';
    ctx.fill();

    // Text (may be animated)
    let textColor = slot.color || '#fff';
    const animKey = `${nodeId}:${slotName}`;
    const anim = store.animation.textSlotAnimations.get(animKey);
    if (anim) {
        const elapsed = now - anim.startTime;
        const t = Math.min(1, elapsed / anim.duration);
        if (anim.type === 'flash') {
            textColor = t < 0.5 ? anim.color : (slot.color || '#fff');
        } else if (anim.type === 'pulse') {
            const alpha = 0.3 + 0.7 * Math.abs(Math.sin(elapsed * 0.01));
            ctx.globalAlpha = alpha;
            textColor = anim.color;
        }
    }

    ctx.fillStyle = textColor;
    ctx.font = theme.slotBadgeFont;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(text, pillX + pillW / 2, pillY + pillH / 2);

    if (anim?.type === 'pulse') {
        ctx.globalAlpha = 1;
    }
}

function renderSparklineSlot(
    ctx: CanvasRenderingContext2D,
    theme: Theme,
    slot: SparklineSlot,
    slotX: number,
    drawY: number,
    slotW: number,
    lineHeight: number,
): void {
    const values = slot.values;
    if (!values || values.length === 0) return;

    const minVal = slot.min ?? Math.min(...values);
    const maxVal = slot.max ?? Math.max(...values);
    const range = maxVal - minVal || 1;
    const yTop = drawY;
    const yBot = drawY + lineHeight;

    const toX = (i: number) => slotX + (i / (values.length - 1 || 1)) * slotW;
    const toY = (v: number) => yBot - ((v - minVal) / range) * lineHeight;

    // Fill area
    ctx.beginPath();
    ctx.moveTo(toX(0), yBot);
    for (let i = 0; i < values.length; i++) {
        ctx.lineTo(toX(i), toY(values[i]));
    }
    ctx.lineTo(toX(values.length - 1), yBot);
    ctx.closePath();
    ctx.fillStyle = theme.slotSparklineFill;
    ctx.fill();

    // Stroke line
    ctx.beginPath();
    ctx.moveTo(toX(0), toY(values[0]));
    for (let i = 1; i < values.length; i++) {
        ctx.lineTo(toX(i), toY(values[i]));
    }
    ctx.strokeStyle = slot.color || theme.slotSparklineStroke;
    ctx.lineWidth = 1.5;
    ctx.stroke();
}

function renderImageSlot(
    ctx: CanvasRenderingContext2D,
    slot: ImageSlot,
    slotX: number,
    drawY: number,
    slotW: number,
    lineHeight: number,
): void {
    const src = slot.src;
    if (!src) return;

    let img = imageCache.get(src);
    if (!img) {
        img = new Image();
        img.src = src;
        imageCache.set(src, img);
        // Will render on next frame once loaded
        return;
    }

    if (!img.complete || img.naturalWidth === 0) return;

    // Scale to fit slotW x lineHeight maintaining aspect ratio
    const aspect = img.naturalWidth / img.naturalHeight;
    let drawW = slotW;
    let drawH = slotW / aspect;
    if (drawH > lineHeight) {
        drawH = lineHeight;
        drawW = lineHeight * aspect;
    }
    const dx = slotX + (slotW - drawW) / 2;
    const dy = drawY + (lineHeight - drawH) / 2;
    ctx.drawImage(img, dx, dy, drawW, drawH);
}

function renderSvgSlot(
    ctx: CanvasRenderingContext2D,
    slot: SvgSlot,
    slotX: number,
    drawY: number,
    slotW: number,
    lineHeight: number,
): void {
    const markup = slot.markup;
    if (!markup) return;

    let img = svgCache.get(markup);
    if (!img) {
        const blob = new Blob([markup], { type: 'image/svg+xml' });
        const url = URL.createObjectURL(blob);
        img = new Image();
        img.src = url;
        svgCache.set(markup, img);
        img.onload = () => URL.revokeObjectURL(url);
        return;
    }

    if (!img.complete || img.naturalWidth === 0) return;

    const aspect = img.naturalWidth / img.naturalHeight;
    let drawW = slot.width || slotW;
    let drawH = slot.height || (drawW / aspect);
    if (drawH > lineHeight) {
        drawH = lineHeight;
        drawW = lineHeight * aspect;
    }
    if (drawW > slotW) {
        drawW = slotW;
        drawH = slotW / aspect;
    }
    const dx = slotX + (slotW - drawW) / 2;
    const dy = drawY + (lineHeight - drawH) / 2;
    ctx.drawImage(img, dx, dy, drawW, drawH);
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

            const slotNow = performance.now();
            const slotCount = content.slots.size;
            const lineHeight = Math.min(contentH - 4, Math.max(14, (contentH - 4) / slotCount));
            const slotX = bounds.x + 8;
            const slotW = bounds.width - 16;
            let yOffset = 0;

            for (const [slotName, slot] of content.slots) {
                const drawY = contentY + 4 + yOffset;

                switch (slot.type) {
                    case 'progress':
                        renderProgressSlot(ctx, store, theme, node.id, slotName, slot, slotX, drawY, slotW, lineHeight);
                        break;
                    case 'led':
                        renderLedSlot(ctx, theme, slot, slotX, drawY, slotW, lineHeight);
                        break;
                    case 'spinner':
                        renderSpinnerSlot(ctx, theme, slot, slotX, drawY, slotW, lineHeight);
                        break;
                    case 'badge':
                        renderBadgeSlot(ctx, store, theme, node.id, slotName, slot, slotX, drawY, slotW, lineHeight, slotNow);
                        break;
                    case 'sparkline':
                        renderSparklineSlot(ctx, theme, slot, slotX, drawY, slotW, lineHeight);
                        break;
                    case 'image':
                        renderImageSlot(ctx, slot, slotX, drawY, slotW, lineHeight);
                        break;
                    case 'svg':
                        renderSvgSlot(ctx, slot as SvgSlot, slotX, drawY, slotW, lineHeight);
                        break;
                    default:
                        // 'text' or legacy slots without type
                        renderTextSlot(ctx, store, theme, node.id, slotName, slot as TextSlot, bounds, slotX, drawY, lineHeight, slotNow);
                        break;
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
