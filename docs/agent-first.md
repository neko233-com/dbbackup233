# Agent-First Product

dbbackup233 is designed as a local operations agent, not only a dump wrapper.

## Principles

- One binary per machine.
- One YAML config per environment.
- Safe defaults.
- Explicit restore.
- Machine-readable manifest.
- HTTP reporting for central operations.
- CLI-first, automation-friendly output.

## Agent Responsibilities

The agent owns:

- backup scheduling entrypoints
- provider orchestration
- artifact naming
- compression/archive format
- SHA-256 integrity metadata
- local and cloud target storage
- restore selection by manifest version
- dry-run previews
- HTTP event reporting
- install/update/verify/cron/prune commands

## Game Operations Fit

Game databases are large, write-heavy, and latency-sensitive. The MySQL physical
path is built for that reality:

- `mode: full`
- `mode: incremental`
- default `.xb.zip` artifact
- automatic latest-base tracking
- Docker validation with concurrent writes

`mysqldump` remains available for small or portable backup needs, but it is not
the preferred path for huge production game data.

The same agent model applies to supporting systems:

- Redis for session/cache state snapshots
- PostgreSQL for analytics or account data
- Elasticsearch for logs/search indexes
- ClickHouse for event pipelines
- MongoDB for document stores
- Files/config repositories for game service state

## Future Agent Features

Planned directions:

- automatic incremental chain materialization for restore
- restore rehearsal command
- health check summaries for central dashboard
- agent identity and fleet tags
- remote policy distribution
- backup SLA reporting
