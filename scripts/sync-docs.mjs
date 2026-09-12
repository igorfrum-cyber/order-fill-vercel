#!/usr/bin/env node
// Regenerates env/RPC/HTTP lists in service READMEs from config.go, proto, and router.go.
// Existing purpose text is kept when the key/RPC still exists.
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const write = process.argv.includes("--write");
const tlsKeys = [
  ["GRPC_TLS_MODE", "insecure"],
  ["GRPC_TLS_CERT_FILE", ""],
  ["GRPC_TLS_KEY_FILE", ""],
  ["GRPC_TLS_CA_FILE", ""],
  ["GRPC_TLS_SERVER_NAME", ""],
];

const protoDir = {
  "audit-service": "audit",
  "brand-service": "brand",
  "calculation-service": "calculation",
  "document-service": "documents",
  "file-service": "files",
  "identity-service": "identity",
  "job-service": "jobs",
  "matching-service": "matching",
  "passkey-service": "passkey",
  "twofa-service": "twofa",
};

function defaultPurpose(key) {
  if (key === "APP_ENV") {
    return "Общий fallback окружения; пустое значение и `local` включают local-режим.";
  }
  if (key === "GRPC_TLS_SERVER_NAME") {
    return "Необязательное имя для проверки TLS-сертификата исходящих gRPC-клиентов.";
  }
  if (key === "SESSION_COOKIE_SECURE") {
    return "`true` включает Secure-флаг cookie; вне local обязателен.";
  }
  if (key === "FILE_S3_USE_SSL") {
    return "TLS для подключения file-service к object storage; вне local должен быть true.";
  }
  return "Читается из окружения; назначение см. config.go.";
}

function collectEnv(configSrc) {
  const keys = [];
  const seen = new Set();
  const add = (key, fallback) => {
    if (!key || seen.has(key)) return;
    seen.add(key);
    keys.push({ key, fallback: fallback ?? "" });
  };
  for (const match of configSrc.matchAll(/getenv\("([A-Z][A-Z0-9_]*)"(?:\s*,\s*"([^"]*)")?/g)) {
    add(match[1], match[2] ?? "");
  }
  for (const match of configSrc.matchAll(/os\.Getenv\("([A-Z][A-Z0-9_]*)"\)/g)) {
    add(match[1], "");
  }
  for (const [key, fallback] of tlsKeys) add(key, fallback);
  return keys;
}

function collectRPCs(protoSrc) {
  const pkg = protoSrc.match(/^\s*package\s+([\w.]+)\s*;/m)?.[1] ?? "";
  const service = protoSrc.match(/^\s*service\s+(\w+)\s*\{/m)?.[1] ?? "";
  const rpcs = [...protoSrc.matchAll(/^\s*rpc\s+(\w+)\s*\(/gm)].map((match) => match[1]);
  return rpcs.map((name) => ({
    name,
    path: pkg && service ? `/${pkg}.${service}/${name}` : name,
  }));
}

function collectHTTP(routerSrc) {
  return [...routerSrc.matchAll(
    /mux\.Handle(?:Func)?\("(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD) ([^"]+)"/g,
  )].map((match) => `${match[1]} ${match[2]}`);
}

function parseTable(block) {
  const lines = block.trim().split(/\r?\n/).filter((line) => line.startsWith("|"));
  if (lines.length < 2) return { header: [], sep: "", rows: new Map(), extra: [] };
  const header = lines[0];
  const sep = lines[1];
  const rows = new Map();
  const extra = [];
  for (const line of lines.slice(2)) {
    const cells = line.split("|").slice(1, -1).map((cell) => cell.trim());
    const key = cells[0]?.replace(/^`|`$/g, "") ?? "";
    if (key) rows.set(key, cells);
    else extra.push(line);
  }
  return { header, sep, rows, extra };
}

function envTable(oldBlock, keys) {
  const parsed = parseTable(oldBlock);
  const header = parsed.header || "| Переменная | По умолчанию | Обязательность и назначение |";
  const sep = parsed.sep || "| --- | --- | --- |";
  const lines = [header, sep];
  for (const { key, fallback } of keys) {
    const prev = parsed.rows.get(key);
    if (prev) {
      const cells = [...prev];
      cells[0] = `\`${key}\``;
      if (cells[1] !== undefined) cells[1] = fallback === "" ? (cells[1] || "пусто") : `\`${fallback}\``;
      if (cells[2] && /уточните назначение|config\.go/.test(cells[2])) cells[2] = defaultPurpose(key);
      lines.push("| " + cells.join(" | ") + " |");
      continue;
    }
    const def = fallback === "" ? "пусто" : `\`${fallback}\``;
    lines.push(`| \`${key}\` | ${def} | ${defaultPurpose(key)} |`);
  }
  return lines.join("\n");
}

function rpcTable(oldBlock, rpcs) {
  const parsed = parseTable(oldBlock);
  const header = parsed.header.includes("RPC")
    ? parsed.header
    : "| RPC | Полный gRPC method | Назначение |";
  const sep = parsed.header.includes("RPC") ? parsed.sep : "| --- | --- | --- |";
  const lines = [header, sep];
  for (const rpc of rpcs) {
    const prev = parsed.header.includes("RPC")
      ? parsed.rows.get(rpc.name) || parsed.rows.get(`\`${rpc.name}\``)
      : undefined;
    if (prev) {
      const cells = [...prev];
      cells[0] = `\`${rpc.name}\``;
      if (cells[1] !== undefined) cells[1] = `\`${rpc.path}\``;
      lines.push("| " + cells.join(" | ") + " |");
      continue;
    }
    lines.push(`| \`${rpc.name}\` | \`${rpc.path}\` | RPC из protobuf-контракта. |`);
  }
  return lines.join("\n");
}

function httpList(operations) {
  return operations.map((op) => `- \`${op}\``).join("\n");
}

function replaceOrWrap(content, name, heading, nextHeading, body) {
  const start = `<!-- docs-sync:${name} -->`;
  const end = `<!-- /docs-sync:${name} -->`;
  const block = `${start}\n${body}\n${end}`;
  if (content.includes(start) && content.includes(end)) {
    return content.replace(new RegExp(`${escapeReg(start)}[\\s\\S]*?${escapeReg(end)}`), block);
  }
  const headingIdx = content.search(heading);
  if (headingIdx < 0) return content;
  const afterHeading = content.slice(headingIdx);
  const headingLineEnd = afterHeading.indexOf("\n");
  let region = afterHeading.slice(headingLineEnd + 1);
  const stop = region.search(nextHeading);
  if (stop >= 0) region = region.slice(0, stop);
  const tableMatch = region.match(/\n(\|.+\n\|[-:| ]+\n(?:\|.*\n)+)/);
  const listMatch = region.match(/\n((?:- `.+`\n)+)/);
  const target = tableMatch?.[1] ?? listMatch?.[1];
  if (!target) {
    const insertAt = headingIdx + headingLineEnd + 1;
    return content.slice(0, insertAt) + "\n" + block + "\n" + content.slice(insertAt);
  }
  return content.replace(target, `\n${block}\n`);
}

function escapeReg(value) {
  return value.replace(/[/\\^$*+?.()|[\]{}]/g, "\\$&");
}

function nextSection() {
  return /\n## /;
}

function syncService(name) {
  const readmePath = join(root, "backend/services", name, "README.md");
  const configPath = join(root, "backend/services", name, "internal/config/config.go");
  if (!existsSync(readmePath) || !existsSync(configPath)) return null;
  let content = readFileSync(readmePath, "utf8");
  const original = content;
  const envKeys = collectEnv(readFileSync(configPath, "utf8"));
  content = replaceOrWrap(
    content,
    "env",
    /^## Конфигурация\s*$/m,
    nextSection(),
    envTable(extractMarked(original, "env") || extractFirstTableAfter(original, /^## Конфигурация\s*$/m), envKeys),
  );

  const protoFolder = protoDir[name];
  if (protoFolder) {
    const protoPath = join(root, "backend/proto/orderfill", protoFolder, "v1", `${protoFolder}.proto`);
    if (existsSync(protoPath)) {
      const rpcs = collectRPCs(readFileSync(protoPath, "utf8"));
      content = replaceOrWrap(
        content,
        "rpc",
        /^## (?:Внутренний )?gRPC API\s*$|^## API\s*$/m,
        nextSection(),
        rpcTable(extractMarked(original, "rpc") || extractFirstTableAfter(original, /^## (?:Внутренний )?gRPC API\s*$|^## API\s*$/m), rpcs),
      );
    }
  }

  if (name === "gateway-service") {
    const router = readFileSync(join(root, "backend/services/gateway-service/internal/transport/httpapi/router.go"), "utf8");
    const httpBody = httpList(collectHTTP(router));
    if (content.includes("<!-- docs-sync:http -->")) {
      content = replaceOrWrap(content, "http", /^## Публичный HTTP API\s*$/m, nextSection(), httpBody);
    } else {
      content = content.replace(
        /(Реально зарегистрированные маршруты:\n)[\s\S]*?(\nБез cookie)/,
        `$1\n<!-- docs-sync:http -->\n${httpBody}\n<!-- /docs-sync:http -->\n$2`,
      );
    }
  }

  if (content !== original) {
    if (write) writeFileSync(readmePath, content);
    return readmePath;
  }
  return null;
}

function extractMarked(content, name) {
  const match = content.match(new RegExp(`<!-- docs-sync:${name} -->\\s*([\\s\\S]*?)<!-- /docs-sync:${name} -->`));
  return match?.[1] ?? "";
}

function extractFirstTableAfter(content, heading) {
  const headingIdx = content.search(heading);
  if (headingIdx < 0) return "";
  let after = content.slice(headingIdx);
  const lineEnd = after.indexOf("\n");
  after = after.slice(lineEnd + 1);
  const stop = after.search(/\n## /);
  if (stop >= 0) after = after.slice(0, stop);
  const match = after.match(/(\|.+\n\|[-:| ]+\n(?:\|.*\n)+)/);
  return match?.[1] ?? "";
}

function selfCheck() {
  const env = collectEnv(`
    getenv("JOB_GRPC_ADDR", ":9094")
    getenv("JOB_ENV", getenv("APP_ENV", "local"))
    getenv("DATABASE_URL", "")
    os.Getenv("SESSION_COOKIE_SECURE")
  `);
  const keys = env.map((item) => item.key);
  if (!keys.includes("JOB_GRPC_ADDR") || !keys.includes("APP_ENV") || !keys.includes("GRPC_TLS_MODE")) {
    throw new Error("sync-docs self-check: env parse failed");
  }
  const rpcs = collectRPCs(`
    package orderfill.jobs.v1;
    service JobService {
      rpc CreateJob(CreateJobRequest) returns (CreateJobResponse);
    }
  `);
  if (rpcs[0]?.path !== "/orderfill.jobs.v1.JobService/CreateJob") {
    throw new Error("sync-docs self-check: rpc parse failed");
  }
}

selfCheck();

const services = readdirSync(join(root, "backend/services")).filter((name) =>
  existsSync(join(root, "backend/services", name, "go.mod")),
);
const changed = [];
for (const name of services.sort()) {
  const path = syncService(name);
  if (path) changed.push(path);
}

if (changed.length === 0) {
  console.log("docs-sync: up to date");
  process.exit(0);
}

if (!write) {
  console.error("docs-sync: README drift, run: node scripts/sync-docs.mjs --write");
  for (const path of changed) console.error(`  ${path}`);
  process.exit(1);
}

console.log(`docs-sync: updated ${changed.length} README files`);
