# Scan archive format 1

A scan archive is a gzip stream whose decompressed payload is exactly one
canonical UTF-8 JSON object. The payload has no byte-order mark, leading or
trailing whitespace, or trailing newline. JSON object keys appear in the order
shown by `fixtures/valid.json`; integers use decimal notation and strings use
Go `encoding/json` escaping. Unknown or missing fields are rejected.

The canonical payload is the normalized form of one ready SavedVariables
format 5 scan. `rows` expands item references so each listing is self-contained.
The scan must pass the field, count, location, completeness, and duplicate
source-row validation described by the SavedVariables specification.

The archive identity is the lowercase hexadecimal SHA-256 digest of the
decompressed canonical JSON bytes. Local archive filenames are
`<digest>.json.gz`. The gzip container checksum is recorded for fixture
integrity but is not the scan identity and may vary across conforming encoders.

`fixtures/valid.json`, `fixtures/valid.json.gz`, and `checksums.txt` are
authoritative golden material. Regenerate them from the SavedVariables fixture
with:

```sh
cd companion
go run ./cmd/generate-contract-fixtures
```
