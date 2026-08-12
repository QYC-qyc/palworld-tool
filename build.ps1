$ErrorActionPreference = "Stop"
Set-Location -LiteralPath $PSScriptRoot
$parent = Split-Path $PSScriptRoot -Parent
$env:GOROOT = Join-Path $parent "pal-goroot\go"
$env:GOPATH = Join-Path $parent "pal-gopath"
$env:GOCACHE = Join-Path $parent "pal-gocache"
$env:PATH = "$env:GOROOT\bin;$env:PATH"
Write-Host "=== go version ==="
go version
Write-Host "=== go env ==="
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOTOOLCHAIN=local
Write-Host "=== go mod tidy ==="
go mod tidy
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "=== go build ==="
go build -o paladmin.exe .
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "=== BUILD OK ==="
Get-Item paladmin.exe | Format-List Name,Length,LastWriteTime
