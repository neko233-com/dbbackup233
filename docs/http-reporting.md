# HTTP Reporting Integration

`dbbackup233` can report backup lifecycle events to your Backup Center. The
mechanism is optional and disabled by default.

## Config

```yaml
report:
  enabled: true
  url: "https://backup-center.example.com/api/backup/events"
  token: "${BACKUP_CENTER_TOKEN}"
  timeout: "5s"
  retry:
    attempts: 3
    backoff: "1s"
  headers:
    X-Project: "game-server"

identity:
  project: "game-server"
  cluster: "prod-shanghai"
  machine_id: "game-001"
```

If `token` is set, dbbackup233 sends:

```http
Authorization: Bearer <token>
Content-Type: application/json
```

## Required Server API

Implement one HTTP endpoint:

```http
POST /api/backup/events
Content-Type: application/json
Authorization: Bearer <token>
```

Return any `2xx` status to acknowledge the event. Non-2xx responses are logged
by the CLI but do not fail the backup.

## Event Payload

Backup start:

```json
{
  "event": "backup.started",
  "project": "game-server",
  "cluster": "prod-shanghai",
  "machine_id": "game-001",
  "job_name": "mysql-game",
  "source_name": "mysql-game",
  "source_type": "mysql",
  "version": "20260531-120000",
  "status": "started",
  "started_at": "2026-05-31T12:00:00Z"
}
```

Backup completion:

```json
{
  "event": "backup.completed",
  "job_name": "mysql-game",
  "source_name": "mysql-game",
  "source_type": "mysql",
  "version": "20260531-120000",
  "status": "success",
  "artifact": {
    "version": "20260531-120000",
    "job_name": "mysql-game",
    "source_name": "mysql-game",
    "source_type": "mysql",
    "file_path": "./backups/mysql/mysql-game-20260531-120000.sql.gz",
    "object_name": "mysql/mysql-game-20260531-120000.sql.gz",
    "size": 123456,
    "targets": ["local"]
  },
  "finished_at": "2026-05-31T12:00:08Z"
}
```

Failed completion uses:

```json
{
  "event": "backup.completed",
  "status": "failed",
  "error": "mysql-game backup failed: exit status 2"
}
```
