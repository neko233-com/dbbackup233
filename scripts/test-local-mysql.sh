#!/usr/bin/env bash
set -euo pipefail

HOST_NAME="${HOST_NAME:-127.0.0.1}"
PORT="${PORT:-3306}"
USER="${USER:-root}"
PASSWORD="${PASSWORD:-root}"
DATABASE="${DATABASE:-dbbackup233_it}"

command -v go >/dev/null

export DBBACKUP233_MYSQL_HOST="$HOST_NAME"
export DBBACKUP233_MYSQL_PORT="$PORT"
export DBBACKUP233_MYSQL_USER="$USER"
export DBBACKUP233_MYSQL_PASSWORD="$PASSWORD"
export DBBACKUP233_MYSQL_DATABASE="$DATABASE"

echo "Running official mysqldump integration test against ${HOST_NAME}:${PORT}"
go test -tags mysql_integration ./backup -run TestLocalMySQL80Backup -count=1 -v

CONFIG_PATH="$PWD/config.local.yaml"
BACKUP_DIR="$PWD/backups-local"
cat > "$CONFIG_PATH" <<YAML
defaults:
  compress: gzip
  timestamp_format: "20060102-150405"
  concurrency: 1
  work_dir: "$BACKUP_DIR"
  manifest_path: "$BACKUP_DIR/manifest.jsonl"

sources:
  - name: "mysql80-local"
    type: mysql
    mysql:
      host: "$HOST_NAME"
      port: $PORT
      user: "$USER"
      password: "$PASSWORD"
      database: "$DATABASE"
      dump_tool: "mysqldump"
      restore_tool: "mysql"
      mode: "mysqldump"
      single_transaction: true
      quick: true
      routines: true
      triggers: true
      events: true
      set_gtid_purged: "OFF"

targets:
  - name: "local"
    type: local
    local:
      path: "$BACKUP_DIR"

jobs:
  - name: "mysql80-local"
    source: "mysql80-local"
    targets: ["local"]
YAML

echo "Running dbbackup233 CLI backup with config.local.yaml"
go run . backup -c "$CONFIG_PATH" --timeout 2m
go run . list -c "$CONFIG_PATH"
echo "Local MySQL integration test passed"
