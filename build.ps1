$ErrorActionPreference = 'Stop'

Write-Host "Building sync-must-simple Server for Windows (with Systray)..."

$env:CGO_ENABLED = "1"
New-Item -ItemType Directory -Force -Path dist | Out-Null
go build -o dist\sync-must-simple.exe -tags systray .\server

Write-Host "Build complete! Binary located at dist\sync-must-simple.exe"
