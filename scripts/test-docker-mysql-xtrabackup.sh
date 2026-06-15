#!/usr/bin/env bash
set -euo pipefail

ROWS="${DBBACKUP233_XTRA_ROWS:-100000}"
PAYLOAD_BYTES="${DBBACKUP233_XTRA_PAYLOAD_BYTES:-512}"

docker --version
docker pull mysql:8.0
docker pull percona/percona-xtrabackup:8.0

DBBACKUP233_XTRA_ROWS="$ROWS" \
DBBACKUP233_XTRA_PAYLOAD_BYTES="$PAYLOAD_BYTES" \
go test -tags "docker_integration docker_benchmark docker_xtrabackup" ./backup -run TestDockerMySQLXtraBackupFullIncrementalAndWrites -count=1 -v
