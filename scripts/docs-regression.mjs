#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const cacheDir = join(root, ".cache");
const neededFile = join(cacheDir, "docs-sync-needed");
const stubRe = /назначение см\. config\.go|RPC из protobuf-контракта/;

function isDocsSource(path) {
  const p = String(path || "").replaceAll("\\", "/");
  return (
    /\/internal\/config\/config\.go$/.test(p) ||
    p.endsWith(".proto") ||
    p.endsWith("/api/openapi.yaml") ||
    /\/transport\/httpapi\/router\.go$/.test(p)
  );
}

function hasAddedDocStubs(diff) {
  return String(diff)
    .split("\n")
    .some((line) => line.startsWith("+") && !line.startsWith("+++") && stubRe.test(line));
}

function readStdin() {
  try {
    return JSON.parse(readFileSync(0, "utf8") || "{}");
  } catch {
    return {};
  }
}

function reply(payload) {
  process.stdout.write(`${JSON.stringify(payload)}\n`);
}

function markDone() {
  rmSync(neededFile, { force: true });
}

if (!isDocsSource("backend/services/job-service/internal/config/config.go")) {
  throw new Error("docs-regression self-check: config.go must match");
}
if (isDocsSource("frontend/src/App.jsx") || isDocsSource("backend/proto/gen/go/x.pb.go")) {
  throw new Error("docs-regression self-check: false positive");
}
if (!hasAddedDocStubs("+| `FOO` | пусто | Читается из окружения; назначение см. config.go. |")) {
  throw new Error("docs-regression self-check: stub detect failed");
}

const mode = process.argv[2];

if (mode === "mark") {
  markDone();
  process.exit(0);
}

if (mode === "touched") {
  const path = String(readStdin().file_path || "");
  if (isDocsSource(path)) {
    mkdirSync(cacheDir, { recursive: true });
    writeFileSync(neededFile, "1\n");
    try {
      execFileSync("node", [join(root, "scripts/sync-docs.mjs"), "--write"], { cwd: root, stdio: "ignore" });
    } catch {
      // ponytail: hook must not fail the edit; stop-hook nags if lists drifted.
    }
  }
  reply({});
  process.exit(0);
}

if (mode === "stop") {
  const input = readStdin();
  if (input.status === "aborted" || !existsSync(neededFile)) {
    reply({});
    process.exit(0);
  }
  let diff = "";
  try {
    diff = execFileSync("git", ["diff", "HEAD", "--", "backend/services"], {
      cwd: root,
      encoding: "utf8",
    });
  } catch {
    diff = "";
  }
  if (!hasAddedDocStubs(diff)) {
    markDone();
    reply({});
    process.exit(0);
  }
  reply({
    followup_message:
      "Docs tables were regenerated. Load `.cursor/skills/order-fill-docs/SKILL.md`, replace stub purposes in `<!-- docs-sync:* -->` (keep other columns), and update the human README sections in the same change.",
  });
  process.exit(0);
}

process.stderr.write("usage: docs-regression.mjs mark|touched|stop\n");
process.exit(1);
