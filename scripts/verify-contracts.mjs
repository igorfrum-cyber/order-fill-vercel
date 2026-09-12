#!/usr/bin/env node

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const routerFile = "backend/services/gateway-service/internal/transport/httpapi/router.go";
const openAPIFile = "backend/services/gateway-service/api/openapi.yaml";
const pointerFile = "packages/contracts/openapi.yaml";
const failures = [];

function fail(message) {
  failures.push(message);
}

const router = readFileSync(resolve(root, routerFile), "utf8");
const openAPI = readFileSync(resolve(root, openAPIFile), "utf8");
const pointer = readFileSync(resolve(root, pointerFile), "utf8");

if (!/^openapi:\s+3\.1(?:\.\d+)?\s*$/m.test(openAPI)) {
  fail(`${openAPIFile}: ожидается OpenAPI 3.1.x`);
}

const runtimeOperations = [...router.matchAll(
  /mux\.Handle(?:Func)?\("(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD) ([^"]+)"/g,
)].map((match) => `${match[1]} ${match[2]}`);

const documentedOperations = [];
const nestedPaths = [];
let insidePaths = false;
let currentPath = "";

for (const line of openAPI.split(/\r?\n/)) {
  if (line === "paths:") {
    insidePaths = true;
    currentPath = "";
    continue;
  }
  if (insidePaths && /^[A-Za-z][A-Za-z0-9_-]*:/.test(line)) {
    insidePaths = false;
    currentPath = "";
  }
  if (!insidePaths) continue;

  const pathMatch = line.match(/^  (\/[^:]+):\s*$/);
  if (pathMatch) {
    currentPath = pathMatch[1];
    continue;
  }
  if (/^\s{4,}\/[^:]+:\s*$/.test(line)) {
    nestedPaths.push(line.trim().replace(/:$/, ""));
    continue;
  }

  const methodMatch = line.match(/^    (get|post|put|patch|delete|options|head):\s*$/);
  if (methodMatch && currentPath) {
    documentedOperations.push(`${methodMatch[1].toUpperCase()} ${currentPath}`);
  }
}

for (const path of nestedPaths) {
  fail(`${openAPIFile}: path ${path} вложен глубже верхнего уровня paths`);
}

function duplicates(values) {
  const seen = new Set();
  return [...new Set(values.filter((value) => seen.has(value) || !seen.add(value)))];
}

for (const operation of duplicates(runtimeOperations)) {
  fail(`${routerFile}: маршрут зарегистрирован повторно: ${operation}`);
}
for (const operation of duplicates(documentedOperations)) {
  fail(`${openAPIFile}: операция описана повторно: ${operation}`);
}

const runtimeSet = new Set(runtimeOperations);
const documentedSet = new Set(documentedOperations);
for (const operation of [...runtimeSet].sort()) {
  if (!documentedSet.has(operation)) fail(`${openAPIFile}: отсутствует ${operation}`);
}
for (const operation of [...documentedSet].sort()) {
  if (!runtimeSet.has(operation)) fail(`${openAPIFile}: нет runtime-маршрута ${operation}`);
}

if (!pointer.includes(openAPIFile)) {
  fail(`${pointerFile}: не указывает на канонический OpenAPI ${openAPIFile}`);
}
if (/example\.com\/proprietary-license/.test(openAPI)) {
  fail(`${openAPIFile}: содержит шаблонную ссылку на лицензию`);
}

if (failures.length > 0) {
  for (const failure of failures) console.error(`contracts: ${failure}`);
  process.exit(1);
}

console.log(`contracts ok: ${runtimeOperations.length} HTTP operations match OpenAPI`);
