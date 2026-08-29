import { readdir, readFile } from "node:fs/promises";
import { extname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = fileURLToPath(new URL("..", import.meta.url));

async function jsFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await jsFiles(path)));
      continue;
    }
    if (extname(entry.name) === ".js") files.push(path);
  }
  return files;
}

const files = await jsFiles(join(root, "src"));
let failed = false;
for (const file of files) {
  const result = spawnSync(process.execPath, ["--check", file], { encoding: "utf8" });
  if (result.status !== 0) {
    failed = true;
    process.stderr.write(result.stderr || result.stdout || `syntax error: ${file}\n`);
  }
}

if (files.length === 0) {
  process.stderr.write("no frontend JavaScript files found under src/\n");
  process.exit(1);
}

const html = await readFile(join(root, "index.html"), "utf8");
if (!html.includes("/src/app.js")) {
  process.stderr.write("frontend/index.html must load /src/app.js\n");
  process.exit(1);
}

if (failed) process.exit(1);
console.log(`lint ok: ${files.length} files`);
