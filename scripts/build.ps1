# Builds bin\autosave.exe (CLI) and bin\autosave-tray.exe (tray, no console).
# Compatible with Windows PowerShell 5.1.
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    $version = 'dev'
    try {
        $described = git describe --tags --always --dirty 2>$null
        if ($LASTEXITCODE -eq 0 -and $described) { $version = $described.Trim() }
    } catch { }

    New-Item -ItemType Directory -Force -Path bin | Out-Null
    go build -ldflags "-X main.version=$version" -o bin\autosave.exe .\cmd\autosave
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    # Embed the tray icon as the exe icon via a .syso resource (git-ignored).
    go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64 --manifest none `
        --icon internal\tray\icon.ico --product-name autosave-tray --out cmd\autosave-tray\rsrc
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    go build -ldflags "-H windowsgui -X main.version=$version" -o bin\autosave-tray.exe .\cmd\autosave-tray
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Output "Built bin\autosave.exe and bin\autosave-tray.exe ($version)"
} finally {
    Pop-Location
}
