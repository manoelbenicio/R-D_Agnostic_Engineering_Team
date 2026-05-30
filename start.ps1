# AgentVerse system bootstrap (PowerShell) - single entrypoint for Windows.
#
# Usage:
#   .\start.ps1                 # default: production preview (built dist/)
#   .\start.ps1 agentic_system  # alias of default
#   .\start.ps1 dev             # vite dev server with HMR
#   .\start.ps1 stop            # stop preview + dev + CAO container
#   .\start.ps1 status          # report listening ports
#   .\start.ps1 -Help           # this help
#
# See start.sh (Bash variant) for the full design notes. This is a near-1:1 port.

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('agentic_system','prod','dev','stop','status','help','')]
    [string]$Command = 'agentic_system',

    [switch]$Help
)

$ErrorActionPreference = 'Stop'

# Repo root = directory of this script
$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $RepoRoot

# Defaults (override by setting env vars before invocation)
if (-not $env:VITE_CAO_BASE_URL) { $env:VITE_CAO_BASE_URL = 'http://127.0.0.1:9889' }
$PreviewPort     = if ($env:PREVIEW_PORT) { [int]$env:PREVIEW_PORT } else { 4173 }
$DevPort         = if ($env:DEV_PORT)     { [int]$env:DEV_PORT }     else { 5173 }
$LogDir          = if ($env:LOG_DIR)      { $env:LOG_DIR }            else { $env:TEMP }
$CaoDockerImage  = $env:CAO_DOCKER_IMAGE
$CaoStartCmd     = $env:CAO_START_CMD
$OpenBrowser     = if ($env:OPEN_BROWSER) { $env:OPEN_BROWSER }       else { 'auto' }

$PreviewLog = Join-Path $LogDir 'agentverse-preview.log'
$DevLog     = Join-Path $LogDir 'agentverse-dev.log'
$CaoLog     = Join-Path $LogDir 'agentverse-cao.log'

function Step($msg) { Write-Host "> $msg" -ForegroundColor Cyan }
function Ok  ($msg) { Write-Host "  [OK] $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "  [WARN] $msg" -ForegroundColor Yellow }
function Err ($msg) { Write-Host "  [ERR] $msg" -ForegroundColor Red }
function Hr        { Write-Host ('-' * 60) -ForegroundColor DarkGray }

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
        $listeners = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue
        return [bool]$listeners
    } catch {
        return $false
    }
}

function Spawn-Detached($logPath, $command, $argList) {
    # Start the process detached, redirect output to the log file
    $argString = ($argList | ForEach-Object { "`"$_`"" }) -join ' '
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName  = $command
    $psi.Arguments = $argString
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError  = $true
    $psi.CreateNoWindow = $true
    $proc = [System.Diagnostics.Process]::Start($psi)

    # Pump stdout/stderr to the log without blocking
    $writer = [System.IO.StreamWriter]::new($logPath, $false)
    $writer.AutoFlush = $true
    Register-ObjectEvent -InputObject $proc -EventName OutputDataReceived -Action { if ($EventArgs.Data) { $writer.WriteLine($EventArgs.Data) } } | Out-Null
    Register-ObjectEvent -InputObject $proc -EventName ErrorDataReceived  -Action { if ($EventArgs.Data) { $writer.WriteLine($EventArgs.Data) } } | Out-Null
    $proc.BeginOutputReadLine()
    $proc.BeginErrorReadLine()
    return $proc.Id
}

function Stop-Pattern($pattern) {
    Get-Process | Where-Object {
        try { ($_.Path -and ($_.Path -like "*$pattern*")) -or ($_.CommandLine -match [regex]::Escape($pattern)) } catch { $false }
    } | ForEach-Object { try { $_.Kill() } catch {} }
}

function Show-Help {
    $helpText = @(
        "AgentVerse bootstrap (PowerShell)",
        "",
        "Usage: .\start.ps1 [COMMAND]",
        "",
        "Commands:",
        "  agentic_system    (default) Build if needed, probe CAO, start production preview",
        "  prod              Alias of agentic_system",
        "  dev               Start vite dev server (HMR, requires real CAO)",
        "  stop              Stop everything this script started",
        "  status            Report what is listening on AgentVerse + CAO ports",
        "  help, -Help       This help",
        "",
        "Environment overrides:",
        "  VITE_CAO_BASE_URL   CAO endpoint to probe        (default http://127.0.0.1:9889)",
        "  PREVIEW_PORT        Production preview port      (default 4173)",
        "  DEV_PORT            Vite dev server port         (default 5173)",
        "  CAO_DOCKER_IMAGE    Docker image to start CAO    (default empty - opt-in)",
        "  CAO_START_CMD       Shell command to start CAO   (default empty - opt-in)",
        "  OPEN_BROWSER        auto | yes | no              (default auto)",
        "",
        "CAO is an EXTERNAL service (master spec Section 13). It is not bundled with this",
        "repository. Set CAO_DOCKER_IMAGE or CAO_START_CMD to have this script start",
        "it for you, or run CAO yourself before invoking start.ps1.",
        "",
        "Examples:",
        "  .\start.ps1",
        "  .\start.ps1 dev",
        "  `$env:CAO_DOCKER_IMAGE='cao-server:latest'; .\start.ps1",
        "  .\start.ps1 stop"
    )
    $helpText | ForEach-Object { Write-Host $_ }
}

function Test-Node {
    Step 'Checking Node + npm'
    if (-not (Get-Command node -ErrorAction SilentlyContinue)) { Err 'node not found'; exit 2 }
    if (-not (Get-Command npm  -ErrorAction SilentlyContinue)) { Err 'npm not found';  exit 2 }
    $nv = (& node -v).TrimStart('v')
    $mv = (& npm -v)
    if ([version]$nv -lt [version]'20.10.0') { Err "Node $nv < 20.10 required"; exit 2 }
    if ([version]$mv -lt [version]'10.2.0')  { Err "npm $mv < 10.2 required";   exit 2 }
    Ok "node $nv · npm $mv"
}

function Test-Deps {
    Step 'Checking dependencies'
    if (-not (Test-Path 'node_modules')) {
        Warn 'node_modules\ missing - running npm ci'
        & npm ci
    }
    Ok 'dependencies installed'
}

function Build-IfStale {
    Step 'Checking production bundle'
    $needBuild = $false
    if (-not (Test-Path 'dist\index.html')) {
        $needBuild = $true
    } else {
        $distMtime = (Get-Item 'dist\index.html').LastWriteTimeUtc
        $candidates = @('src','public','index.html','vite.config.ts','package.json','.env.production') |
                      Where-Object { Test-Path $_ } |
                      Get-ChildItem -Recurse -File -ErrorAction SilentlyContinue
        if ($candidates | Where-Object { $_.LastWriteTimeUtc -gt $distMtime } | Select-Object -First 1) {
            $needBuild = $true
        }
    }
    if ($needBuild) {
        Step 'Building production bundle (npm run build)'
        & npm run build
        Ok 'build complete'
    } else {
        Ok 'dist\ is up-to-date'
    }
    if (Test-Path 'dist\mockServiceWorker.js') {
        Remove-Item 'dist\mockServiceWorker.js' -Force
        Warn 'stripped dist\mockServiceWorker.js (no mock infrastructure in production)'
    }
}

function Start-CaoIfConfigured {
    Step "Probing CAO at $($env:VITE_CAO_BASE_URL)"
    $code = Test-Url "$($env:VITE_CAO_BASE_URL)/health"
    if ($code -eq 200) { Ok "CAO is reachable (HTTP 200)"; return $true }
    Warn "CAO is NOT reachable (HTTP $code)"

    if ($CaoDockerImage) {
        Step "Starting CAO from Docker image: $CaoDockerImage"
        if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { Err 'docker not found'; return $false }
        & docker run -d --name agentverse-cao -p 9889:9889 -e CAO_CORS_ORIGINS=http://localhost:5173,http://localhost:4173 -e CAO_ALLOWED_HOSTS=127.0.0.1,localhost -e CAO_WS_ALLOWED_CLIENTS=http://localhost:5173,http://localhost:4173 $CaoDockerImage 2>$CaoLog | Out-Null
        for ($i = 0; $i -lt 30; $i++) {
            Start-Sleep 1
            if ((Test-Url "$($env:VITE_CAO_BASE_URL)/health") -eq 200) { Ok 'CAO is up (HTTP 200)'; return $true }
        }
        Err "CAO did not become healthy within 30s - check $CaoLog"
        return $false
    }

    if ($CaoStartCmd) {
        Step 'Starting CAO via CAO_START_CMD'
        Spawn-Detached -logPath $CaoLog -command 'cmd.exe' -argList @('/c', $CaoStartCmd) | Out-Null
        for ($i = 0; $i -lt 30; $i++) {
            Start-Sleep 1
            if ((Test-Url "$($env:VITE_CAO_BASE_URL)/health") -eq 200) { Ok 'CAO is up (HTTP 200)'; return $true }
        }
        Err "CAO did not become healthy within 30s - check $CaoLog"
        return $false
    }

    Write-Host "  ! CAO is required for canvas deploys, terminal streaming, and flows." -ForegroundColor Yellow
    Write-Host "    Per master spec Section 13 it is an EXTERNAL service. To start it, do ONE of:" -ForegroundColor Yellow
    Write-Host "      1) `$env:CAO_DOCKER_IMAGE = 'cao-server:latest'; .\start.ps1" -ForegroundColor Yellow
    Write-Host "      2) `$env:CAO_START_CMD = 'uv run cao serve --port 9889'; .\start.ps1" -ForegroundColor Yellow
    Write-Host "      3) Start CAO yourself in another terminal at $($env:VITE_CAO_BASE_URL)." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "    The SPA will start anyway and the Health page will show CAO offline." -ForegroundColor Yellow
    return $true
}

function Start-Preview {
    Step "Starting production preview on :$PreviewPort"
    if (Test-Listening $PreviewPort) {
        Warn "port $PreviewPort already in use - assuming a preview is already running"
    } else {
        Spawn-Detached -logPath $PreviewLog -command 'npm.cmd' -argList @('run','preview') | Out-Null
        for ($i = 0; $i -lt 15; $i++) {
            Start-Sleep 1
            if ((Test-Url "http://localhost:$PreviewPort/") -eq 200) { Ok 'preview responding'; break }
        }
    }
    if ((Test-Url "http://localhost:$PreviewPort/") -ne 200) { Err "preview did not come up - check $PreviewLog"; return $false }
    return $true
}

function Start-Dev {
    Step "Starting vite dev server on :$DevPort"
    if (Test-Listening $DevPort) {
        Warn "port $DevPort already in use - assuming dev server is already running"
    } else {
        Spawn-Detached -logPath $DevLog -command 'npm.cmd' -argList @('run','dev') | Out-Null
        for ($i = 0; $i -lt 15; $i++) {
            Start-Sleep 1
            if ((Test-Url "http://localhost:$DevPort/") -eq 200) { Ok 'dev responding'; break }
        }
    }
    if ((Test-Url "http://localhost:$DevPort/") -ne 200) { Err "dev server did not come up - check $DevLog"; return $false }
    return $true
}

function Open-Browser($url) {
    if ($OpenBrowser -eq 'no') { return }
    try { Start-Process $url } catch {}
}

function Cmd-Status {
    Hr
    Step 'AgentVerse system status'
    Hr
    Write-Host ("  preview ({0}) : HTTP {1}" -f $PreviewPort, (Test-Url "http://localhost:$PreviewPort/"))
    Write-Host ("  dev     ({0}) : HTTP {1}" -f $DevPort,     (Test-Url "http://localhost:$DevPort/"))
    Write-Host ("  CAO     : {0} HTTP {1}"   -f $env:VITE_CAO_BASE_URL, (Test-Url "$($env:VITE_CAO_BASE_URL)/health"))
    Hr
}

function Cmd-Stop {
    Step 'Stopping everything'
    Stop-Pattern 'vite preview'
    Stop-Pattern 'vite dev'
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        & docker rm -f agentverse-cao 2>$null | Out-Null
    }
    Ok 'stopped'
}

function Cmd-UpProd {
    Hr; Step 'Bootstrapping AgentVerse - production'; Hr
    Test-Node
    Test-Deps
    Build-IfStale
    [void](Start-CaoIfConfigured)
    if (-not (Start-Preview)) { exit 1 }
    Hr
    Ok 'AgentVerse production preview is live'
    Write-Host ("    URL          : http://localhost:{0}" -f $PreviewPort) -ForegroundColor White
    Write-Host ("    CAO endpoint : {0} (HTTP {1})" -f $env:VITE_CAO_BASE_URL, (Test-Url "$($env:VITE_CAO_BASE_URL)/health"))
    Write-Host ("    Logs         : {0}" -f $PreviewLog)
    Write-Host ("    Stop         : .\start.ps1 stop")
    Hr
    Open-Browser "http://localhost:$PreviewPort"
}

function Cmd-UpDev {
    Hr; Step 'Bootstrapping AgentVerse - dev (HMR)'; Hr
    Test-Node
    Test-Deps
    [void](Start-CaoIfConfigured)
    if (-not (Start-Dev)) { exit 1 }
    Hr
    Ok 'AgentVerse dev server is live'
    Write-Host ("    URL          : http://localhost:{0}" -f $DevPort) -ForegroundColor White
    Write-Host ("    CAO endpoint : {0} (HTTP {1})" -f $env:VITE_CAO_BASE_URL, (Test-Url "$($env:VITE_CAO_BASE_URL)/health"))
    Write-Host ("    Logs         : {0}" -f $DevLog)
    Write-Host ("    Stop         : .\start.ps1 stop")
    Hr
    Open-Browser "http://localhost:$DevPort"
}

# Dispatch
if ($Help -or $Command -eq 'help') { Show-Help; exit 0 }
switch ($Command) {
    ''                 { Cmd-UpProd }
    'agentic_system'   { Cmd-UpProd }
    'prod'             { Cmd-UpProd }
    'dev'              { Cmd-UpDev }
    'stop'             { Cmd-Stop }
    'status'           { Cmd-Status }
    default            { Show-Help; exit 64 }
}
