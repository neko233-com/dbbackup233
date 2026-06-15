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
- restore rejects checksum mismatch before provider execution

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

### MySQL XtraBackup Docker E2E

Must test through Docker:

- large synthetic InnoDB table loads successfully
- XtraBackup full backup creates `.xb.zip` artifact by default
- XtraBackup incremental backup uses prior local base
- `xtrabackup_checkpoints` exists in latest base directory
- manifest records both full and incremental artifacts with SHA-256
- concurrent writes continue during XtraBackup
- Docker sidecar can run `percona/percona-xtrabackup:8.0`
- copy-back command includes `--copy-back`, `--target-dir`, and `--datadir`

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
- restore extracts archive and compares file content

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
- `backup --job NAME --dry-run`
- unknown `backup --job NAME` fails clearly
- `install all --dry-run`
- `update --tag vX.Y.Z --dry-run`
- `update --tag vX.Y.Z --check`
- `cron "0 2 * * *" --dry-run`
- `cron remove --dry-run`
- `version`

### Cron

Must test:

- Windows `schtasks /Create` command generation
- Windows hourly schedule generation
- platform-specific list command generation
- platform-specific remove command generation

### Platform Matrix

CI must verify native smoke tests on:

- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64
- windows/arm64

Release must build all six artifacts.

### PostgreSQL

Current status:

- provider exists
- SQL fixtures exist
- integration test is pending

PostgreSQL Docker E2E must verify:

- start PostgreSQL on non-5432 host port
- load `tests/sql/postgres_schema.sql`
- load `tests/sql/postgres_seed.sql`
- run `pg_dump`
- restore to another database
- compare data
- verify manifest

### Additional Providers

Must test:

- Redis RDB copy command path
- Elasticsearch snapshot request builder and receipt artifact
- ClickHouse backup/restore query builders
- provider config defaults and validation for Elasticsearch and ClickHouse

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

Docker MySQL XtraBackup:

```bash
go test -tags "docker_integration docker_benchmark docker_xtrabackup" ./backup -run TestDockerMySQLXtraBackupFullIncrementalAndWrites -count=1 -v
```
