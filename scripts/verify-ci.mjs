#!/usr/bin/env node

import { existsSync, readFileSync, readdirSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const workflowFile = ".github/workflows/verify.yml";
const workflow = readFileSync(resolve(root, workflowFile), "utf8");

function goModules(directory) {
  const modules = [];
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const path = resolve(directory, entry.name);
    if (existsSync(resolve(path, "go.mod"))) {
      modules.push(relative(root, path));
      continue;
    }
    modules.push(...goModules(path));
  }
  return modules;
}

const expected = goModules(resolve(root, "backend")).sort();
const configured = [];
let insideModules = false;

for (const line of workflow.split(/\r?\n/)) {
  if (/^\s{8}module:\s*$/.test(line)) {
    insideModules = true;
    continue;
  }
  if (!insideModules) continue;

  const item = line.match(/^\s{10}-\s+(backend\/\S+)\s*$/);
  if (item) {
    configured.push(item[1]);
    continue;
  }
  if (line.trim() && !/^\s{10,}/.test(line)) break;
}

const actual = [...new Set(configured)].sort();
const missing = expected.filter((module) => !actual.includes(module));
const stale = actual.filter((module) => !expected.includes(module));
const duplicateCount = configured.length - actual.length;

if (missing.length || stale.length || duplicateCount) {
  if (missing.length) console.error(`ci: modules missing from Go matrix: ${missing.join(", ")}`);
  if (stale.length) console.error(`ci: stale Go matrix entries: ${stale.join(", ")}`);
  if (duplicateCount) console.error("ci: Go matrix contains duplicate module entries");
  process.exit(1);
}

for (const requiredJob of [
  "docs-contracts",
  "frontend",
  "go-quality",
  "go-test",
  "go-vulnerability",
  "compose",
  "required",
]) {
  if (!new RegExp(`^  ${requiredJob}:\\s*$`, "m").test(workflow)) {
    console.error(`ci: required job is missing: ${requiredJob}`);
    process.exit(1);
  }
}

console.log(`ci config ok: ${actual.length} Go modules in matrix`);
