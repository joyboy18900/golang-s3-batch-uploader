# golang-s3-batch-uploader

Batch-processes a directory of CSV files with a bounded worker pool,
normalizes each one, and uploads the processed output to S3.

## Run

```
docker-compose up
```

Brings up LocalStack (S3 on `localhost:4566`) and the app. No AWS account
needed.

## Endpoint

```
POST /batches
```

See `curl/flow.md` for full request/response examples, including the
partial-failure case (`testdata/` has two valid CSVs and one malformed
one).

## Tests

```
go test ./...
go generate ./...   # regenerate repository mocks
```

`go test ./...` runs standalone, no LocalStack or network required. The
`Uploader` port is mocked or faked in-memory.

See `curl/flow.md` for a manual walkthrough against LocalStack.
