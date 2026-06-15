param(
    [int]$Rows = 100000,
    [int]$PayloadBytes = 512
)

$ErrorActionPreference = "Stop"

docker --version | Out-Host
docker pull mysql:8.0 | Out-Host
docker pull percona/percona-xtrabackup:8.0 | Out-Host

$env:DBBACKUP233_XTRA_ROWS = "$Rows"
$env:DBBACKUP233_XTRA_PAYLOAD_BYTES = "$PayloadBytes"
go test -tags "docker_integration docker_benchmark docker_xtrabackup" ./backup -run TestDockerMySQLXtraBackupFullIncrementalAndWrites -count=1 -v
