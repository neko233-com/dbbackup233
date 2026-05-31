param(
    [string]$Repo = "neko233-com/dbbackup233",
    [string]$InstallDir = "$env:USERPROFILE\.dbbackup233\bin"
)

$ErrorActionPreference = "Stop"

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { throw "Only amd64 Windows is supported by the published binary matrix." }
$asset = "dbbackup233-windows-$arch.exe"
$api = "https://api.github.com/repos/$Repo/releases/latest"

Write-Host "Resolving latest release from $api"
$release = Invoke-RestMethod -Uri $api -Headers @{ "Accept" = "application/vnd.github+json" }
$download = ($release.assets | Where-Object { $_.name -eq $asset } | Select-Object -First 1).browser_download_url
if (-not $download) {
    throw "Release asset not found: $asset"
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$target = Join-Path $InstallDir "dbbackup233.exe"
Write-Host "Downloading $asset to $target"
Invoke-WebRequest -Uri $download -OutFile $target

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ';') -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to user PATH. Open a new terminal to use dbbackup233."
}

& $target version
