# golang-s3-batch-uploader

Batch-processes a directory of CSV files with a bounded worker pool,
normalizes each one, and uploads the processed output to S3.

## Run

```
docker-compose up
```

Brings up LocalStack (S3 on `localhost:4566`) and the app - no AWS account
needed.

## Endpoint

```
POST /batches
```

See `curl/flow.md` for full request/response examples, including the
partial-failure case (`testdata/` has two valid CSVs and one malformed
one).

## Key Technical Takeaways / Gotchas

- Worker pool, not unbounded goroutines: fixed size (`batch.worker_count`
  in `config.yaml`, default 4) so a directory with thousands of files
  can't open thousands of file descriptors/S3 connections at once.
  Config-driven, not a constant, so it's tunable per environment.
- Partial failure is a first-class result, not an exception: each worker
  reports success/failure per file onto a results channel; the service
  collects both into `BatchResult{Succeeded, Failed}` instead of aborting
  on the first error.
- The uploaded object is not the input file verbatim: `service/csv.go`'s
  `processCSV` parses every row and trims whitespace before rewriting -
  the object in S3 is normalized output, not a byte copy.
- S3 adapter (`repository/uploader_s3.go`) uses `aws-sdk-go-v2` with
  path-style addressing when an endpoint override is set (LocalStack
  needs this). Bucket auto-create (`s3.auto_create_bucket`) is opt-in,
  off by default - real AWS only needs `s3:PutObject`, not bucket-admin
  permissions.

## Not done on purpose

- No retry/backoff on upload failure, no resumable batches, no per-file
  size limit.

## Tests

```
go test ./...
go generate ./...   # regenerate repository mocks
```

`go test ./...` runs standalone, no LocalStack or network required - the
`Uploader` port is mocked or faked in-memory.

See `curl/flow.md` for a manual walkthrough against LocalStack.
