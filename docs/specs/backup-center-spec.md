# Backup Center Spec

## Purpose

`dbbackup233` is the game-project Backup Center agent. It must let any game
server machine gain backup and restore capability with one CLI binary and one
config file.

The tool is optimized for small and mid-size game teams that often deploy game
logic services and databases on the same host for cost, latency, and operational
simplicity.

## Product Goals

1. One command backs up all configured local assets.
2. One command restores a selected historical version.
3. Operators can install, update, schedule, and verify the agent on
   Windows/Linux/macOS.
4. Backup status can be reported to a central game operations system through a
   simple HTTP API.
5. The CLI can run without cloud vendor lock-in. Local backup must be first
   class. Cloud upload is an output target.

## Current MVP Scope

Supported backup sources:

- MySQL via official `mysqldump` and restore via official `mysql`
- PostgreSQL via official `pg_dump` and restore via official `pg_restore`
- Files via platform archive tools

Supported targets:

- Local directory
- S3-compatible upload provider, including OSS/TOS/S3 naming aliases

Supported operations:

- `backup`
- `restore`
- `list`
- `install`
- `update`
- `cron`
- `version`

## Explicit Non-Goals For MVP

- Native cloud snapshot APIs
- Hot physical MySQL backup through XtraBackup as default behavior
- PostgreSQL integration test without a PostgreSQL environment
- MongoDB/Redis production support
- Central Backup Center server implementation

MongoDB, Redis, config repository backup, XtraBackup, and `pg_basebackup` remain
extension points, but they should not block the MySQL/files production path.

## Configuration Model

The config is the public product contract.

```yaml
defaults:
  compress: gzip
  concurrency: 1
  work_dir: "./backups"
  manifest_path: "./backups/manifest.jsonl"

report:
  enabled: false
  url: "https://backup-center.example.com/api/backup/events"

sources:
  - name: "mysql-game"
    type: mysql

targets:
  - name: "local"
    type: local

jobs:
  - name: "mysql-game"
    source: "mysql-game"
    targets: ["local"]
```

Rules:

- `sources` define inputs.
- `targets` define outputs.
- `jobs` bind one source to one or more targets.
- A single `backup` command may execute multiple jobs.
- `defaults.concurrency` controls job-level parallelism.
- Default concurrency is `1` to avoid surprise database pressure.

## MySQL Version Enum

MySQL version must be an enum:

- `mysql57`
- `mysql80`

Accepted aliases can be normalized internally:

- `5.7` and `57` -> `mysql57`
- `8.0` and `80` -> `mysql80`

Unknown versions must fail validation.

## Backup Semantics

### MySQL

Default mode:

- `mysqldump`
- `--single-transaction`
- `--quick`
- `--default-character-set=utf8mb4`

The provider must support:

- logical dump
- restore to the original database
- restore to a different database with `restore_database`
- `no_create_db` for cross-database restore validation

### PostgreSQL

Default mode:

- `pg_dump`
- `format: custom`
- restore through `pg_restore`

PostgreSQL is provider-ready but integration testing is pending until a test
environment exists.

### Files

File backups must use platform-available tools:

- Windows: PowerShell `Compress-Archive` / `Expand-Archive`
- Linux/macOS: `tar`

## Restore Semantics

Each successful backup writes a manifest record. Restore must resolve artifacts
from manifest history.

Rules:

- `restore JOB` restores the latest version for that job.
- `restore JOB --version VERSION` restores the exact version.
- If history is missing, fail with a clear error.
- Restore must never guess from directory scanning unless an explicit future
  feature adds index rebuilding.

## History Contract

Manifest format is JSON Lines.

Each record must contain:

- version
- job name
- source name
- source type
- local artifact path
- object name
- artifact size
- created timestamp
- target names

The manifest is the local source of truth for versioned restore.

## HTTP Reporting Contract

Reporting is optional and must not break the backup if the HTTP endpoint fails.

Events:

- `backup.started`
- `backup.completed`

Completion statuses:

- `success`
- `failed`
- `dry-run`

Server contract:

```http
POST /api/backup/events
Content-Type: application/json
Authorization: Bearer <token>
```

Any `2xx` response means accepted.

## Upload Provider Contract

Cloud upload is implemented through S3-compatible APIs.

Current SDK:

- `github.com/minio/minio-go/v7 v7.0.95`

Aliases:

- `s3`
- `oss`
- `tos`

Automated tests must validate upload API invocation and uploaded gzip content
without requiring real cloud credentials.

## Platform Contract

The CLI must support:

- Windows
- Linux
- macOS

Required platform commands:

- install script
- update command
- cron/scheduled task command
- test script

Release artifacts must include:

- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64
- windows/arm64

## Reliability Requirements

- Backups must fail loudly if any job fails.
- Manifest append happens only after artifact creation and target storage
  complete.
- Default concurrency must be conservative.
- Secrets must be supplied through env vars or ignored local config files.
- Upload/report failures must be observable in logs.
- Restore must be explicit and version-aware.
