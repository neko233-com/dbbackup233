# dbbackup233

Docs: https://neko233-com.github.io/dbbackup233/

`dbbackup233` is an agent-first Backup Center CLI for game projects: one binary
plus one YAML config gives any Windows/Linux/macOS machine a repeatable backup,
restore, verification, reporting, and scheduling workflow.

Current MVP scope:

- MySQL physical backup through Percona XtraBackup, with full and incremental
  modes for large game databases
- MySQL logical backup/restore through official `mysqldump` and `mysql` for
  small databases and compatibility
- PostgreSQL provider through `pg_dump` and `pg_restore`, with Docker E2E
- File backup/restore through platform-native archive tools
- Local backup target first; S3/TOS/OSS provider remains in the library for
  later rollout
- Manifest-based backup history and versioned restore
- Restore-time SHA-256 integrity verification before any write
- One-key restore for logical dumps and opt-in one-key physical MySQL restore
- Optional HTTP reporting for backup start/completion
- Cross-platform install script, hot-update-next-run `dbbackup233 update`,
  `dbbackup233 cron`,
  `dbbackup233 verify`, and `dbbackup233 prune`
- Release binaries for Linux/macOS/Windows on amd64 and arm64

## Specs

This project uses spec-driven development:

- [Backup Center Spec](docs/specs/backup-center-spec.md)
- [Testing Spec](docs/specs/testing-spec.md)
- [Implementation Plan](docs/specs/implementation-plan.md)
- [Operator Runbook](docs/operator-runbook.md)
- [Benchmark Guide](docs/benchmark.md)
- [Technical Architecture](docs/technical-architecture.md)
- [Agent-First Product](docs/agent-first.md)
- [Restore Guide](docs/restore.md)

## Quick Start

```bash
dbbackup233 backup -c config.yaml
dbbackup233 backup -c config.yaml --job mysql-game --job server-files
dbbackup233 list -c config.yaml
dbbackup233 restore mysql-game --version 20260531-120000 -c config.yaml
dbbackup233 prune -c config.yaml --dry-run
```

Install required database dump tools:

```bash
dbbackup233 install mysqldump
dbbackup233 install xtrabackup
dbbackup233 install pg_dump
```

## Install dbbackup233

Windows PowerShell:

```powershell
iwr https://raw.githubusercontent.com/neko233-com/dbbackup233/main/scripts/install.ps1 -UseB | iex
```

Linux/macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/neko233-com/dbbackup233/main/scripts/install.sh | sh
```

Upgrade:

```bash
dbbackup233 update
dbbackup233 update --check
```

`update` replaces the installed binary for the next process run. A backup that
is already running keeps using its current process; the next manual run or cron
run uses the new binary.

## Scheduling

Preview a schedule:

```bash
dbbackup233 cron "0 2 * * *" -c config.yaml --dry-run
```

Install it on the machine:

```bash
dbbackup233 cron "0 2 * * *" -c config.yaml --install
```

Windows uses `schtasks`. Linux and macOS use `crontab`.

Inspect or remove a schedule:

```bash
dbbackup233 cron list
dbbackup233 cron remove --dry-run
dbbackup233 cron remove
```

## Config

The committed [config.yaml](config.yaml)
is the shared spec template. Local secrets and ports should live in ignored
`config.local.yaml`.

MySQL version is an enum:

- `mysql57`
- `mysql80`

For large game databases, use XtraBackup instead of `mysqldump`:

```yaml
sources:
  - name: "mysql-game-full"
    type: mysql
    mysql:
      host: "127.0.0.1"
      port: 3306
      user: "backup"
      password: "${MYSQL_BACKUP_PASSWORD}"
      database: "game"
      version: "mysql80"
      mode: "full"
      physical_engine: "xtrabackup"
      physical_artifact_format: "zip" # zip or tar.gz
      xtrabackup_tool: "xtrabackup"
      xtrabackup_dir: "./backups/xtrabackup"
      xtrabackup_restore_dir: "./restore/mysql-xtrabackup"
      restore_datadir: "" # set for opt-in one-key copy-back
      restore_stop_command: []
      restore_start_command: []
      restore_fix_ownership_command: []

  - name: "mysql-game-incremental"
    type: mysql
    mysql:
      host: "127.0.0.1"
      port: 3306
      user: "backup"
      password: "${MYSQL_BACKUP_PASSWORD}"
      database: "game"
      version: "mysql80"
      mode: "incremental"
      physical_engine: "xtrabackup"
      physical_artifact_format: "zip" # zip or tar.gz
      xtrabackup_tool: "xtrabackup"
      xtrabackup_dir: "./backups/xtrabackup"
      incremental_base_dir: "" # optional; empty uses xtrabackup_dir/latest-base.txt
      xtrabackup_restore_dir: "./restore/mysql-xtrabackup"
      restore_datadir: ""
```

Physical MySQL artifacts use `.xb.zip` by default. Set
`physical_artifact_format: "tar.gz"` for `.xb.tar.gz`. Restore extracts and
prepares the backup into `xtrabackup_restore_dir`; operator then stops MySQL and
runs `xtrabackup --copy-back --target-dir=<restore-dir>` manually.
Each successful XtraBackup updates `xtrabackup_dir/latest-base.txt`, so the next
incremental job can chain from the newest local physical backup.

Set `restore_datadir` plus stop/start/ownership commands for opt-in one-key
physical restore. Without `restore_datadir`, restore prepares the backup only
and never touches the live MySQL datadir.

One config can submit many backup jobs in one command:

```yaml
jobs:
  - name: "mysql-game"
    source: "mysql-game"
    targets: ["local"]
  - name: "server-files"
    source: "server-files"
    targets: ["local"]
```

## History And Restore

Every successful backup appends one JSON line to `manifest_path`. Restore uses
that history:

```bash
dbbackup233 list -c config.yaml
dbbackup233 restore mysql-game --version 20260531-120000 -c config.yaml
```

Without `--version`, restore uses the latest artifact for that job.
Before restore writes data, the local artifact SHA-256 must match the manifest.

## HTTP Reporting

Reporting is optional:

```yaml
report:
  enabled: true
  url: "https://backup-center.example.com/api/backup/events"
  token: "${BACKUP_CENTER_TOKEN}"
```

Your server only needs to implement:

```http
POST /api/backup/events
```

See [docs/http-reporting.md](docs/http-reporting.md)
for payload examples.

## Upload Provider

Cloud uploads use the S3-compatible API through `github.com/minio/minio-go/v7
v7.0.95`. `type: oss`, `type: tos`, and `type: s3` all route to that provider.
The automated test uses a local S3-compatible mock server, verifies the upload
interface is called, and checks the uploaded gzip dump content.

See [docs/upload-provider.md](docs/upload-provider.md).

## Automated Tests

Normal unit tests:

```bash
go test ./...
```

Docker integration test. It starts MySQL on an automatically selected non-3306
host port. Override with `DBBACKUP233_DOCKER_MYSQL_PORT` if needed. The test
backs up `game_src`, restores into `game_restore`, compares data, verifies file
backup, checks manifest history, and validates the mock OSS/S3 upload path:

```powershell
.\scripts\test-docker-mysql.ps1
```

or:

```bash
./scripts/test-docker-mysql.sh
```

PostgreSQL Docker E2E is included in `test.cmd`.

Benchmark MySQL backup time:

```powershell
.\scripts\bench-docker-mysql.ps1 -Rows 50000 -PayloadBytes 256
```

## Release

GitHub Actions follow the `unicli` style: CI verifies format/vet/build/test and
release builds Linux/macOS/Windows binaries on `v*` tags.

Local platform matrix check:

```powershell
.\scripts\check-platform-matrix.ps1
```
