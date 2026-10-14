import { spawn } from 'node:child_process';
import { mkdir, mkdtemp, readFile } from 'node:fs/promises';
import { createServer } from 'node:net';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const currentURL = process.env.TEST_URL || 'http://127.0.0.1:3210/';
const require = createRequire(import.meta.url);
const vite = join(dirname(require.resolve('vite/package.json')), 'bin', 'vite.js');

async function run(script, args = [], env = {}) {
  await new Promise((resolveRun, reject) => {
    const child = spawn(process.execPath, [script, ...args], {
      cwd: root,
      env: { ...process.env, ...env },
      stdio: 'inherit',
    });
    child.once('error', reject);
    child.once('exit', (code, signal) => code === 0
      ? resolveRun()
      : reject(new Error(`${script} failed: ${signal || code}`)));
  });
}

async function freePort() {
  const listener = createServer();
  return await new Promise((resolvePort, reject) => {
    listener.once('error', reject);
    listener.listen(0, '127.0.0.1', () => {
      const port = listener.address().port;
      listener.close(error => error ? reject(error) : resolvePort(port));
    });
  });
}

async function startReference(port) {
  const child = spawn(process.execPath, [vite, 'test-output/ui-reference',
    '--host', '127.0.0.1', '--port', String(port), '--strictPort'], {
    cwd: root, env: process.env, stdio: ['ignore', 'pipe', 'pipe'],
  });
  try {
    await new Promise((ready, reject) => {
      let output = '';
      const timer = setTimeout(() => reject(new Error('Baseline server startup timed out')), 15000);
      const fail = error => { clearTimeout(timer); reject(error); };
      child.once('error', fail);
      child.once('exit', code => fail(new Error(`Baseline server exited: ${code}\n${output}`)));
      child.stderr.on('data', data => { output += data; });
      child.stdout.on('data', data => {
        output += data;
        if (output.includes(`http://127.0.0.1:${port}/`)) {
          clearTimeout(timer);
          ready();
        }
      });
    });
    return child;
  } catch (error) {
    child.kill('SIGTERM');
    throw error;
  }
}

await run('scripts/prepare-fixtures.mjs');
await run('scripts/prepare-ui-reference.mjs');
await mkdir(join(root, 'test-output/ui-baselines'), { recursive: true });
const baselineDir = await mkdtemp(join(root, 'test-output/ui-baselines/run-'));
const commit = JSON.parse(await readFile(join(root, 'tests/parity/baseline.json'))).commit;
const referencePort = await freePort();
const reference = await startReference(referencePort);
const sourceURL = `http://127.0.0.1:${referencePort}/`;
try {
  console.log(`Capture baseline from original Git commit ${commit}; artifacts: ${baselineDir}`);
  await run('tests/e2e/parity.mjs', ['--capture'], {
    TEST_URL: sourceURL, UI_BASELINE_DIR: baselineDir,
  });
  console.log(`Compare current application ${currentURL} against that original baseline`);
  await run('tests/e2e/parity.mjs', [], {
    TEST_URL: currentURL, UI_BASELINE_DIR: baselineDir,
  });
} finally {
  reference.kill('SIGTERM');
}
