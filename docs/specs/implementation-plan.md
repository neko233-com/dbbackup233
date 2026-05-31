# Implementation Plan

## Recommendation

Prioritize a stable MySQL + files Backup Center agent before expanding provider
surface area.

Recommended production path:

1. Local backup and restore with manifest history.
2. MySQL Docker-tested restore into a different database.
3. HTTP reporting into the future Backup Center server.
4. S3-compatible upload with one real vendor smoke test per deployment team.
5. Cross-platform install/update/cron.
6. PostgreSQL Docker E2E.
7. Retention, pruning, encryption, and checksum verification.
8. MongoDB/Redis/config repository providers.

## Phase 0: Spec Lock

Status: mostly complete.

Deliverables:

- product spec
- testing spec
- implementation plan
- committed `config.yaml`
- ignored `config.local.yaml`
- SQL fixtures

Exit criteria:

- README points to specs.
- `test.cmd` passes.

## Phase 1: MySQL + Files Production MVP

Status: current focus.

Deliverables:

- MySQL logical backup through official `mysqldump`
- MySQL restore through official `mysql`
- restore to alternate database
- files backup
- local target
- manifest history
- `list`
- `restore`
- Docker E2E
- HTTP reporting
- install/update/cron

Exit criteria:

- `cmd /c test.cmd` passes on Windows.
- Docker E2E validates MySQL and files.
- A real game server can run `backup`, `list`, and `restore --dry-run`.

## Phase 2: Backup Center Integration

Deliverables:

- server-side API implementation for `POST /api/backup/events`
- dashboard data model
- token auth
- job status view
- last success / last failure view
- artifact metadata view

CLI requirements:

- retry policy for reports
- optional machine ID
- optional project ID
- clear logs for report failures

Exit criteria:

- report events are visible in the Backup Center server.
- failed report does not fail backup.

## Phase 3: Cloud Upload Hardening

Deliverables:

- real OSS smoke test documentation
- real TOS smoke test documentation
- object checksum metadata
- optional upload retry count
- optional multipart threshold config
- retention policy documentation

Exit criteria:

- one real OSS/TOS/S3 upload is verified outside CI.
- local mock upload test remains deterministic in CI.

## Phase 4: PostgreSQL

Deliverables:

- Docker PostgreSQL integration test
- `pg_dump` custom format backup
- restore into another database
- row count and aggregate comparison
- optional plain SQL mode

Exit criteria:

- PostgreSQL Docker E2E passes.
- PostgreSQL config docs are validated.

## Phase 5: Operational Safety

Deliverables:

- `verify` command for config/tool availability
- `doctor` command for environment diagnostics
- backup checksum file
- artifact pruning by count/days
- optional encryption before upload
- restore confirmation prompts for destructive restores

Exit criteria:

- operator can run one command before enabling cron and see readiness status.

## Phase 6: Provider Expansion

Only after phases 1-5 are stable:

- XtraBackup provider
- `pg_basebackup`
- Redis RDB provider
- MongoDB provider
- config repository backup provider

Each new provider must ship with:

- config spec
- Docker or deterministic integration test
- backup test
- restore test
- manifest test
- reporting test coverage

## What To Land Next

Recommended immediate tasks:

1. Add `verify` command.
2. Add file restore extraction test.
3. Add checksum field to manifest.
4. DONE: Add PostgreSQL Docker E2E.
5. Add real install-script CI smoke on Windows/Linux/macOS.
6. Add upload retry config.
7. Add `retention` config and prune command.

## Recommended Landing Backlog

### P0: Must Land Before Real Server Rollout

- DONE: `verify` command: validate tools, config, target paths, report URL, and
  writable backup directories.
- DONE: file restore extraction test: current Docker E2E verifies file backup; restore
  extraction should also be asserted.
- DONE: manifest checksum: add SHA-256 for every local artifact.
- DONE: restore safety: require `--yes` or `--dry-run` for restore commands that write
  to a database.
- DONE: Windows/Linux/macOS install smoke in CI.
- DONE: README operator runbook: install, configure, dry-run, backup, list, restore,
  schedule, update.

### P1: Should Land Before Backup Center Server Integration

- DONE: HTTP report retry with bounded backoff.
- DONE: machine/project identity fields in report payload.
- report server OpenAPI spec.
- log format stabilization for operations parsing.
- DONE: upload retry config.
- TODO: upload per-request timeout config.
- DONE: `retention` config and `prune` command.
- DONE: PostgreSQL Docker E2E.

### P2: Land After MySQL Path Has Production Mileage

- encryption at rest before upload.
- native checksum sidecar file.
- XtraBackup provider.
- `pg_basebackup` provider.
- Redis provider.
- MongoDB provider.
- central server dashboard.

## What Not To Land Yet

Do not prioritize these before the production MVP is stable:

- MongoDB production provider
- Redis production provider
- XtraBackup default mode
- native cloud SDKs
- complex Backup Center server UI
- encrypted backup format

These are useful, but they increase operational risk before the core MySQL
restore path has enough production mileage.
