#!/usr/bin/env node
import { readFile } from "node:fs/promises";
import path from "node:path";
import { performance } from "node:perf_hooks";
import { fileURLToPath } from "node:url";

const DEFAULT_API = "http://127.0.0.1:8080";
const DEFAULT_JOBS = 20;
const DEFAULT_CONCURRENCY = 5;
const DEFAULT_BRAND = "angiopharm";
const DEFAULT_POLL_MS = 1000;
const DEFAULT_TIMEOUT_MS = 10 * 60 * 1000;
const DONE_STATUSES = new Set(["needs_review", "completed", "failed"]);

export function parseArgs(argv = process.argv.slice(2)) {
  const options = {
    apiBaseUrl: env("ORDERFILL_API", DEFAULT_API),
    jobs: intEnv("ORDERFILL_LOAD_JOBS", DEFAULT_JOBS),
    concurrency: intEnv("ORDERFILL_LOAD_CONCURRENCY", DEFAULT_CONCURRENCY),
    brand: env("ORDERFILL_BRAND", DEFAULT_BRAND),
    orderMonth: env("ORDERFILL_ORDER_MONTH", nextMonth(new Date())),
    sourcePath: env("ORDERFILL_SOURCE", ""),
    blankPaths: envList("ORDERFILL_BLANKS"),
    pollIntervalMs: intEnv("ORDERFILL_LOAD_POLL_MS", DEFAULT_POLL_MS),
    timeoutMs: intEnv("ORDERFILL_LOAD_TIMEOUT_MS", DEFAULT_TIMEOUT_MS),
    fetchReport: true,
    fetchFiles: true,
    fetchPreview: false,
    downloadArchive: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    switch (token) {
      case "--api":
        options.apiBaseUrl = readValue(argv, ++index, token);
        break;
      case "--jobs":
        options.jobs = parsePositiveInt(readValue(argv, ++index, token), token);
        break;
      case "--concurrency":
        options.concurrency = parsePositiveInt(readValue(argv, ++index, token), token);
        break;
      case "--brand":
        options.brand = readValue(argv, ++index, token);
        break;
      case "--order-month":
        options.orderMonth = readValue(argv, ++index, token);
        break;
      case "--source":
        options.sourcePath = readValue(argv, ++index, token);
        break;
      case "--blank":
        options.blankPaths.push(readValue(argv, ++index, token));
        break;
      case "--poll-ms":
        options.pollIntervalMs = parsePositiveInt(readValue(argv, ++index, token), token);
        break;
      case "--timeout-ms":
        options.timeoutMs = parsePositiveInt(readValue(argv, ++index, token), token);
        break;
      case "--preview":
        options.fetchPreview = true;
        break;
      case "--download-archive":
        options.downloadArchive = true;
        break;
      case "--no-report":
        options.fetchReport = false;
        break;
      case "--no-files":
        options.fetchFiles = false;
        break;
      case "--help":
      case "-h":
        throw new UsageError(usage());
      default:
        throw new Error(`Unknown argument: ${token}\n\n${usage()}`);
    }
  }

  if (!options.sourcePath) {
    throw new Error(`--source is required\n\n${usage()}`);
  }
  if (options.blankPaths.length === 0) {
    throw new Error(`--blank is required\n\n${usage()}`);
  }
  options.jobs = parsePositiveInt(options.jobs, "--jobs");
  options.concurrency = parsePositiveInt(options.concurrency, "--concurrency");
  options.pollIntervalMs = parsePositiveInt(options.pollIntervalMs, "--poll-ms");
  options.timeoutMs = parsePositiveInt(options.timeoutMs, "--timeout-ms");
  options.apiBaseUrl = options.apiBaseUrl.replace(/\/+$/, "");
  return options;
}

export function percentile(values, rank) {
  if (!values.length) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  const index = Math.max(0, Math.ceil(sorted.length * rank) - 1);
  return sorted[Math.min(index, sorted.length - 1)];
}

export function summarizeResults(results) {
  const okResults = results.filter((item) => item.ok);
  const enqueue = okResults.map((item) => item.enqueueMs);
  const completion = okResults.map((item) => item.totalMs);
  const statuses = {};
  let reportRows = 0;
  let files = 0;
  for (const result of results) {
    statuses[result.status || "unknown"] = (statuses[result.status || "unknown"] || 0) + 1;
    reportRows += result.reportRows || 0;
    files += result.files || 0;
  }
  return {
    total: results.length,
    ok: okResults.length,
    failed: results.length - okResults.length,
    statuses,
    enqueue: metricSummary(enqueue),
    completion: metricSummary(completion),
    reportRows,
    files,
  };
}

export async function runLoad(options, log = console.log) {
  const fixtures = await loadFixtures(options);
  const startedAt = performance.now();
  let completed = 0;
  log(
    `Starting load: jobs=${options.jobs}, concurrency=${options.concurrency}, blanks=${options.blankPaths.length}, api=${options.apiBaseUrl}`,
  );

  const results = await runPool(options.jobs, options.concurrency, async (index) => {
    const result = await runOneJob(index + 1, options, fixtures);
    completed += 1;
    const suffix = result.ok ? `${result.status} ${formatMs(result.totalMs)}` : `failed ${result.error}`;
    log(`[${completed}/${options.jobs}] job ${index + 1}: ${suffix}`);
    return result;
  });

  const wallMs = performance.now() - startedAt;
  const summary = summarizeResults(results);
  printSummary(summary, wallMs, log);
  if (summary.failed > 0) {
    process.exitCode = 1;
  }
  return summary;
}

async function runOneJob(index, options, fixtures) {
  const startedAt = performance.now();
  try {
    const createStartedAt = performance.now();
    const created = await createJob(index, options, fixtures);
    const enqueueMs = performance.now() - createStartedAt;
    const job = await pollJob(options, created.id, startedAt);
    let reportRows = 0;
    let files = 0;
    let outputFiles = [];

    if (options.fetchReport && job.status !== "failed") {
      const report = await requestJSON(options, `/api/v1/jobs/${encodeURIComponent(created.id)}/report`);
      reportRows = Array.isArray(report.rows) ? report.rows.length : 0;
    }
    if ((options.fetchFiles || options.fetchPreview) && job.status !== "failed") {
      const listed = await requestJSON(options, `/api/v1/jobs/${encodeURIComponent(created.id)}/files`);
      outputFiles = Array.isArray(listed.files) ? listed.files : [];
      files = outputFiles.length;
    }
    if (options.fetchPreview && job.status !== "failed") {
      await fetchFirstPreviewWindows(options, created.id, outputFiles);
    }
    if (options.downloadArchive && job.status !== "failed") {
      await requestBytes(options, `/api/v1/jobs/${encodeURIComponent(created.id)}/archive`);
    }

    return {
      ok: job.status !== "failed",
      id: created.id,
      status: job.status,
      enqueueMs,
      totalMs: performance.now() - startedAt,
      progress: job.progress,
      reportRows,
      files,
      error: job.error?.message || "",
    };
  } catch (error) {
    return {
      ok: false,
      status: "client_error",
      enqueueMs: 0,
      totalMs: performance.now() - startedAt,
      error: error.message || String(error),
    };
  }
}

async function createJob(index, options, fixtures) {
  const form = new FormData();
  form.append("brand", options.brand);
  form.append("order_month", options.orderMonth);
  form.append("source_file", fixtures.source.blob, numberedName(index, fixtures.source.name));
  for (const blank of fixtures.blanks) {
    form.append("blank_files", blank.blob, numberedName(index, blank.name));
  }
  return requestJSON(options, "/api/v1/jobs/order-fill", { method: "POST", body: form, expectedStatus: 202 });
}

async function pollJob(options, jobID, startedAt) {
  for (;;) {
    if (performance.now() - startedAt > options.timeoutMs) {
      throw new Error(`job ${jobID} timed out after ${options.timeoutMs}ms`);
    }
    const job = await requestJSON(options, `/api/v1/jobs/${encodeURIComponent(jobID)}`);
    if (DONE_STATUSES.has(job.status)) {
      return job;
    }
    await sleep(options.pollIntervalMs);
  }
}

async function fetchFirstPreviewWindows(options, jobID, files) {
  for (const file of files) {
    const meta = await requestJSON(options, `/api/v1/jobs/${encodeURIComponent(jobID)}/files/${encodeURIComponent(file.id)}/preview`);
    const sheet = meta.sheets?.[0];
    if (!sheet?.max_row) continue;
    const toRow = Math.min(sheet.max_row, 50);
    const params = new URLSearchParams({ sheet: "0", from_row: "1", to_row: String(toRow) });
    await requestJSON(
      options,
      `/api/v1/jobs/${encodeURIComponent(jobID)}/files/${encodeURIComponent(file.id)}/preview/window?${params}`,
    );
  }
}

async function requestJSON(options, pathname, init = {}) {
  const response = await fetch(options.apiBaseUrl + pathname, {
    method: init.method || "GET",
    body: init.body,
  });
  const text = await response.text();
  let payload = null;
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      throw new Error(`${pathname}: response is not JSON: ${text.slice(0, 160)}`);
    }
  }
  const expectedStatus = init.expectedStatus || 200;
  if (response.status !== expectedStatus) {
    throw new Error(`${pathname}: HTTP ${response.status}: ${payload?.message || text}`);
  }
  return payload;
}

async function requestBytes(options, pathname) {
  const response = await fetch(options.apiBaseUrl + pathname);
  if (response.status !== 200) {
    const text = await response.text();
    throw new Error(`${pathname}: HTTP ${response.status}: ${text.slice(0, 160)}`);
  }
  return response.arrayBuffer();
}

async function loadFixtures(options) {
  const source = await loadFixture(options.sourcePath);
  const blanks = [];
  for (const blankPath of options.blankPaths) {
    blanks.push(await loadFixture(blankPath));
  }
  return { source, blanks };
}

async function loadFixture(filePath) {
  const content = await readFile(filePath);
  return {
    name: path.basename(filePath),
    blob: new Blob([content], {
      type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    }),
  };
}

async function runPool(count, concurrency, worker) {
  const results = new Array(count);
  let next = 0;
  const workers = Array.from({ length: Math.min(concurrency, count) }, async () => {
    for (;;) {
      const index = next;
      next += 1;
      if (index >= count) return;
      results[index] = await worker(index);
    }
  });
  await Promise.all(workers);
  return results;
}

function metricSummary(values) {
  if (!values.length) {
    return { min: 0, p50: 0, p95: 0, max: 0 };
  }
  return {
    min: Math.min(...values),
    p50: percentile(values, 0.5),
    p95: percentile(values, 0.95),
    max: Math.max(...values),
  };
}

function printSummary(summary, wallMs, log) {
  log("");
  log(`Load summary: total=${summary.total}, ok=${summary.ok}, failed=${summary.failed}, wall=${formatMs(wallMs)}`);
  log(`Statuses: ${JSON.stringify(summary.statuses)}`);
  log(`Enqueue latency: min=${formatMs(summary.enqueue.min)}, p50=${formatMs(summary.enqueue.p50)}, p95=${formatMs(summary.enqueue.p95)}, max=${formatMs(summary.enqueue.max)}`);
  log(`Completion latency: min=${formatMs(summary.completion.min)}, p50=${formatMs(summary.completion.p50)}, p95=${formatMs(summary.completion.p95)}, max=${formatMs(summary.completion.max)}`);
  log(`Fetched report rows=${summary.reportRows}, listed files=${summary.files}`);
}

function numberedName(index, fileName) {
  const extension = path.extname(fileName);
  const base = path.basename(fileName, extension);
  return `${base}-${String(index).padStart(4, "0")}${extension}`;
}

function readValue(argv, index, flag) {
  const value = argv[index];
  if (!value || value.startsWith("--")) {
    throw new Error(`${flag} requires a value`);
  }
  return value;
}

function parsePositiveInt(value, flag) {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < 1) {
    throw new Error(`${flag} must be a positive integer`);
  }
  return parsed;
}

function env(name, fallback) {
  return process.env[name]?.trim() || fallback;
}

function intEnv(name, fallback) {
  const value = process.env[name]?.trim();
  return value ? parsePositiveInt(value, name) : fallback;
}

function envList(name) {
  return (process.env[name] || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function nextMonth(date) {
  const year = date.getUTCFullYear();
  const month = date.getUTCMonth() + 2;
  const normalized = new Date(Date.UTC(year, month - 1, 1));
  return `${normalized.getUTCFullYear()}-${String(normalized.getUTCMonth() + 1).padStart(2, "0")}`;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function formatMs(value) {
  if (!Number.isFinite(value) || value <= 0) return "0ms";
  if (value < 1000) return `${Math.round(value)}ms`;
  return `${(value / 1000).toFixed(2)}s`;
}

function usage() {
  return `Usage:
  node scripts/load-order-fill.mjs --source SOURCE.xlsx --blank BLANK.xlsx [options]

Options:
  --api URL              API base URL, default ${DEFAULT_API}
  --jobs N              Jobs to create, default ${DEFAULT_JOBS}
  --concurrency N       Active jobs at once, default ${DEFAULT_CONCURRENCY}
  --brand ID            Brand id, default ${DEFAULT_BRAND}
  --order-month YYYY-MM Order month, default next UTC month
  --blank PATH          Blank workbook. Repeat for split-blank brands.
  --poll-ms N           Poll interval, default ${DEFAULT_POLL_MS}
  --timeout-ms N        Timeout per job, default ${DEFAULT_TIMEOUT_MS}
  --preview             Fetch preview meta and first 50 rows for every output file
  --download-archive    Download the result archive for every successful job
  --no-report           Skip report fetch after completion
  --no-files            Skip file listing after completion`;
}

class UsageError extends Error {}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  try {
    await runLoad(parseArgs());
  } catch (error) {
    if (error instanceof UsageError) {
      console.log(error.message);
      process.exit(0);
    }
    console.error(error.message || error);
    process.exit(1);
  }
}
