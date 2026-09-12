import test from "node:test";
import assert from "node:assert/strict";

import { warehouseOfficeTarget, warehouseTransferQuantity } from "./warehouseTransfer.js";

test("warehouseTransferQuantity tops office up to 25%", () => {
  assert.equal(warehouseOfficeTarget(10, 90), 25);
  assert.equal(warehouseTransferQuantity(10, 90), 15);
  assert.equal(warehouseTransferQuantity(35, 65), 0);
  assert.equal(warehouseTransferQuantity(0, 40), 10);
});
