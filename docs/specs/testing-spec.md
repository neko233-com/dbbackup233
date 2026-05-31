# Testing Spec

## Philosophy

`dbbackup233` must be tested as an operations tool, not only as a library.

Every critical feature needs one of:

- unit test
- integration test
- CLI dry-run verification
- Docker end-to-end test

## Required Test Matrix

### Config Validation

Must test:

- unknown source type fails
- unknown target type fails
- unknown job source fails
- unknown target reference fails
- MySQL version enum rejects unsupported values
- MySQL aliases normalize to enum values

### Manifest History

Must test:

- append record
- list newest first
- find latest by job
- find exact version by job
- missing version fails clearly

### MySQL Docker E2E

Must test through Docker:

- Docker MySQL starts on a non-3306 host port
- schema SQL loads successfully
- seed SQL loads successfully
- `mysqldump` creates gzip artifact
- local target receives artifact
- S3/OSS mock target receives upload
- uploaded body can be decoded and contains SQL dump
- restore writes into a different database
- source and restore database row counts match
- representative aggregate values match
- manifest has expected history records

MySQL fixture must include:

- integer
- bigint
- decimal
- double
- varchar
- text
- utf8mb4
- datetime/timestamp
- json
- blob
- enum
- set
- null
- indexes
- foreign keys

### File Backup

Must test:

- archive creation
- local target storage
- platform-specific archive command works

Future restore validation should extract archive and compare file content.

### HTTP Reporting

Must test:

- disabled reporter does not call endpoint
- enabled reporter sends `backup.started`
- enabled reporter sends `backup.completed`
- bearer token header is sent
- non-2xx reporting errors do not fail backup

### Upload Provider

Must test without cloud credentials:

- S3-compatible mock receives upload
- SDK sends a PUT object call
- AWS chunked streaming payload can be decoded
- decoded payload is valid gzip
- gzip contains expected dump SQL

Current tested SDK:

- `github.com/minio/minio-go/v7 v7.0.95`

### CLI Smoke Tests

Must verify:

- `backup --dry-run`
- `install all --dry-run`
- `update --tag vX.Y.Z --dry-run`
- `cron "0 2 * * *" --dry-run`
- `version`

### PostgreSQL

Current status:

- provider exists
- SQL fixtures exist
- integration test is pending

PostgreSQL Docker E2E should be added when ready:

- start PostgreSQL on non-5432 host port
- load `tests/sql/postgres_schema.sql`
- load `tests/sql/postgres_seed.sql`
- run `pg_dump`
- restore to another database
- compare data
- verify manifest

## Test Commands

Fast tests:

```bash
go test ./...
```

Full Windows validation:

```cmd
test.cmd
```

Docker MySQL only:

```bash
go test -tags docker_integration ./backup -run TestDockerMySQLBackupRestoreAndHistory -count=1 -v
```

