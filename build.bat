@echo off
setlocal
cd /d "%~dp0"
set GOROOT=%~dp0goroot\go
set GOPATH=%~dp0gopath
set GOCACHE=%~dp0gocache
set PATH=%GOROOT%\bin;%PATH%
echo === go version ===
go version
if errorlevel 1 exit /b 1
echo === go env ===
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOTOOLCHAIN=local
echo === go mod tidy ===
go mod tidy
if errorlevel 1 exit /b 1
echo === go build ===
go build -o paladmin.exe .
if errorlevel 1 exit /b 1
echo === BUILD OK ===
dir paladmin.exe
