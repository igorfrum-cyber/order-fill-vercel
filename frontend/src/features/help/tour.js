const GAP = 14;
const MARGIN = 16;

export function nextTourIndex(index, length) {
  if (length <= 0) return 0;
  return Math.min(index + 1, length - 1);
}

export function prevTourIndex(index, _length) {
  return Math.max(index - 1, 0);
}

export function isLastTourStep(index, length) {
  return length > 0 && index >= length - 1;
}

export function visibleTourSteps(steps, hasTarget) {
  return steps.filter((step) => hasTarget(step.target));
}

export function mixRect(from, to, t) {
  return {
    left: from.left + (to.left - from.left) * t,
    top: from.top + (to.top - from.top) * t,
    width: from.width + (to.width - from.width) * t,
    height: from.height + (to.height - from.height) * t,
  };
}

export function easeOutCubic(t) {
  return 1 - (1 - t) ** 3;
}

export function animationFrameProgress(elapsed, duration) {
  if (duration <= 0) return 1;
  if (elapsed <= 0) return 0;
  return Math.min(1, elapsed / duration);
}

export function prefersReducedMotion(query = globalThis.matchMedia) {
  try {
    return Boolean(query?.("(prefers-reduced-motion: reduce)")?.matches);
  } catch {
    return false;
  }
}

export const TOUR_MOVE_MS = 520;

export function spotlightRect(target, padding = 8) {
  return {
    left: target.left - padding,
    top: target.top - padding,
    width: target.width + padding * 2,
    height: target.height + padding * 2,
  };
}

export function tooltipLayout(target, size, viewport, placement = "bottom") {
  const side = fits(target, size, viewport, placement) ? placement : flip(placement);
  const centerX = target.left + target.width / 2;
  const centerY = target.top + target.height / 2;
  let left;
  let top;
  let arrow;
  if (side === "bottom") {
    left = centerX - size.width / 2;
    top = target.top + target.height + GAP;
    arrow = "top";
  } else if (side === "top") {
    left = centerX - size.width / 2;
    top = target.top - size.height - GAP;
    arrow = "bottom";
  } else if (side === "left") {
    left = target.left - size.width - GAP;
    top = centerY - size.height / 2;
    arrow = "right";
  } else {
    left = target.left + target.width + GAP;
    top = centerY - size.height / 2;
    arrow = "left";
  }
  left = clamp(left, MARGIN, viewport.width - size.width - MARGIN);
  top = clamp(top, MARGIN, viewport.height - size.height - MARGIN);
  return { left, top, arrow, arrowLeft: clamp(centerX - left, 20, size.width - 20) };
}

function fits(target, size, viewport, placement) {
  if (placement === "bottom") {
    return target.top + target.height + GAP + size.height <= viewport.height - MARGIN;
  }
  if (placement === "top") {
    return target.top - GAP - size.height >= MARGIN;
  }
  if (placement === "left") {
    return target.left - GAP - size.width >= MARGIN;
  }
  return target.left + target.width + GAP + size.width <= viewport.width - MARGIN;
}

function flip(placement) {
  if (placement === "bottom") return "top";
  if (placement === "top") return "bottom";
  if (placement === "left") return "right";
  return "left";
}

function clamp(value, min, max) {
  if (max < min) return min;
  return Math.min(Math.max(min, value), max);
}
