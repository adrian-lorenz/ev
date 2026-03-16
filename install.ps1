# install.ps1 — Download and install ev from GitHub Releases
#
# Usage:
#   iwr -useb https://raw.githubusercontent.com/adrian-lorenz/ev/main/install.ps1 | iex
#
# Or with a custom install directory:
#   $env:EV_INSTALL_DIR = "C:\tools"; iwr -useb .../install.ps1 | iex

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repo      = "adrian-lorenz/ev"
$Binary    = "ev.exe"
$InstallDir = if ($env:EV_INSTALL_DIR) { $env:EV_INSTALL_DIR } else { "$env:LOCALAPPDATA\ev" }

function Info  ($msg) { Write-Host "  [+] $msg" -ForegroundColor Green }
function Warn  ($msg) { Write-Host "  [!] $msg" -ForegroundColor Yellow }
function Fatal ($msg) { Write-Host "  [x] $msg" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  ev installer" -ForegroundColor Cyan
Write-Host "  ────────────────────────────────────────"

# ── detect architecture ───────────────────────
$arch = (Get-CimInstance Win32_Processor).Architecture
$GoArch = switch ($arch) {
  9  { "amd64" }   # x86_64
  12 { "arm64" }   # ARM64
  default { Fatal "Unsupported architecture: $arch" }
}
Info "Platform: windows/$GoArch"

# ── fetch latest version ──────────────────────
Write-Host ""
Write-Host "  Fetching latest release…"

try {
  $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
  $version = $release.tag_name
} catch {
  Fatal "Could not reach GitHub API. Check your internet connection."
}

Info "Latest version: $version"

# ── build URLs ───────────────────────────────
$archive      = "ev_windows_${GoArch}.zip"
$downloadUrl  = "https://github.com/$Repo/releases/download/$version/$archive"
$checksumUrl  = "https://github.com/$Repo/releases/download/$version/checksums.txt"

# ── download ──────────────────────────────────
Write-Host ""
Write-Host "  Downloading $archive…"

$tmp = New-TemporaryFile | ForEach-Object { $_.DirectoryName }
$tmpDir = Join-Path $tmp "ev-install-$([System.IO.Path]::GetRandomFileName())"
New-Item -ItemType Directory -Path $tmpDir | Out-Null

$archivePath  = Join-Path $tmpDir $archive
$checksumPath = Join-Path $tmpDir "checksums.txt"

try {
  Invoke-WebRequest -Uri $downloadUrl  -OutFile $archivePath  -UseBasicParsing
  Invoke-WebRequest -Uri $checksumUrl  -OutFile $checksumPath -UseBasicParsing
} catch {
  Fatal "Download failed: $_"
}

Info "Downloaded successfully"

# ── verify checksum ───────────────────────────
Write-Host ""
Write-Host "  Verifying checksum…"

$expected = (Get-Content $checksumPath | Where-Object { $_ -match $archive }) -split '\s+' | Select-Object -First 1
if (-not $expected) {
  Warn "Checksum entry not found — skipping verification"
} else {
  $actual = (Get-FileHash -Algorithm SHA256 $archivePath).Hash.ToLower()
  if ($actual -ne $expected.ToLower()) {
    Fatal "Checksum mismatch! Expected $expected, got $actual"
  }
  Info "Checksum OK"
}

# ── extract ───────────────────────────────────
Write-Host ""
Write-Host "  Extracting…"

Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force
$extracted = Join-Path $tmpDir $Binary
if (-not (Test-Path $extracted)) {
  Fatal "Binary not found in archive: $Binary"
}

# ── install ───────────────────────────────────
Write-Host ""
Write-Host "  Installing to $InstallDir…"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Copy-Item -Force $extracted (Join-Path $InstallDir $Binary)

# ── add to PATH ───────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
  Warn "Added $InstallDir to your PATH. Restart your terminal to use 'ev'."
} else {
  Info "$InstallDir already in PATH"
}

# ── cleanup ───────────────────────────────────
Remove-Item -Recurse -Force $tmpDir

# ── done ──────────────────────────────────────
Write-Host ""
Write-Host "  ev $version installed!" -ForegroundColor Green
Write-Host ""
Write-Host "  Location : $InstallDir\$Binary"
Write-Host ""
Write-Host "  Get started:"
Write-Host "    ev init"
Write-Host "    ev set MY_SECRET"
Write-Host "    ev run <your-command>"
Write-Host ""
Write-Host "  Web UI:"
Write-Host "    ev manage"
Write-Host ""
