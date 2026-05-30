# bootstrap.ps1 - AgentVerse full-stack bootstrap for Windows
#
# Starts ALL services needed to run AgentVerse at 100%:
#   1. CAO Runtime (cli-agent-orchestrator) via Docker in WSL - port 9889
#   2. Vite dev server (HMR) - port 5173
#
# Usage:
#   .\bootstrap.ps1              # start everything
#   .\bootstrap.ps1 start        # same as above
#   .\bootstrap.ps1 stop         # stop all services
#   .\bootstrap.ps1 status       # show service health
#   .\bootstrap.ps1 help         # this help

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('start','stop','status','help','')]
    [string]$Command = 'start',

    [Parameter(Position = 1)]
    [string]$Target = ''
)

$ErrorActionPreference = 'Stop'

# ── Paths & Ports ──────────────────────────────────────────────────────────
$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $RepoRoot

$CaoPort    = if ($env:CAO_PORT) { [int]$env:CAO_PORT } else { 9889 }
$DevPort    = if ($env:DEV_PORT) { [int]$env:DEV_PORT } else { 5173 }
$CaoBaseUrl = if ($env:VITE_CAO_BASE_URL) { $env:VITE_CAO_BASE_URL } else { "http://127.0.0.1:$CaoPort" }
$CaoImage   = 'agentverse-cao:latest'
$CaoContainer = 'agentverse-cao'

$env:VITE_CAO_BASE_URL = $CaoBaseUrl

# ── Pretty output ──────────────────────────────────────────────────────────
function Header($msg) { Write-Host "`n====== $msg ======" -ForegroundColor Magenta }
function Step($msg)   { Write-Host "  >> $msg" -ForegroundColor Cyan }
function Ok($msg)     { Write-Host "  [OK] $msg" -ForegroundColor Green }
function Warn($msg)   { Write-Host "  [!!] $msg" -ForegroundColor Yellow }
function Err($msg)    { Write-Host "  [ERR] $msg" -ForegroundColor Red }
function Info($msg)   { Write-Host "       $msg" -ForegroundColor Gray }
function Hr           { Write-Host ('-' * 64) -ForegroundColor DarkGray }

# ── Helpers ────────────────────────────────────────────────────────────────
function Test-Url($url, [int]$timeout = 3) {
    try {
        $r = Invoke-WebRequest -Uri $url -TimeoutSec $timeout -UseBasicParsing -ErrorAction Stop
        return [int]$r.StatusCode
    } catch {
        return 0
    }
}

function Test-Listening([int]$port) {
    try {
        $c = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue
        return [bool]$c
    } catch {
        return $false
    }
}

function Stop-ByPort([int]$port) {
    try {
        $conns = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue
        if ($conns) {
            $pids = @($conns | Select-Object -ExpandProperty OwningProcess -Unique)
            foreach ($p in $pids) {
                if ($p -gt 0) {
                    Stop-Process -Id $p -Force -ErrorAction SilentlyContinue
                }
            }
        }
    } catch {}
}

function Invoke-Wsl {
    param([string]$Script)
    $prevEAP = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $result = & wsl bash -c $Script 2>&1 | Out-String
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prevEAP
    return @{ Output = $result.Trim(); ExitCode = $code }
}

# ── Prerequisites ──────────────────────────────────────────────────────────
function Test-Prerequisites {
    Header "Checking prerequisites"
    $ok = $true

    Step "Node.js"
    if (Get-Command node -ErrorAction SilentlyContinue) {
        $nv = (& node -v).TrimStart('v')
        if ([version]$nv -ge [version]'20.10.0') { Ok "node $nv" }
        else { Err "node $nv < 20.10.0 required"; $ok = $false }
    } else { Err "node not found"; $ok = $false }

    Step "npm"
    if (Get-Command npm -ErrorAction SilentlyContinue) {
        $mv = (& npm -v)
        if ([version]$mv -ge [version]'10.2.0') { Ok "npm $mv" }
        else { Err "npm $mv < 10.2.0 required"; $ok = $false }
    } else { Err "npm not found"; $ok = $false }

    Step "WSL + Docker"
    if (Get-Command wsl -ErrorAction SilentlyContinue) {
        $r = Invoke-Wsl "docker --version 2>/dev/null && echo OK || echo FAIL"
        if ($r.Output -match 'OK') {
            $ver = ($r.Output -split "`n")[0]
            Ok "WSL Docker: $ver"
        } else {
            Err "Docker not found inside WSL"
            Info "Install: wsl bash -c 'curl -fsSL https://get.docker.com | sh'"
            $ok = $false
        }
    } else {
        Err "WSL not found. Required for Docker + CAO runtime."
        $ok = $false
    }

    Step "Docker daemon"
    if ($ok) {
        $r = Invoke-Wsl "docker info >/dev/null 2>&1 && echo RUNNING || echo STOPPED"
        if ($r.Output -match 'RUNNING') {
            Ok "Docker daemon is running"
        } else {
            Warn "Docker daemon is not running - attempting to start..."
            $r2 = Invoke-Wsl "sudo service docker start 2>&1; sleep 2; docker info >/dev/null 2>&1 && echo RUNNING || echo STOPPED"
            if ($r2.Output -match 'RUNNING') {
                Ok "Docker daemon started"
            } else {
                Err "Could not start Docker daemon"
                Info "Try: wsl bash -c 'sudo service docker start'"
                $ok = $false
            }
        }
    }

    return $ok
}

# ── npm dependencies ───────────────────────────────────────────────────────
function Install-NpmDeps {
    Header "Checking npm dependencies"
    if (-not (Test-Path 'node_modules')) {
        Step "Running npm ci..."
        & npm ci
        if ($LASTEXITCODE -ne 0) { Err "npm ci failed"; return $false }
    }
    Ok "node_modules present"
    return $true
}

# ── CAO via Docker in WSL ─────────────────────────────────────────────────
function Start-CaoDocker {
    Header "Starting CAO runtime (Docker) on :$CaoPort"

    # Already healthy?
    $code = Test-Url "$CaoBaseUrl/health"
    if ($code -eq 200) {
        Ok "CAO is already running (HTTP 200)"
        return $true
    }

    # Convert repo root to WSL path
    $drive = $RepoRoot.Substring(0,1).ToLower()
    $rest = $RepoRoot.Substring(2) -replace '\\','/'
    $wslRepo = "/mnt/$drive$rest"

    # Check if image exists; build if not
    Step "Checking for Docker image $CaoImage..."
    $r = Invoke-Wsl "docker images -q $CaoImage 2>/dev/null | head -1"
    # Filter out WSL path translation warnings - only a 12-char hex ID means the image exists
    $imageId = ($r.Output -split "`n" | Where-Object { $_ -match '^[0-9a-f]{12}' } | Select-Object -First 1)
    if (-not $imageId) {
        Step "Building CAO image with CLI providers (first time - 2-3 minutes)..."
        $workerCli = "npm install --global @openai/codex && curl -fsSL https://cli.kiro.dev/install | bash && curl -fsSL https://antigravity.google/cli/install.sh | bash"
        $r = Invoke-Wsl "cd '$wslRepo' && docker build -t $CaoImage -f infra/runtime/Dockerfile --build-arg WORKER_CLI='$workerCli' . 2>&1"
        if ($r.ExitCode -ne 0) {
            Err "Docker build failed:"
            Info $r.Output
            return $false
        }
        Ok "Image built: $CaoImage"
    } else {
        Ok "Image exists: $CaoImage ($imageId)"
    }

    # Remove stale container
    Invoke-Wsl "docker rm -f $CaoContainer 2>/dev/null" | Out-Null

    # Run with credential mounts
    Step "Starting container $CaoContainer..."
    $dockerRun = "HOME_DIR=~; docker run -d --rm --name $CaoContainer -p ${CaoPort}:9889 " +
        "--memory=12g --cpus=8 --shm-size=1g " +
        "-e CAO_CORS_ORIGINS=http://localhost:$DevPort,http://localhost:4173 " +
        "-e CAO_ALLOWED_HOSTS=127.0.0.1,localhost,0.0.0.0 " +
        "-e CAO_WS_ALLOWED_CLIENTS=http://localhost:$DevPort,http://localhost:4173 " +
        '-v ${HOME_DIR}/.codex:/root/.codex:ro ' +
        '-v ${HOME_DIR}/.kiro:/root/.kiro:ro ' +
        '-v ${HOME_DIR}/.gemini:/root/.gemini:ro ' +
        '-v agentverse-runtime-state:/root/.cao ' +
        "$CaoImage 2>&1"
    $r = Invoke-Wsl $dockerRun
    if ($r.ExitCode -ne 0) {
        Err "docker run failed: $($r.Output)"
        return $false
    }
    Info "Container started"

    # Wait for health
    Step "Waiting for CAO health check..."
    for ($i = 0; $i -lt 45; $i++) {
        Start-Sleep 1
        $code = Test-Url "$CaoBaseUrl/health"
        if ($code -eq 200) {
            Ok "CAO runtime is healthy (HTTP 200)"
            return $true
        }
        if ($i % 10 -eq 9) { Info "still waiting... ($($i+1)s)" }
    }

    # Show container logs on failure
    Err "CAO did not become healthy within 45s"
    Step "Container logs:"
    $r = Invoke-Wsl "docker logs --tail 20 $CaoContainer 2>&1"
    Info $r.Output
    return $false
}

# ── Vite dev server ────────────────────────────────────────────────────────
function Start-ViteDev {
    Header "Starting Vite dev server on :$DevPort"

    if (Test-Listening $DevPort) {
        $code = Test-Url "http://localhost:$DevPort/"
        if ($code -eq 200) {
            Ok "Vite already running (HTTP 200)"
            return $true
        }
        Warn "Port $DevPort in use but not responding - killing"
        Stop-ByPort $DevPort
        Start-Sleep 1
    }

    Step "Launching npm run dev (minimized window)..."
    Start-Process -FilePath 'npm.cmd' -ArgumentList 'run','dev' `
        -WorkingDirectory $RepoRoot `
        -WindowStyle Minimized `
        -PassThru | Out-Null

    # Wait for ready
    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep 1
        $code = Test-Url "http://localhost:$DevPort/"
        if ($code -eq 200) {
            Ok "Vite dev server is live (HTTP 200)"
            return $true
        }
    }

    Err "Vite did not respond within 20s"
    return $false
}

# ── Commands ───────────────────────────────────────────────────────────────
function Cmd-Start {
    Hr
    Write-Host ""
    Write-Host "  AGENTVERSE FULL-STACK BOOTSTRAP" -ForegroundColor White
    Write-Host "  Starting all services for 100% functionality" -ForegroundColor Gray
    Write-Host ""
    Hr

    # 1. Prerequisites
    if (-not (Test-Prerequisites)) {
        Hr; Err "Fix the errors above and re-run."; Hr; return
    }

    # 2. npm deps
    if (-not (Install-NpmDeps)) { return }

    # 3. CAO runtime
    $caoOk = Start-CaoDocker

    # 4. Vite dev
    $devOk = Start-ViteDev

    # Summary
    Hr
    Write-Host ""
    if ($caoOk -and $devOk) {
        Write-Host "  ALL SERVICES RUNNING - AgentVerse is at 100%!" -ForegroundColor Green
    } elseif ($devOk) {
        Write-Host "  PARTIAL - Vite up, CAO failed (see errors above)" -ForegroundColor Yellow
    } else {
        Write-Host "  STARTUP ISSUES - check errors above" -ForegroundColor Red
    }
    Write-Host ""

    Write-Host "  Service          Port    Status" -ForegroundColor White
    Write-Host "  -------          ----    ------" -ForegroundColor DarkGray
    $cc = if ($caoOk) { 'Green' } else { 'Red' }
    $dc = if ($devOk) { 'Green' } else { 'Red' }
    Write-Host "  CAO Runtime      $CaoPort    " -NoNewline; Write-Host $(if ($caoOk) { "HEALTHY" } else { "DOWN" }) -ForegroundColor $cc
    Write-Host "  Vite Dev         $DevPort    " -NoNewline; Write-Host $(if ($devOk) { "HEALTHY" } else { "DOWN" }) -ForegroundColor $dc
    Write-Host ""
    Write-Host "  App URL:     http://localhost:$DevPort/" -ForegroundColor White
    Write-Host "  CAO URL:     $CaoBaseUrl" -ForegroundColor White
    Write-Host ""
    Write-Host "  Stop all:    .\bootstrap.ps1 stop" -ForegroundColor Gray
    Write-Host "  Status:      .\bootstrap.ps1 status" -ForegroundColor Gray
    Write-Host "  CAO logs:    wsl docker logs -f $CaoContainer" -ForegroundColor Gray
    Hr

    try { Start-Process "http://localhost:$DevPort/" } catch {}
}

function Cmd-Stop {
    Header "Stopping all AgentVerse services"

    Step "Stopping Vite (port $DevPort)..."
    Stop-ByPort $DevPort
    # Kill any orphaned node processes from previous Vite runs
    $orphans = Get-Process node -ErrorAction SilentlyContinue
    if ($orphans) {
        $count = @($orphans).Count
        $orphans | Stop-Process -Force -ErrorAction SilentlyContinue
        Info "Cleaned up $count orphaned node processes"
    }
    Ok "Vite stopped"

    Step "Stopping CAO container..."
    Invoke-Wsl "docker rm -f $CaoContainer 2>/dev/null" | Out-Null
    Ok "CAO stopped"

    Hr; Ok "All services stopped"; Hr
}

function Cmd-Status {
    Hr
    Header "AgentVerse service status"

    $caoCode = Test-Url "$CaoBaseUrl/health"
    $devCode = Test-Url "http://localhost:$DevPort/"

    Write-Host ""
    Write-Host "  Service          Port    Status" -ForegroundColor White
    Write-Host "  -------          ----    ------" -ForegroundColor DarkGray

    $caoColor = if ($caoCode -eq 200) { 'Green' } else { 'Red' }
    $devColor = if ($devCode -eq 200) { 'Green' } else { 'Red' }
    $caoLabel = if ($caoCode -eq 200) { "HEALTHY (HTTP 200)" } else { "DOWN (HTTP $caoCode)" }
    $devLabel = if ($devCode -eq 200) { "HEALTHY (HTTP 200)" } else { "DOWN (HTTP $devCode)" }

    Write-Host "  CAO Runtime      $CaoPort    " -NoNewline; Write-Host $caoLabel -ForegroundColor $caoColor
    Write-Host "  Vite Dev         $DevPort    " -NoNewline; Write-Host $devLabel -ForegroundColor $devColor
    Write-Host ""

    if ($caoCode -eq 200 -and $devCode -eq 200) {
        Write-Host "  Overall: 100% operational" -ForegroundColor Green
    } elseif ($devCode -eq 200) {
        Write-Host "  Overall: Partial - CAO is down" -ForegroundColor Yellow
    } else {
        Write-Host "  Overall: Services not running" -ForegroundColor Red
        Write-Host "  Run: .\bootstrap.ps1 start" -ForegroundColor Gray
    }
    Write-Host ""
    Hr
}

function Cmd-Help {
    @(
        "",
        "  AGENTVERSE FULL-STACK BOOTSTRAP",
        "  ================================",
        "",
        "  Starts ALL services to run AgentVerse at 100% functionality.",
        "",
        "  Services:",
        "    CAO Runtime   Docker container via WSL (port $CaoPort)",
        "                  Orchestration backend: deploys, terminals, flows, health",
        "",
        "    Vite Dev       npm run dev with HMR (port $DevPort)",
        "                  Frontend SPA development server",
        "",
        "  Commands:",
        "    .\bootstrap.ps1              Start everything (default)",
        "    .\bootstrap.ps1 stop         Stop all services",
        "    .\bootstrap.ps1 status       Show health",
        "    .\bootstrap.ps1 help         This help",
        "",
        "  Environment overrides:",
        "    CAO_PORT              Default 9889",
        "    DEV_PORT              Default 5173",
        "    VITE_CAO_BASE_URL    Default http://127.0.0.1:9889",
        "",
        "  First run builds the Docker image (~2 min). Subsequent runs are fast.",
        ""
    ) | ForEach-Object { Write-Host $_ }
}

# ── Dispatch ───────────────────────────────────────────────────────────────
switch ($Command) {
    ''       { Cmd-Start }
    'start'  { Cmd-Start }
    'stop'   { Cmd-Stop }
    'status' { Cmd-Status }
    'help'   { Cmd-Help }
    default  { Cmd-Help }
}
