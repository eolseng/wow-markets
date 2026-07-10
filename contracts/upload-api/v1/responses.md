# Upload API v1 responses

An accepted scan returns HTTP `201`; a checksum duplicate returns HTTP `200`.
Both use this shape:

```json
{"checksum":"<64 lowercase hex characters>","items":2,"price_levels":2,"price_snapshots":2,"rows":2,"scan_id":42,"status":"accepted","submission_id":"<uuid>"}
```

`status` is `accepted` or `duplicate`. The client must verify that `checksum`
matches the queued archive before treating the upload as successful. Counts and
identifiers are informational server results.

Errors are JSON objects with `error` and an optional `detail`. Stable error
codes include `unauthenticated`, `authentication_failed`, `invalid_gzip`,
`scan_too_large`, `invalid_scan`, `truncated_scan`, `submission_failed`, and
`ingestion_failed`.

- `401` authentication errors require token replacement and are not retried
  automatically.
- `400`/`413`/`422` archive or contract failures are terminal for that archive.
- `408`, `425`, `429`, and `5xx` failures are retryable with bounded backoff.
- Network and offline failures are retryable and must not interrupt archiving.
