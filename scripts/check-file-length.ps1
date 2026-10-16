# Fails when a source or doc file exceeds the line limits defined in CLAUDE.md.
# Compatible with Windows PowerShell 5.1.
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$skipDirs = @('.git', 'bin', 'dist', '.superpowers', 'docs\plans')

function Test-Skipped([string]$relPath) {
    foreach ($dir in $skipDirs) {
        if ($relPath -eq $dir -or $relPath.StartsWith($dir + '\')) { return $true }
    }
    return $false
}

function Get-Limit([string]$relPath, [string]$name) {
    if ($name -like '*_test.go') { return 400 }
    if ($name -like '*.go') { return 300 }
    if ($relPath -eq 'CLAUDE.md') { return 150 }
    if ($name -like '*.md') { return 250 }
    return 0
}

$offenders = @()
Get-ChildItem -Path $root -Recurse -File | ForEach-Object {
    $relPath = $_.FullName.Substring($root.Length).TrimStart('\')
    if (Test-Skipped $relPath) { return }
    $limit = Get-Limit $relPath $_.Name
    if ($limit -eq 0) { return }
    $lines = @(Get-Content -LiteralPath $_.FullName).Count
    if ($lines -gt $limit) {
        $offenders += "${relPath}: $lines lines (limit $limit)"
    }
}

if ($offenders.Count -gt 0) {
    $offenders | ForEach-Object { Write-Output $_ }
    exit 1
}
Write-Output 'OK'
