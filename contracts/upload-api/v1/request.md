# Upload API v1 request

Official companions upload a scan with:

```http
POST /v1/scans
Authorization: Bearer <installation-token>
Content-Type: application/gzip
```

The request body is a scan archive v1 gzip stream. The compressed body limit is
16 MiB and the decompressed canonical JSON limit is 128 MiB. The bearer token
identifies an installation; clients must never place it in query parameters,
logs, configuration files, or fixtures.

Clients must send the stored archive bytes without reserializing them. Servers
authenticate before decoding, reject malformed gzip and non-canonical or
invalid scans, and use the canonical-payload SHA-256 digest for idempotency.
