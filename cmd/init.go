package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	var output string

	c := &cobra.Command{
		Use:   "init",
		Short: "Write an example config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(output); err == nil {
				return fmt.Errorf("%s already exists", output)
			}
			return os.WriteFile(output, []byte(exampleConfig), 0o600)
		},
	}

	c.Flags().StringVarP(&output, "output", "o", "config.yaml", "config file to create")
	return c
}

const exampleConfig = `# dbbackup233 example config.
# Keep this file private: it may contain database and object-storage secrets.

defaults:
  compress: gzip
  timestamp_format: "20060102-150405"
  concurrency: 1
  work_dir: "./backups"
  manifest_path: "./backups/manifest.jsonl"

sources:
  - name: "mysql-game"
    type: mysql
    mysql:
      host: "127.0.0.1"
      port: 3306
      user: "backup"
      password: "${MYSQL_BACKUP_PASSWORD}"
      database: "game"
      dump_tool: "mysqldump"
      restore_tool: "mysql"
      mode: "mysqldump"
      single_transaction: true
      quick: true
      routines: true
      triggers: true
      events: true
      set_gtid_purged: "OFF"

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
      restore_datadir: ""
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
      # Optional. If empty, dbbackup233 uses xtrabackup_dir/latest-base.txt.
      incremental_base_dir: ""
      xtrabackup_restore_dir: "./restore/mysql-xtrabackup"
      restore_datadir: ""
      restore_stop_command: []
      restore_start_command: []
      restore_fix_ownership_command: []

  - name: "postgres-analytics"
    type: postgres
    postgres:
      host: "127.0.0.1"
      port: 5432
      user: "backup"
      password: "${PG_BACKUP_PASSWORD}"
      database: "analytics"
      dump_tool: "pg_dump"
      restore_tool: "pg_restore"
      mode: "pg_dump"
      format: "custom"
      jobs: 1

  - name: "mongo-events"
    type: mongo
    mongo:
      uri: "${MONGO_BACKUP_URI}"
      database: "events"
      dump_tool: "mongodump"
      restore_tool: "mongorestore"

  - name: "redis-cache"
    type: redis
    redis:
      host: "127.0.0.1"
      port: 6379
      password: "${REDIS_PASSWORD}"
      mode: "bgsave"
      cli_path: "redis-cli"
      rdb_path: "/var/lib/redis/dump.rdb"
      restore_path: "./restore/redis/dump.rdb"

  - name: "elasticsearch-logs"
    type: elasticsearch
    elasticsearch:
      url: "http://127.0.0.1:9200"
      username: "${ES_USERNAME}"
      password: "${ES_PASSWORD}"
      repository: "game-backups"
      snapshot: "dbbackup233-game-logs"
      mode: "snapshot"
      indices: ["game-*"]
      include_global_state: false

  - name: "clickhouse-events"
    type: clickhouse
    clickhouse:
      host: "127.0.0.1"
      port: 9000
      user: "backup"
      password: "${CLICKHOUSE_PASSWORD}"
      database: "game_events"
      mode: "full"
      backup_name: "game-events-full"
      backup_destination: "Disk('backups', 'game-events-full.zip')"
      client_tool: "clickhouse-client"

  - name: "server-files"
    type: file
    file:
      paths: ["./data", "./logs"]
      restore_dir: "./restore/files"

  - name: "server-config"
    type: config
    config:
      path: "./config"
      mode: "zip"
      restore_dir: "./restore/config"
      tag_prefix: "dbbackup233"

targets:
  - name: "local"
    type: local
    local:
      path: "./backups"

  - name: "oss"
    type: s3
    s3:
      endpoint: "oss-cn-shanghai.aliyuncs.com"
      bucket: "my-db-backups"
      region: "oss-cn-shanghai"
      access_key: "${OSS_ACCESS_KEY_ID}"
      secret_key: "${OSS_ACCESS_KEY_SECRET}"
      prefix: "dbbackup233"
      use_ssl: true

jobs:
  - name: "mysql-game"
    source: "mysql-game"
    targets: ["local", "oss"]
  - name: "mysql-game-full"
    source: "mysql-game-full"
    targets: ["local", "oss"]
  - name: "mysql-game-incremental"
    source: "mysql-game-incremental"
    targets: ["local", "oss"]
  - name: "postgres-analytics"
    source: "postgres-analytics"
    targets: ["local"]
  - name: "mongo-events"
    source: "mongo-events"
    targets: ["local"]
  - name: "redis-cache"
    source: "redis-cache"
    targets: ["local"]
  - name: "elasticsearch-logs"
    source: "elasticsearch-logs"
    targets: ["local"]
  - name: "clickhouse-events"
    source: "clickhouse-events"
    targets: ["local"]
  - name: "server-files"
    source: "server-files"
    targets: ["local"]
  - name: "server-config"
    source: "server-config"
    targets: ["local"]
`
