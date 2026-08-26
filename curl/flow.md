# Manual test flow

Full walkthrough for exercising the API by hand, from starting the stack to
tearing it down.

## Start

```bash
docker compose up -d --build
docker compose ps
docker compose logs app --tail 20   # should show "server started on port 8080"
```

## 1. Run the batch

`docker-compose.yml` mounts `./testdata` into the app container at `/data`.
It already has two valid CSVs (one with stray whitespace) and one malformed
one:

```bash
curl -X POST http://localhost:8080/batches \
  -H "Content-Type: application/json" \
  -d '{"source_dir":"/data"}'
```

```json
{
  "code": 200,
  "message": "batch processed",
  "data": {
    "succeeded": [{ "file": "good1.csv" }, { "file": "good2.csv" }],
    "failed": [
      {
        "file": "bad.csv",
        "error": "parse error on line 2, column 21: extraneous or missing \" in quoted-field"
      }
    ]
  }
}
```

One malformed file never stops the rest of the batch - that is the point.

## 2. Confirm the objects actually landed in S3 (LocalStack)

```bash
docker compose exec localstack awslocal s3 ls s3://batch-uploads/
```

```
2026-08-26 06:34:12         20 good1.csv
2026-08-26 06:34:12         14 good2.csv
```

`bad.csv` never reaches the uploader - it isn't listed.

## 3. Confirm the upload is processed, not a byte-for-byte copy

`good1.csv` on disk has stray whitespace around fields. Compare it against
what actually landed in the bucket:

```bash
docker compose exec localstack awslocal s3 cp s3://batch-uploads/good1.csv /tmp/good1.csv
docker compose exec localstack cat /tmp/good1.csv
```

```
id,name
1,foo
2,bar
```

The uploaded object has every field trimmed - `service/csv.go`'s
`processCSV` normalized it before upload.

## 4. Rejection case: missing `source_dir`

```bash
curl -X POST http://localhost:8080/batches \
  -H "Content-Type: application/json" \
  -d '{}'
```

```json
{ "code": 422, "message": "source_dir is required", "data": null }
```

## Stop

```bash
docker compose down -v
```
