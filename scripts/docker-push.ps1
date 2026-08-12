# 构建并推送 PalAdmin 镜像到 Gitee 容器镜像仓库
# 镜像仓库：https://gitee.com/QYC-qyc/docker-palworld-tool
# 镜像地址：gitee.com/qyc-qyc/docker-palworld-tool:latest
#
# 用法：
#   $env:DOCKER_PASSWORD="你的Gitee令牌"
#   powershell -ExecutionPolicy Bypass -File scripts\docker-push.ps1
$ErrorActionPreference = "Stop"
Set-Location -LiteralPath (Split-Path $PSScriptRoot -Parent)

$REGISTRY = "gitee.com"
$NAMESPACE = "QYC-qyc"
$IMAGE = "docker-palworld-tool"
$TAG = "latest"
$full = "$REGISTRY/$($NAMESPACE.ToLower())/$IMAGE`:$TAG"

Write-Host "==> 构建镜像 $full (linux/amd64)"
docker build --platform linux/amd64 -t $full .

if ($env:DOCKER_PASSWORD) {
    Write-Host "==> 登录 $REGISTRY"
    $env:DOCKER_PASSWORD | docker login $REGISTRY -u $NAMESPACE --password-stdin
} else {
    Write-Host "==> 未设置 DOCKER_PASSWORD，请先 docker login $REGISTRY" -ForegroundColor Yellow
}

Write-Host "==> 推送 $full"
docker push $full
Write-Host "==> 完成，服务器拉取: docker pull $full" -ForegroundColor Green
