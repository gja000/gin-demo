# 本地一键部署（Windows）：构建前端、编译后端、启动服务
param(
  [string]$GoOS = "windows",
  [string]$GoArch = "amd64"
)

$ErrorActionPreference = "Stop"
Write-Host "==> 构建前端"
Push-Location frontend
npm install --legacy-peer-deps
npm run build
Pop-Location

Write-Host "==> 编译后端 ($GoOS/$GoArch)"
$env:CGO_ENABLED = "0"
$env:GOOS = $GoOS
$env:GOARCH = $GoArch
go build -o dist/server .

Write-Host "==> 启动服务"
$id = (Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue).OwningProcess | Select-Object -First 1
if ($id) { Stop-Process -Id $id -Force }
Start-Process -FilePath "dist/server.exe" -RedirectStandardOutput "server.log" -RedirectStandardError "server.err"
Start-Sleep -Seconds 6
$r = Invoke-WebRequest -Uri "http://127.0.0.1:8080/" -UseBasicParsing
Write-Host "服务状态: $($r.StatusCode)，静态首页 OK"
