# golang-s3-batch-uploader

Batch-processes a directory of CSV files with a bounded worker pool and
uploads each valid file to S3. Hexagonal architecture
(`handler/service/repository`), same conventions as the other projects in
this workspace.

## Run

```
docker-compose up
```

Brings up LocalStack (S3 on `localhost:4566`) and the app. No AWS account
needed - LocalStack gives a real S3 API locally, and the app talks to it
through the same AWS SDK code path it would use against real AWS (only the
endpoint differs).

## Endpoint

```
POST /batches   {"source_dir": "/data"}
```

```json
{
  "code": 200,
  "message": "batch processed",
  "data": {
    "succeeded": [{ "file": "a.csv" }],
    "failed": [{ "file": "b.csv", "error": "..." }]
  }
}
```

`code` mirrors the HTTP status, `data` is `null` when empty. `docker-compose.yml`
mounts `./testdata` into the app container at `/data` -
`testdata/` already has two valid CSVs and one malformed one, so:

```
curl -X POST localhost:8080/batches -d '{"source_dir":"/data"}'
```

returns `good1.csv` and `good2.csv` in `succeeded` and `bad.csv` in `failed`
in one response - the DoD's partial-failure behavior, live.

## Design notes

- **Worker pool, not unbounded goroutines**: a fixed-size pool (`batch.worker_count`
  in `config.yaml`, default 4) reads jobs from a channel, one worker per file.
  Unbounded goroutines-per-file would let a directory with thousands of files
  open thousands of file descriptors and S3 connections at once; a bounded
  pool caps concurrent I/O to a size the box and the S3 rate limits can
  actually sustain. The tradeoff: too small a pool under-uses available
  throughput, too large one risks the same resource exhaustion the pool
  exists to avoid - `batch.worker_count` is deliberately a config value, not
  a constant, so it can be tuned per environment without a code change.
- **Partial failure is a first-class result, not an exception**: each worker
  reports a per-file success or failure on a results channel; the service
  collects both into `BatchResult{Succeeded, Failed}` rather than stopping
  the batch on the first error. One malformed CSV or one failed upload never
  blocks the other files - see `service/batch_service_test.go` and
  `batch_integration_test.go` for both failure shapes (a bad CSV and a
  failed upload) proven independently.
- The S3 adapter (`repository/uploader_s3.go`) uses `aws-sdk-go-v2` with
  path-style addressing when an endpoint override is set (LocalStack
  requires this) and creates the target bucket if it doesn't exist yet, so a
  fresh `docker-compose up` needs no manual bucket setup.
- Not done on purpose: no retry/backoff on upload failure, no resumable
  batches, no per-file size limit.

## Tests

```
go test ./...
go generate ./...   # regenerate repository mocks
```

`go test ./...` runs standalone, no LocalStack or network required - the
`Uploader` port is mocked (`service/batch_service_test.go`) or faked
in-memory (`batch_integration_test.go`).
