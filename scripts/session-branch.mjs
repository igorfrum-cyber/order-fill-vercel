#!/usr/bin/env node
import { execFileSync } from "node:child_process";

let branch = "";
try {
  branch = execFileSync("git", ["symbolic-ref", "--short", "HEAD"], { encoding: "utf8" }).trim();
} catch {
  branch = "(detached)";
}

const locked = new Set(["artemch", "dev", "main", "igorfrum"]);
const extra = locked.has(branch)
  ? `HEAD is ${branch}. Before any code change, read .cursor/skills/order-fill-work/SKILL.md and create feat/<slug> from artemch. Do not commit here. Merge to artemch only when the user asks. New microservice or screen needs explicit «да».`
  : `HEAD is ${branch}. Keep work on this branch. Read .cursor/skills/order-fill-work/SKILL.md. Merge to artemch only when the user asks. New microservice or screen needs explicit «да».`;

process.stdout.write(`${JSON.stringify({ additional_context: extra })}\n`);
