#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, isAbsolute, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const failures = [];

function fail(message) {
  failures.push(message);
}

function repositoryFiles() {
  const output = execFileSync(
    "git",
    ["ls-files", "--cached", "--others", "--exclude-standard", "--", "*.md", "llms.txt"],
    { cwd: root, encoding: "utf8" },
  );
  return [...new Set(output.split(/\r?\n/).filter(Boolean))].sort();
}

function localLinkTarget(markdownFile, rawTarget) {
  const withoutDelimiters = rawTarget.replace(/^<|>$/g, "");
  const withoutFragment = withoutDelimiters.split("#", 1)[0].split("?", 1)[0];
  if (
    !withoutFragment ||
    withoutFragment.startsWith("#") ||
    /^[a-z][a-z0-9+.-]*:/i.test(withoutFragment)
  ) {
    return "";
  }

  let decoded;
  try {
    decoded = decodeURIComponent(withoutFragment);
  } catch {
    fail(`${markdownFile}: ссылка содержит некорректный percent-encoding: ${rawTarget}`);
    return "";
  }

  return isAbsolute(decoded)
    ? resolve(root, `.${decoded}`)
    : resolve(root, dirname(markdownFile), decoded);
}

function verifyLinks(file) {
  const content = readFileSync(resolve(root, file), "utf8");
  const inlineLink = /\[[^\]]*\]\(([^)]+)\)/g;
  for (const match of content.matchAll(inlineLink)) {
    const target = localLinkTarget(file, match[1]);
    if (target && !existsSync(target)) {
      fail(`${file}: отсутствует цель ссылки ${match[1]}`);
    }
  }
}

function requireFile(file) {
  const path = resolve(root, file);
  if (!existsSync(path) || !statSync(path).isFile() || readFileSync(path, "utf8").trim() === "") {
    fail(`обязательный файл отсутствует или пуст: ${file}`);
  }
}

function requireHeading(file, pattern, description) {
  const content = readFileSync(resolve(root, file), "utf8");
  if (!pattern.test(content)) {
    fail(`${file}: отсутствует раздел ${description}`);
  }
}

for (const file of ["README.md", "CONTRIBUTING.md", "llms.txt", "docs/README.md", "docs/ARCHITECTURE.md"]) {
  requireFile(file);
}

const servicesRoot = resolve(root, "backend/services");
const serviceReadmes = readdirSync(servicesRoot)
  .filter((name) => existsSync(resolve(servicesRoot, name, "go.mod")))
  .map((name) => `backend/services/${name}/README.md`)
  .sort();

if (serviceReadmes.length === 0) {
  fail("в backend/services не найдено ни одного Go-сервиса");
}

for (const file of serviceReadmes) {
  requireFile(file);
  if (!existsSync(resolve(root, file))) continue;
  requireHeading(file, /^## Конфигурация\s*$/m, "«Конфигурация»");
  requireHeading(file, /^## Тест(?:ы|ирование)(?: и проверки)?\s*$/m, "«Тесты»");
  requireHeading(file, /^## Эксплуатац[^\n]*$/m, "«Эксплуатация»");
}

const docs = repositoryFiles();
for (const file of docs.filter((name) => name.endsWith(".md"))) {
  verifyLinks(file);
}

if (failures.length > 0) {
  for (const failure of failures) console.error(`docs: ${failure}`);
  process.exit(1);
}

console.log(`docs ok: ${docs.length} files, ${serviceReadmes.length} service README files`);
