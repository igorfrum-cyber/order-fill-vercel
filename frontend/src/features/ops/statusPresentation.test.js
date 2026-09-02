import assert from "node:assert/strict";
import test from "node:test";

import { presentStatus, statusHeadline } from "./statusPresentation.js";

test("presentStatus keeps a stable set of human tiles", () => {
  const tiles = presentStatus([{ id: "api", ok: true }, { id: "queue", ok: false }]);
  assert.deepEqual(
    tiles.map((tile) => tile.id),
    ["api", "worker", "postgres", "queue", "files"],
  );
  assert.equal(tiles[0].title, "API");
  assert.equal(tiles[0].ok, true);
  assert.equal(tiles[0].hint, "Отвечает");
  assert.equal(tiles[3].ok, false);
  assert.equal(tiles[3].hint, "Не отвечает");
  assert.equal(tiles[1].known, false);
  assert.equal(tiles[1].hint, "Проверяю…");
});

test("statusHeadline summarizes the board without jargon", () => {
  const ready = presentStatus([
    { id: "api", ok: true },
    { id: "worker", ok: true },
    { id: "postgres", ok: true },
    { id: "queue", ok: true },
    { id: "files", ok: true },
  ]);
  assert.equal(statusHeadline(ready), "Все сервисы отвечают");
  const mixed = presentStatus([{ id: "api", ok: true }, { id: "queue", ok: false }]);
  assert.equal(statusHeadline(mixed), "Проверяю сервисы");
  const knownDown = presentStatus([
    { id: "api", ok: true },
    { id: "worker", ok: true },
    { id: "postgres", ok: true },
    { id: "queue", ok: false },
    { id: "files", ok: true },
  ]);
  assert.equal(statusHeadline(knownDown), "Очередь не отвечает");
});
