# setup.ps1 — Build and install ev on Windows
#
# Run with:  .\setup.ps1
# Or to install system-wide (requires admin):  .\setup.ps1 -SystemWide

param(
    [switch]$SystemWide
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Change to script directory
Set-Location $PSScriptRoot

Write-Host "Building ev..."
go build -ldflags="-s -w" -o ev.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed."
    exit 1
}
Write-Host "Build successful."

if ($SystemWide) {
    # Install to C:\Program Files\ev\ (requires admin)
    $dest = "C:\Program Files\ev"
} else {
    # Install to %LOCALAPPDATA%\ev\ (no admin needed)
    $dest = "$env:LOCALAPPDATA\ev"
}

New-Item -ItemType Directory -Force -Path $dest | Out-Null
Copy-Item -Force ev.exe "$dest\ev.exe"
Write-Host "Installed to $dest\ev.exe"

# Add to PATH if not already there
$currentPath = [Environment]::GetEnvironmentVariable(
    "Path",
    $(if ($SystemWide) { "Machine" } else { "User" })
)

if ($currentPath -notlike "*$dest*") {
    $scope = if ($SystemWide) { "Machine" } else { "User" }
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$dest", $scope)
    Write-Host "Added $dest to PATH ($scope)."
    Write-Host "Restart your terminal for the PATH change to take effect."
} else {
    Write-Host "$dest is already in PATH."
}

Write-Host ""
Write-Host "Done. Run: ev --version"
