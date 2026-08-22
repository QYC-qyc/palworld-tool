$ErrorActionPreference = "Stop"
Set-Location -LiteralPath $PSScriptRoot
$parent = Split-Path $PSScriptRoot -Parent
$env:GOROOT = Join-Path $parent "pal-goroot\go"
$env:GOPATH = Join-Path $parent "pal-gopath"
$env:GOCACHE = Join-Path $parent "pal-gocache"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

$out = Join-Path $PSScriptRoot "dist\paladmin\paladmin"
New-Item -ItemType Directory -Force -Path (Split-Path $out) | Out-Null

Write-Host "=== 构建前端 ==="
Push-Location web
npm install --no-audit --no-fund
npm run build
Pop-Location

Write-Host "=== 交叉编译 Linux amd64 (CGO_ENABLED=0) ==="
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o $out .

Write-Host "=== 组装发布目录 ==="
$dist = Split-Path $out
Copy-Item config.yaml $dist -Force
Copy-Item -Recurse -Force web $dist
Copy-Item -Recurse -Force data $dist
Copy-Item -Recurse -Force module $dist
Copy-Item -Recurse -Force deploy $dist
Copy-Item scripts/install.sh $dist -Force
Copy-Item DEPLOY.md $dist -Force
Copy-Item .env.example $dist -Force

Write-Host "=== 打包 tar.gz ==="
Push-Location dist
tar -czf paladmin-linux-amd64.tar.gz paladmin
Pop-Location

Write-Host "=== 完成 ==="
Get-Item dist\paladmin-linux-amd64.tar.gz | Format-List Name,Length,LastWriteTime
