param(
    [int]$Rows = 50000,
    [int]$PayloadBytes = 256
)

$ErrorActionPreference = "Stop"

docker --version | Out-Host
$env:DBBACKUP233_BENCH_ROWS = "$Rows"
$env:DBBACKUP233_BENCH_PAYLOAD_BYTES = "$PayloadBytes"
go test -tags "docker_integration docker_benchmark" ./backup -run TestDockerMySQLBackupBenchmark -count=1 -v
