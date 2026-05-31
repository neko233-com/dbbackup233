$ErrorActionPreference = "Stop"

$targets = @(
    @{ GOOS = "linux"; GOARCH = "amd64"; EXT = "" },
    @{ GOOS = "linux"; GOARCH = "arm64"; EXT = "" },
    @{ GOOS = "darwin"; GOARCH = "amd64"; EXT = "" },
    @{ GOOS = "darwin"; GOARCH = "arm64"; EXT = "" },
    @{ GOOS = "windows"; GOARCH = "amd64"; EXT = ".exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; EXT = ".exe" }
)

New-Item -ItemType Directory -Force -Path "dist/check" | Out-Null

foreach ($target in $targets) {
    $env:GOOS = $target.GOOS
    $env:GOARCH = $target.GOARCH
    $env:CGO_ENABLED = "0"
    $out = "dist/check/dbbackup233-$($target.GOOS)-$($target.GOARCH)$($target.EXT)"
    Write-Host "building $($target.GOOS)/$($target.GOARCH)"
    go build -o $out .
}

Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host "platform matrix build passed"
