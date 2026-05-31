# Backup Benchmark

Benchmarks are Docker-driven and use official `mysqldump` inside the MySQL
container. The benchmark intentionally separates total test time from measured
backup time:

- total test time includes Docker startup and data generation
- reported `elapsed` measures only `dbbackup233 backup`

## Run

Windows:

```powershell
.\scripts\bench-docker-mysql.ps1 -Rows 50000 -PayloadBytes 256
```

Linux/macOS:

```bash
DBBACKUP233_BENCH_ROWS=50000 DBBACKUP233_BENCH_PAYLOAD_BYTES=256 ./scripts/bench-docker-mysql.sh
```

## Current Local Results

Environment:

- Windows host
- Docker Desktop
- MySQL `mysql:8.0`
- official `mysqldump`
- local target
- gzip enabled

| Rows | Payload bytes | Compressed artifact | Backup elapsed | Rows/sec | Compressed MiB/sec | Total test time |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 50,000 | 256 | 0.59 MiB | 419 ms | 119,468 | 1.41 | 40 s |
| 200,000 | 512 | 2.92 MiB | 2.157 s | 92,716 | 1.35 | 118 s |

The low compressed artifact size is expected because the synthetic payload is
highly compressible. For production-like estimates, run with a larger payload or
more random data.

## Notes

- The default benchmark does not occupy local `3306`; it uses an automatically
  selected non-3306 host port.
- The measured backup path is the same provider path used by normal backups.
- For large game databases, run benchmarks on the same disk type and CPU class
  as the target server.
