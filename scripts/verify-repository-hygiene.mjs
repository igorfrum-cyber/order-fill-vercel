import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

const ignoredExamples = [
  '.env', '.env.local', '.env.production', '.npmrc', '.netrc',
  'secrets/provider.json', 'credentials.json', 'service-account-prod.json',
  'private.pem', 'id_ed25519', '.aws/credentials',
  'node_modules/package/index.js', 'dist/assets/index.js',
  'test-output/result.xlsx', 'tests/fixtures/input/input.xlsx',
  'tests/parity/ui/order-desktop.png', 'tests/parity/ui/order-download.json',
  'tests/private/fixtures/customer.xlsx', 'customer.xlsm',
  'playwright-report/index.html', 'coverage/report.json',
  'screenshots/debug.png', 'Screenshot 2026-10-14.png',
  'Снимок экрана 2026-10-14.png', '.cache/result', 'debug.log',
];
const allowedExamples = [
  '.gitignore', '.env.example', '.env.production.example', '.env.template',
  'src/app.js', 'src/styles.css', 'src/ui/icon.svg', 'public/logo.png',
  'public/photo.webp', 'README.md', 'docs/architecture/modules.md',
  'package.json', 'package-lock.json', 'vercel.json',
  'tests/fixtures/workbooks.mjs', 'tests/parity/baseline.json',
];
function isIgnored(file) {
  const result = spawnSync('git', ['check-ignore', '--no-index', '--', file]);
  assert.ok([0, 1].includes(result.status), `git check-ignore failed for ${file}`);
  return result.status === 0;
}
for (const file of ignoredExamples) assert.ok(isIgnored(file), `Must be ignored: ${file}`);
for (const file of allowedExamples) assert.ok(!isIgnored(file), `Required asset/config must remain versionable: ${file}`);

const tracked = execFileSync('git', ['ls-files', '-z'], { encoding: 'utf8' }).split('\0').filter(Boolean);
const ignoredTracked = tracked.filter(isIgnored);
assert.deepEqual(ignoredTracked, [], 'Tracked files violate .gitignore; remove only their index entries');

const secretPatterns = [
  /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/,
  /\bgh[pousr]_[A-Za-z0-9]{30,}\b/,
  /\bgithub_pat_[A-Za-z0-9_]{40,}\b/,
  /\bAKIA[A-Z0-9]{16}\b/,
  /\bsk-[A-Za-z0-9_-]{30,}\b/,
];
const suspicious = tracked.filter(file => {
  const content = readFileSync(file).toString('utf8');
  return secretPatterns.some(pattern => pattern.test(content));
});
assert.deepEqual(suspicious, [], 'Possible credential found in tracked files; inspect locally without printing its value');
console.log('Repository hygiene: ignored secrets/artifacts, allowed runtime assets/config, no ignored tracked files or recognized key signatures');
