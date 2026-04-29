# 多终端 AI 编排平台 - 启动脚本
# 双击运行或在 PowerShell 中执行: .\start.ps1

$ErrorActionPreference = "Stop"
$Root = $PSScriptRoot

# --- 颜色输出 ---
function Write-Status($msg) { Write-Host "[*] $msg" -ForegroundColor Cyan }
function Write-Ok($msg)    { Write-Host "[+] $msg" -ForegroundColor Green }
function Write-Warn($msg)  { Write-Host "[!] $msg" -ForegroundColor Yellow }
function Write-Fail($msg)  { Write-Host "[-] $msg" -ForegroundColor Red }

# --- 检查端口占用 ---
$Port = 8080
$ConfigPath = Join-Path $Root "config.yaml"

if (Test-Path $ConfigPath) {
    $Port = (Select-String -Path $ConfigPath -Pattern 'port:\s*(\d+)' -ErrorAction SilentlyContinue).Matches.Groups[1].Value
    if (-not $Port) { $Port = 8080 }
}

Write-Status "检查端口 $Port ..."
$listener = $null
try {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)
    $listener.Start()
    $listener.Stop()
} catch {
    Write-Fail "端口 $Port 已被占用，服务可能已在运行"
    Write-Warn "如需重启，请先关闭已运行的 ai-orchestrator 进程"
    $proc = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique
    if ($proc) {
        $procName = (Get-Process -Id $proc[0] -ErrorAction SilentlyContinue).ProcessName
        Write-Warn "占用进程: PID=$($proc[0]) ($procName)"
    }
    Read-Host "按 Enter 退出"
    exit 1
}

# --- 构建检查 ---
$ExePath = Join-Path $Root "ai-orchestrator.exe"

if (-not (Test-Path $ExePath)) {
    Write-Warn "未找到 ai-orchestrator.exe，正在编译 ..."
    Push-Location $Root
    go build -o ai-orchestrator.exe ./cmd/server/
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "编译失败"
        Pop-Location
        Read-Host "按 Enter 退出"
        exit 1
    }
    Pop-Location
    Write-Ok "编译完成"
}

# --- 前端构建检查 ---
$WebDist = Join-Path $Root "web\dist"
if (-not (Test-Path $WebDist)) {
    Write-Warn "未找到前端构建产物，正在构建 ..."
    Push-Location (Join-Path $Root "web")
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "前端构建失败"
        Pop-Location
        Read-Host "按 Enter 退出"
        exit 1
    }
    Pop-Location
    Write-Ok "前端构建完成"
}

# --- 启动服务 ---
Write-Status "启动 ai-orchestrator ..."
Write-Host ""
Write-Host "  多终端 AI 编排平台" -ForegroundColor White
Write-Host "  http://localhost:${Port}/board" -ForegroundColor Yellow
Write-Host ""

Push-Location $Root
try {
    .\ai-orchestrator.exe
} finally {
    Pop-Location
}
