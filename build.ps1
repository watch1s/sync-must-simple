$ErrorActionPreference = 'Stop'

Write-Host '[*] Building sync-must-simple (Windows Tray App - Zero Dependencies)...' -ForegroundColor Cyan

$env:CGO_ENABLED = '0'
$DistDir = Join-Path $PSScriptRoot 'dist'
$ServerDir = Join-Path $PSScriptRoot 'server'
$OutputFile = Join-Path $DistDir 'sync-must-simple.exe'

if (-not (Test-Path -Path $DistDir)) {
    New-Item -ItemType Directory -Force -Path $DistDir | Out-Null
}

Push-Location $ServerDir
try {
    # Auto-detect Go if not in PATH
    $GoExe = 'go'
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        $GoPaths = @("$env:ProgramFiles\Go\bin\go.exe", "$env:USERPROFILE\go\bin\go.exe", "C:\Go\bin\go.exe")
        foreach ($gp in $GoPaths) {
            if (Test-Path $gp) { $GoExe = $gp; break }
        }
    }
    # -H=windowsgui hides the black console window on launch
    & $GoExe build -ldflags "-H=windowsgui -s -w" -o $OutputFile .
}
finally {
    Pop-Location
}

if (Test-Path -Path $OutputFile) {
    Write-Host ''
    Write-Host '[+] Build succeeded!' -ForegroundColor Green
    Write-Host "[*] Binary output: $OutputFile" -ForegroundColor Yellow
    Write-Host '[*] Double-click the executable to launch in System Tray (Hidden Icons).' -ForegroundColor Cyan
} else {
    Write-Error 'Build failed: Output binary not found.'
}
