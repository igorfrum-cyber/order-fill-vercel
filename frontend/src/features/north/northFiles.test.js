import test from "node:test";
import assert from "node:assert/strict";

import { sameNorthFile, uniqueNorthFiles } from "./northFiles.js";

const file = (name, size = 10, lastModified = 1) => ({ name, size, lastModified });

test("uniqueNorthFiles keeps the first copy of the same file", () => {
  const first = file("surgut.xlsx", 20, 5);
  const copy = file("surgut.xlsx", 20, 5);
  const other = file("urengoy.xlsx", 20, 5);
  assert.equal(sameNorthFile(first, copy), true);
  assert.deepEqual(uniqueNorthFiles([first, copy, other]).map((item) => item.name), ["surgut.xlsx", "urengoy.xlsx"]);
});
