#!/usr/bin/env sh
set -eu

docker --version
if [ -n "${DBBACKUP233_DOCKER_MYSQL_PORT:-}" ]; then
  echo "Using Docker MySQL host port $DBBACKUP233_DOCKER_MYSQL_PORT"
else
  echo "Using an automatically selected non-3306 Docker MySQL host port"
fi
go test -tags docker_integration ./backup -run TestDockerMySQLBackupRestoreAndHistory -count=1 -v
