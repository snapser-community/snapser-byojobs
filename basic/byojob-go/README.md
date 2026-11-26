# BYOJob Go Basic Tutorial

A minimal worker for the Snapser [Jobs snap](https://snapser.com/docs/infra/snaps/jobs). Snapser sends your worker batches of user IDs, and the worker processes each user. This example logs every batch and reports every user as done. It is the starting point for your own data migrations, backfills and cleanups.

It also has an optional **failure mode**, so you can see how retries, failures and auto-pause behave before you write real code.

## Application
- The worker is in **main.go**. It uses only the Go standard library.
- **scripts/seed_users.sh** creates test users in your Snapend, so the job has users to process.
- The **Makefile** wraps the commands below.

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Returns `200` when the worker is ready. Snapser uses it to tell when the worker has started. |
| `POST /batch` | Receives `{"job_id", "run_id", "batch_id", "attempt", "items": [...]}` and returns `{"results": [...]}` for the users that did not succeed. A user you leave out counts as a success. |

See [The worker contract](https://snapser.com/docs/infra/snaps/jobs#the-worker-contract) for the full request, response and status-code rules.

## Pre-Requisites

### A. Install Go
Follow the setup [instructions](https://go.dev/dl/) to install Go 1.22 or later on your machine.

To confirm your setup is working run
```bash
go version
```

### B. Snapctl Setup
You need a working setup of snapctl, Snapser's CLI tool. Follow the step by step [tutorial](https://snapser.com/docs/guides/tutorials/setup-snapctl) if you do not have snapctl installed. Make sure your snapctl version has the `byojob` command. Run the following to confirm your setup:

```bash
# Validate your snapctl setup
snapctl validate
# Confirm the byojob command is available
snapctl byojob --help
```

### C. Docker
Make sure Docker engine is running on your machine. Open Docker Desktop settings and make sure **Use containerd for pulling and storing images** is **disabled**.

### D. A Snapend with the Jobs snap
You need a Snapend with:
- The **Jobs** snap. Jobs is a Premium snap, so you need to be on a Premium plan.
- The **Auth** snap with **anonymous login** enabled, if you want to use the seed script below.

## Running the worker locally

```bash
make run            # start the worker on :8080
make smoke          # in another shell: calls /health and sends one /batch
make test           # unit tests
```

Each batch prints one JSON log line, with running totals:

```json
{"level":"INFO","msg":"batch","run_id":"r1","batch_id":"b1","attempt":1,"items":3,"failed":0,"first":"u1","total_batches":1,"total_items":3,"total_failed":0}
```

To build and run the same image that snapctl publishes:

```bash
make build          # docker build for linux/arm64
make docker-run     # run that image on :8080
```

## Running it on Snapser

### 1. Seed users
A job runs over the users in your Snapend's Auth snap. If your Snapend has no users yet, create some:

```bash
./scripts/seed_users.sh $snapend_id          # 100 new users
./scripts/seed_users.sh $snapend_id 500      # 500 more
```

Every run uses a new username prefix (`byojob_<timestamp>`), so each run adds users. Run `./scripts/seed_users.sh -h` for all options, including `GATEWAY`, `PREFIX`, `START` and `CONCURRENCY`.

### 2. Publish the worker

```bash
make publish tag=my-worker-v1
# which runs:
snapctl byojob publish --tag my-worker-v1 --path .
```

Pass `profile=<name>` to use a snapctl profile other than your default. Tags can be up to 56 characters. Use a new tag for each version, such as `my-worker-v2`.

### 3. Create a job
In the Snapser Web App, open your Snapend, go to the [Jobs tool](https://snapser.com/docs/infra/snaps/jobs/config-tools/jobs) and click **Add Job**. Set the worker **Image tag** to `my-worker-v1`. The default settings work for this example.

### 4. Test it on a few users

```bash
snapctl byojob job-list --snapend-id $snapend_id
snapctl byojob job-test --snapend-id $snapend_id --job-id $job_id --sample-size 10
```

### 5. Run it and watch it

```bash
snapctl byojob job-run --snapend-id $snapend_id --job-id $job_id
snapctl byojob run-status --snapend-id $snapend_id --job-id $job_id --run-id $run_id --watch
```

You can also follow the run in the Web App under the [Runs tool](https://snapser.com/docs/infra/snaps/jobs/config-tools/runs).

## Failure mode

To see retries, failures and `run-rerun-failures` in action, add these environment variables to the job's worker. The same image works with and without them.

| Variable       | Value                                          |
|----------------|------------------------------------------------|
| `FAILURE_MODE` | `true` to turn failures on (off by default)    |
| `FAILURE_RATE` | optional, 0 to 1; default `0.1` (10% of users) |

Each user then fails with that probability:
- Half of the failed users answer `retry`. Snapser retries them with backoff, so most succeed on a later attempt.
- The rest answer `failed` and land in the run's failure list. See them with `snapctl byojob run-failures`, and start a new run over only those users with `snapctl byojob run-rerun-failures`.

A 10% rate stays under the default 50% auto-pause error rate. Set `FAILURE_RATE` above the job's auto-pause error rate to see the run pause itself.

To try failure mode locally:

```bash
make run-failing                      # 10% of users fail
FAILURE_RATE=0.3 make run-failing     # 30% of users fail
```

## Writing your own worker
Replace the `failures(...)` call in the `/batch` handler with your own work, and return a `retry` or `failed` result for each user that did not succeed. Keep these in mind:

- **Make it idempotent.** Snapser delivers every user at least once, so processing the same user twice must be safe.
- **Answer within the batch timeout** (30 seconds by default). Lower the batch size if batches take too long.
- **Return `429`** if a service you depend on is rate-limiting you. The run slows down without counting it as an error.
- **Call your snaps** with the `SNAPEND_<SNAP>_<PORT>_URL` variables and the `SNAPEND_INTERNAL_HEADER_NAME` / `SNAPEND_INTERNAL_HEADER` header, which Snapser sets on every worker.

See [What makes a good worker](https://snapser.com/docs/infra/snaps/jobs#what-makes-a-good-worker) for more.
