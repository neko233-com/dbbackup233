#!/usr/bin/env sh
set -eu

export DBBACKUP233_BENCH_ROWS="${DBBACKUP233_BENCH_ROWS:-50000}"
export DBBACKUP233_BENCH_PAYLOAD_BYTES="${DBBACKUP233_BENCH_PAYLOAD_BYTES:-256}"

docker --version
go test -tags "docker_integration docker_benchmark" ./backup -run TestDockerMySQLBackupBenchmark -count=1 -v
