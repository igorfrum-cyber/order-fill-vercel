import assert from "node:assert/strict";
import test from "node:test";

import { animationFrameProgress, easeOutCubic, isLastTourStep, mixRect, nextTourIndex, prefersReducedMotion, prevTourIndex, spotlightRect, tooltipLayout, visibleTourSteps } from "./tour.js";

test("tour navigation stays inside the step range", () => {
  assert.equal(nextTourIndex(0, 4), 1);
  assert.equal(nextTourIndex(3, 4), 3);
  assert.equal(prevTourIndex(2, 4), 1);
  assert.equal(prevTourIndex(0, 4), 0);
  assert.equal(isLastTourStep(2, 4), false);
  assert.equal(isLastTourStep(3, 4), true);
});

test("spotlightRect pads the highlighted control", () => {
  assert.deepEqual(spotlightRect({ left: 100, top: 40, width: 200, height: 80 }, 8), {
    left: 92,
    top: 32,
    width: 216,
    height: 96,
  });
});

test("tooltipLayout places the card below the target with an upward arrow", () => {
  const layout = tooltipLayout(
    { left: 80, top: 60, width: 200, height: 80 },
    { width: 280, height: 140 },
    { width: 1024, height: 768 },
    "bottom",
  );
  assert.equal(layout.arrow, "top");
  assert.ok(layout.top >= 60 + 80);
  assert.ok(layout.left >= 16);
  assert.ok(layout.left + 280 <= 1024 - 16);
});

test("tooltipLayout flips above when there is no room below", () => {
  const layout = tooltipLayout(
    { left: 80, top: 620, width: 200, height: 80 },
    { width: 280, height: 140 },
    { width: 1024, height: 768 },
    "bottom",
  );
  assert.equal(layout.arrow, "bottom");
  assert.ok(layout.top + 140 <= 620);
});

test("visibleTourSteps keeps only anchors that exist on screen", () => {
  const steps = [{ target: "order" }, { target: "north" }, { target: "help" }];
  assert.deepEqual(
    visibleTourSteps(steps, (id) => id !== "north").map((step) => step.target),
    ["order", "help"],
  );
});

test("mixRect interpolates spotlight geometry", () => {
  const mixed = mixRect(
    { left: 0, top: 0, width: 100, height: 40 },
    { left: 40, top: 20, width: 200, height: 80 },
    0.5,
  );
  assert.deepEqual(mixed, { left: 20, top: 10, width: 150, height: 60 });
});

test("easeOutCubic starts slow and finishes at 1", () => {
  assert.equal(easeOutCubic(0), 0);
  assert.equal(easeOutCubic(1), 1);
  assert.ok(easeOutCubic(0.5) > 0.5);
});

test("animationFrameProgress clamps to the 0..1 range", () => {
  assert.equal(animationFrameProgress(0, 400), 0);
  assert.equal(animationFrameProgress(200, 400), 0.5);
  assert.equal(animationFrameProgress(800, 400), 1);
});

test("prefersReducedMotion follows the system setting", () => {
  assert.equal(
    prefersReducedMotion(() => ({ matches: true })),
    true,
  );
  assert.equal(
    prefersReducedMotion(() => ({ matches: false })),
    false,
  );
});

test("prefersReducedMotion safely handles an unavailable media query", () => {
  assert.equal(prefersReducedMotion(undefined), false);
  assert.equal(prefersReducedMotion(() => { throw new Error("media query unavailable"); }), false);
});

test("tooltipLayout supports top, left and right placements and flips when needed", () => {
  const target = { left: 400, top: 300, width: 100, height: 40 };
  const size = { width: 120, height: 70 };
  const viewport = { width: 1000, height: 800 };

  assert.equal(tooltipLayout(target, size, viewport, "top").arrow, "bottom");
  assert.equal(tooltipLayout(target, size, viewport, "left").arrow, "right");
  assert.equal(tooltipLayout(target, size, viewport, "right").arrow, "left");

  assert.equal(tooltipLayout({ ...target, left: 20 }, size, viewport, "left").arrow, "left");
  assert.equal(tooltipLayout({ ...target, left: 850 }, size, viewport, "right").arrow, "right");
  assert.equal(tooltipLayout({ ...target, top: 10 }, size, viewport, "top").arrow, "top");
});

test("tooltipLayout keeps the minimum margin when the viewport is smaller than the card", () => {
  const layout = tooltipLayout(
    { left: 0, top: 0, width: 1, height: 1 },
    { width: 200, height: 200 },
    { width: 20, height: 20 },
  );
  assert.equal(layout.left, 16);
  assert.equal(layout.top, 16);
});
