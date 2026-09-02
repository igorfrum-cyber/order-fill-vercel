import test from "node:test";
import assert from "node:assert/strict";

import {
  defaultOrderMonth,
  formatOrderMonthLabel,
  isSelectableOrderMonth,
  sanitizeOrderMonth,
  selectableOrderMonths,
} from "./monthPolicy.js";

test("defaultOrderMonth is the next calendar month", () => {
  assert.equal(defaultOrderMonth(new Date("2026-08-29T10:00:00")), "2026-09");
  assert.equal(defaultOrderMonth(new Date("2026-12-15T10:00:00")), "2027-01");
});

test("formatOrderMonthLabel uses Russian month names", () => {
  assert.equal(formatOrderMonthLabel("2026-09"), "Сентябрь 2026");
  assert.equal(formatOrderMonthLabel("2027-01"), "Январь 2027");
});

test("selectableOrderMonths starts at the current month and excludes past months", () => {
  const options = selectableOrderMonths(new Date("2026-08-29T10:00:00"), 4);
  assert.deepEqual(options.map((item) => item.value), ["2026-08", "2026-09", "2026-10", "2026-11"]);
  assert.equal(options[1].label, "Сентябрь 2026");
});

test("isSelectableOrderMonth rejects months that already ended", () => {
  const now = new Date("2026-08-29T10:00:00");
  assert.equal(isSelectableOrderMonth("2026-07", now), false);
  assert.equal(isSelectableOrderMonth("2026-08", now), true);
  assert.equal(isSelectableOrderMonth("2026-09", now), true);
  assert.equal(isSelectableOrderMonth("not-a-month", now), false);
});

test("sanitizeOrderMonth falls back to the default when the value is in the past", () => {
  const now = new Date("2026-08-29T10:00:00");
  assert.equal(sanitizeOrderMonth("2026-07", now), "2026-09");
  assert.equal(sanitizeOrderMonth("2026-08", now), "2026-08");
  assert.equal(sanitizeOrderMonth("", now), "2026-09");
});
