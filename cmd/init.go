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
  - name: "postgres-analytics"
    source: "postgres-analytics"
    targets: ["local"]
  - name: "mongo-events"
    source: "mongo-events"
    targets: ["local"]
  - name: "redis-cache"
    source: "redis-cache"
    targets: ["local"]
  - name: "server-files"
    source: "server-files"
    targets: ["local"]
  - name: "server-config"
    source: "server-config"
    targets: ["local"]
`
