# 构建并推送 PalAdmin 镜像到 Gitee 容器镜像仓库
# 用法：
#   1. 先修改下面的 REGISTRY/IMAGE/TAG
#   2. 在 PowerShell 中执行：
#        $env:DOCKER_PASSWORD="你的Gitee令牌"
#        powershell -ExecutionPolicy Bypass -File scripts/docker-push.ps1

$ErrorActionPreference = "Stop"
Set-Location -LiteralPath (Split-Path $PSScriptRoot -Parent)

# ====== 按你的 Gitee 信息修改 ======
$REGISTRY = "gitee.com"
$NAMESPACE = "qyc-qyc"          # Gitee 用户名/组织名（小写）
$IMAGE = "paladmin"
$TAG = "latest"
# ==================================

$fullImage = "$REGISTRY/$NAMESPACE/$IMAGE`:$TAG"

Write-Host "==> 构建镜像 $fullImage (linux/amd64)"
docker build --platform linux/amd64 -t $fullImage .

if ($env:DOCKER_PASSWORD) {
    Write-Host "==> 登录 $REGISTRY"
    $env:DOCKER_PASSWORD | docker login $REGISTRY -u $NAMESPACE --password-stdin
} else {
    Write-Host "==> 未设置 DOCKER_PASSWORD，请先手动执行: docker login $REGISTRY" -ForegroundColor Yellow
}

Write-Host "==> 推送 $fullImage"
docker push $fullImage

Write-Host "==> 完成" -ForegroundColor Green
