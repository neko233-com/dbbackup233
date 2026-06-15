# Technical Architecture

## Agent Runtime

`dbbackup233` runs as a local agent on the game server. It reads one YAML config,
executes selected jobs, records manifest history, verifies checksums before
restore, and reports lifecycle events to an optional HTTP endpoint.

Flow:

1. Load config and apply defaults.
2. Select jobs.
3. Run providers with conservative concurrency.
4. Write local artifact.
5. Compute SHA-256.
6. Store to local/S3-compatible targets.
7. Append manifest JSONL.
8. Emit HTTP report events.

## MySQL Logical Backup

`mode: mysqldump` uses official MySQL tools:

- `mysqldump --single-transaction --quick`
- gzip artifact: `.sql.gz`
- restore through `mysql`

This mode is useful for small databases, schema portability, and simple restore
into another database.

## MySQL Physical Backup

`mode: full` and `mode: incremental` are dbbackup233 product modes. The default
physical engine is:

```yaml
physical_engine: "xtrabackup"
```

The engine remains explicit because hot physical backup correctness depends on
InnoDB internals: tablespaces, redo log, LSN, binlog position, and crash
recovery. dbbackup233 owns orchestration, artifact layout, manifest, integrity,
targets, scheduling, reporting, and restore workflow.

Default artifact:

- `.xb.zip`

Optional artifact:

- `.xb.tar.gz`

Incremental chain:

- Each successful physical backup writes `xtrabackup_dir/latest-base.txt`.
- Incremental backup uses that latest base when `incremental_base_dir` is empty.
- Explicit `incremental_base_dir` can pin a base.

## Restore Safety

All restore paths verify SHA-256 from the manifest before provider restore
begins.

Physical restore has two modes:

- prepare only: extract and run `xtrabackup --prepare`
- one-key restore: stop service, prepare, copy-back, fix ownership, start
  service

One-key physical restore is opt-in through `restore_datadir`. Without it,
dbbackup233 never writes to the MySQL datadir.

## Targets

Targets are output sinks:

- `local`
- `s3`
- aliases: `oss`, `tos`

S3-compatible upload uses `github.com/minio/minio-go/v7`.

## Provider Matrix

| Source | Full Backup | Incremental / Native Delta | Restore |
| --- | --- | --- | --- |
| MySQL | XtraBackup full or mysqldump | XtraBackup incremental | prepare + optional copy-back, or mysql import |
| PostgreSQL | `pg_dump` custom/plain | planned `pg_basebackup` incremental chain | `pg_restore` or `psql` |
| Redis | RDB copy or `BGSAVE` then copy | Redis AOF/RDB policy external today | copy RDB to restore path |
| MongoDB | `mongodump --archive --gzip` | oplog/PITR planned | `mongorestore --archive --gzip` |
| Elasticsearch | snapshot API | snapshot repository is incremental internally | snapshot restore API |
| ClickHouse | `BACKUP ... TO ...` | `BACKUP ... SETTINGS base_backup = ...` | `RESTORE ... FROM ...` |
| Files | zip or tar.gz | planned changed-file index | archive extract |

## Manifest

Manifest is JSON Lines. Each successful artifact records:

- version
- job name
- source name/type
- local artifact path
- object name
- size
- SHA-256
- created timestamp
- target names

Manifest is restore source of truth. Restore never guesses from directory
scanning.
