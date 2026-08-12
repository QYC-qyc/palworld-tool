# 构建并推送到 GitHub Container Registry (GHCR)
# 镜像：ghcr.io/qyc-qyc/palworld-tool:latest
#
# 用法：
#   $env:GITHUB_TOKEN="你的GitHubToken"   # 需要 write:packages 权限
#   powershell -ExecutionPolicy Bypass -File scripts\docker-push.ps1
$ErrorActionPreference = "Stop"
Set-Location -LiteralPath (Split-Path $PSScriptRoot -Parent)

$REGISTRY = "ghcr.io"
$NAMESPACE = "QYC-qyc"
$IMAGE = "palworld-tool"
$TAG = "latest"
$full = "$REGISTRY/$($NAMESPACE.ToLower())/$IMAGE`:$TAG"

Write-Host "==> 构建镜像 $full (linux/amd64)"
docker build --platform linux/amd64 -t $full .

if ($env:GITHUB_TOKEN) {
    Write-Host "==> 登录 $REGISTRY"
    $env:GITHUB_TOKEN | docker login $REGISTRY -u $NAMESPACE --password-stdin
} else {
    Write-Host "==> 未设置 GITHUB_TOKEN，请先 docker login $REGISTRY" -ForegroundColor Yellow
}

Write-Host "==> 推送 $full"
docker push $full
Write-Host "==> 完成，服务器拉取: docker pull $full" -ForegroundColor Green
