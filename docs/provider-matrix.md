# Provider Matrix

dbbackup233 has native providers for common game infrastructure and a universal
command provider for everything else.

| Type | Full | Incremental / Delta | Restore | Artifact |
| --- | --- | --- | --- | --- |
| `mysql` | XtraBackup full, mysqldump | XtraBackup incremental | prepare/copy-back or mysql import | `.xb.zip`, `.xb.tar.gz`, `.sql.gz` |
| `postgres` | pg_dump | planned physical chain | pg_restore or psql | `.dump`, `.sql.gz` |
| `redis` | RDB save/copy | AOF policy external today | copy RDB | `.rdb` |
| `mongo` | mongodump archive | oplog/PITR planned | mongorestore | `.archive.gz` |
| `elasticsearch` | snapshot API | repository incremental internally | snapshot restore API | `.es-snapshot.json` |
| `clickhouse` | BACKUP command | `base_backup` setting | RESTORE command | `.clickhouse-backup.json` |
| `file`, `dir`, `directory`, `path` | zip/tar.gz | planned changed-file index | extract | `.zip`, `.tar.gz` |
| `command`, `exec`, `custom` | any command | delegated to command | delegated to command | configured extension |

## File / Directory

Use this for game configs, uploaded assets, logs, replay files, maps, shard
state, or any plain filesystem data.

```yaml
sources:
  - name: "game-files"
    type: dir
    file:
      paths: ["/srv/game/data", "/srv/game/maps"]
      restore_dir: "/srv/restore/game-files"
      exclude: ["*.tmp", "cache/*"]
```

## Universal Command Provider

Use this when native provider does not exist yet, or when a product already has
official backup CLI.

dbbackup233 injects:

- `ARTIFACT_PATH` environment variable
- `$ARTIFACT_PATH`, `${ARTIFACT_PATH}`, `{{ARTIFACT_PATH}}` template values in
  command args

Backup command must create the artifact unless `capture_stdout: true` is set.
Restore command runs only after manifest SHA-256 verification.

```yaml
sources:
  - name: "etcd-cluster"
    type: command
    command:
      extension: "db"
      backup_command:
        ["etcdctl", "snapshot", "save", "${ARTIFACT_PATH}"]
      restore_command:
        ["etcdutl", "snapshot", "restore", "${ARTIFACT_PATH}", "--data-dir", "/srv/restore/etcd"]

  - name: "custom-streaming-tool"
    type: command
    command:
      extension: "tar.gz"
      backup_command: ["sh", "-lc", "mytool dump --format tar.gz"]
      restore_command: ["sh", "-lc", "mytool restore --stdin"]
      capture_stdout: true
      restore_stdin: true
```

This makes Cassandra, TiDB, Etcd, Consul, Milvus, OpenSearch, Neo4j, object
metadata, proprietary game services, and future stores usable now. Native
providers can still be added later for richer validation and first-class
incremental semantics.
