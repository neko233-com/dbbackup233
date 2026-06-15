# Restore Guide

## Logical MySQL Restore

Logical restore uses `mysql` and can restore to the original database or to
`restore_database`.

```bash
dbbackup233 restore mysql-game --version 20260615-120000 -c config.local.yaml --yes
```

Before restore writes data, dbbackup233 verifies the artifact SHA-256 recorded
in the manifest.

## Physical MySQL Prepare

For physical backup, restore first extracts `.xb.zip` or `.xb.tar.gz` and runs:

```bash
xtrabackup --prepare --target-dir=<xtrabackup_restore_dir>
```

If `restore_datadir` is empty, dbbackup233 stops here. This is safe default
behavior.

## One-Key Physical Restore

Set `restore_datadir` and service commands to allow dbbackup233 to perform
copy-back automatically.

Linux example:

```yaml
mysql:
  mode: "full"
  physical_engine: "xtrabackup"
  physical_artifact_format: "zip"
  xtrabackup_tool: "xtrabackup"
  xtrabackup_restore_dir: "/var/restore/mysql"
  restore_datadir: "/var/lib/mysql"
  restore_stop_command: ["systemctl", "stop", "mysql"]
  restore_fix_ownership_command: ["chown", "-R", "mysql:mysql", "/var/lib/mysql"]
  restore_start_command: ["systemctl", "start", "mysql"]
```

Restore command:

```bash
dbbackup233 restore mysql-game-full --version 20260615-120000 -c config.local.yaml --yes
```

Execution order:

1. Verify SHA-256.
2. Extract artifact.
3. Run `xtrabackup --prepare`.
4. Run `restore_stop_command`.
5. Run `xtrabackup --copy-back --target-dir=<restore> --datadir=<restore_datadir>`.
6. Run `restore_fix_ownership_command`.
7. Run `restore_start_command`.

dbbackup233 does not delete or empty `restore_datadir`. XtraBackup will fail if
the datadir is not in a valid restore state. This prevents accidental
destructive overwrite.

## Incremental Restore Note

Current physical restore prepares the selected artifact. For a multi-step
incremental chain, operators should restore from a prepared chain directory or
use a full artifact. Chain application automation is planned as a next product
step.

## Other Providers

- PostgreSQL restores through `pg_restore` or `psql`, depending on dump format.
- Redis restores by copying the selected RDB artifact to `restore_path`.
- MongoDB restores through `mongorestore --archive --gzip`.
- Elasticsearch restores by calling `_snapshot/<repo>/<snapshot>/_restore`.
- ClickHouse restores by running `RESTORE ... FROM <backup_destination>`.
