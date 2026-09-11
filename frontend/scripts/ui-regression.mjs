#!/usr/bin/env node
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const cacheDir = join(root, ".cache");
const stampFile = join(cacheDir, "ui-tests.ok");
const neededFile = join(cacheDir, "frontend-ui-needed");

function fingerprint() {
  const diff = execFileSync("git", ["diff", "HEAD", "--", "frontend"], {
    cwd: root,
    encoding: "utf8",
  });
  const extra = execFileSync("git", ["ls-files", "--others", "--exclude-standard", "--", "frontend"], {
    cwd: root,
    encoding: "utf8",
  });
  return createHash("sha256").update(diff).update("\n").update(extra).digest("hex");
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

const mode = process.argv[2];

if (mode === "mark") {
  mkdirSync(cacheDir, { recursive: true });
  writeFileSync(stampFile, `${fingerprint()}\n`);
  rmSync(neededFile, { force: true });
  process.exit(0);
}

if (mode === "touched") {
  const path = String(readStdin().file_path || "");
  if (path.includes("/frontend/") || path.endsWith("/frontend")) {
    mkdirSync(cacheDir, { recursive: true });
    writeFileSync(neededFile, "1\n");
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
  const ok = existsSync(stampFile) && readFileSync(stampFile, "utf8").trim() === fingerprint();
  if (ok) {
    rmSync(neededFile, { force: true });
    reply({});
    process.exit(0);
  }
  reply({
    followup_message:
      "Frontend UI changed and `npm run test:ui --prefix frontend` has not passed on this diff. Run it (Chromium will open on screen). Read the failure, fix the root cause, re-run test:ui. Do not stop while it is red.",
  });
  process.exit(0);
}

process.stderr.write("usage: ui-regression.mjs mark|touched|stop\n");
process.exit(1);
