# Load Testing

Use this to measure how the gateway, Redis queue, document worker, PostgreSQL and
MinIO behave when many order-fill jobs arrive at the same time.

## Local Stand

```bash
docker compose -f deploy/docker-compose.yml up --build
```

One `document-worker` process handles one Redis message at a time. To test
parallel processing, scale the worker container; all replicas join the same
Redis consumer group:

```bash
docker compose -f deploy/docker-compose.yml up --build --scale document-worker=3
```

Increase the replica count gradually while watching CPU, memory, queue delay,
PostgreSQL and object storage. Each worker can inflate and parse large Excel XML
parts, serialize output files and build preview chunks; too many replicas can
exhaust shared resources before they improve throughput.

## Runner

The runner creates jobs through the public API, polls every job until
`needs_review`, `completed` or `failed`, then prints enqueue and completion
latency percentiles.

```bash
npm run load:order-fill -- \
  --source "testdata/private/source_100000.xlsx" \
  --blank "testdata/private/2026 08 25 Бланк заказа ANGIOPHARM.xlsx"
```

Same command through Make:

```bash
make load-order-fill ARGS='--source "testdata/private/source_100000.xlsx" --blank "testdata/private/2026 08 25 Бланк заказа ANGIOPHARM.xlsx"'
```

## Source Sizes

Generate fixed source sizes:

```bash
npm run generate:source:1k
npm run generate:source:5k
npm run generate:source:10k
```

Generate a benchmark series: `1000`, `5000`, `10000`, then every `5000` rows
up to the chosen maximum:

```bash
npm run generate:sources:bench -- --rows 30000
```

Measure one job per size:

```bash
for rows in 1000 5000 10000 15000 20000 25000 30000; do
  npm run load:order-fill -- \
    --jobs 1 \
    --concurrency 1 \
    --source "testdata/private/source_${rows}.xlsx" \
    --blank "testdata/private/2026 08 25 Бланк заказа ANGIOPHARM.xlsx"
done
```

## Scenarios

Baseline, one active job:

```bash
npm run load:order-fill -- \
  --jobs 5 \
  --concurrency 1 \
  --source "testdata/private/source_100000.xlsx" \
  --blank "testdata/private/2026 08 25 Бланк заказа ANGIOPHARM.xlsx"
```

Queue burst, 100 active client jobs:

```bash
npm run load:order-fill -- \
  --jobs 100 \
  --concurrency 100 \
  --source "testdata/private/source_100000.xlsx" \
  --blank "testdata/private/2026 08 25 Бланк заказа ANGIOPHARM.xlsx"
```

Preview read load after processing:

```bash
npm run load:order-fill -- \
  --jobs 20 \
  --concurrency 10 \
  --preview \
  --source "testdata/private/source_100000.xlsx" \
  --blank "testdata/private/2026 08 25 Бланк заказа ANGIOPHARM.xlsx"
```

Archive download load after processing:

```bash
npm run load:order-fill -- \
  --jobs 20 \
  --concurrency 10 \
  --download-archive \
  --source "testdata/private/source_100000.xlsx" \
  --blank "testdata/private/2026 08 25 Бланк заказа ANGIOPHARM.xlsx"
```

## What To Watch

- `enqueue latency`: gateway upload, S3 input writes, PostgreSQL job insert and
  Redis publish.
- `completion latency`: full time from create request to terminal job status.
  If this grows linearly while enqueue stays low, the queue is doing its job and
  the worker is the bottleneck.
- `failed`: any non-zero value needs logs from `gateway-service`, `job-service`, `file-service`, `document-worker`,
  PostgreSQL, Redis and MinIO before tuning further.
- Docker stats: aggregate document-worker RSS is the main guardrail for
  increasing the replica count.

## 100 Files At Once

With the current API, one order-fill job usually means two uploaded files:
source plus blank. A 100-file user burst is roughly 50 order-fill jobs, or fewer
if split-blank brands upload more than one blank per job.

The intended behavior is:

1. `gateway-service` accepts requests and stores input files through `file-service`.
2. Each accepted job is appended to the Redis stream `order-fill:jobs`.
3. `document-worker` containers read from the `document-service` consumer group.
4. A worker ACKs a message only after the job handler returns.
5. Extra jobs remain queued until a worker loop is free.

So the system should degrade as longer wait time, not duplicate processing. The
load test verifies that by comparing accepted jobs, terminal jobs and failures.
If a worker dies after taking a message, a later worker can reclaim that pending
message once it has been idle long enough.
