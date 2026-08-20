@echo off
chcp 65001 >nul
title PalAdmin 帕鲁管理面板

echo ==========================================
echo   PalAdmin 帕鲁服务器管理面板
echo ==========================================
echo.

cd /d "%~dp0"

if not exist "paladmin.exe" (
  echo [错误] 未找到 paladmin.exe，请确保本脚本与 paladmin.exe 在同一目录
  pause
  exit /b 1
)

echo 正在启动面板...
echo 启动后浏览器会自动打开 http://localhost:8190
echo 关闭此窗口即可停止面板。
echo.

paladmin.exe

echo.
echo 面板已停止。
pause
