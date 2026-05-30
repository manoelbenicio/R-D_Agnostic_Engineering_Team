# preflight-check.ps1 — Verify all prerequisites for AgentVerse canvas deployment
$ErrorActionPreference = 'Continue'

Write-Host ""
Write-Host "  AGENTVERSE PRE-FLIGHT CHECK" -ForegroundColor White
Write-Host "  ===========================" -ForegroundColor DarkGray
Write-Host ""

$allOk = $true

# 1. CAO Runtime
Write-Host "  1. CAO Runtime (port 9889)" -ForegroundColor Cyan
try {
    $health = Invoke-RestMethod -Uri 'http://127.0.0.1:9889/health' -TimeoutSec 3 -ErrorAction Stop
    if ($health.status -eq 'ok') {
        Write-Host "     [PASS] Healthy" -ForegroundColor Green
    } else {
        Write-Host "     [FAIL] Unhealthy response" -ForegroundColor Red
        $allOk = $false
    }
} catch {
    Write-Host "     [FAIL] Not reachable" -ForegroundColor Red
    $allOk = $false
}

# 2. Vite Dev Server
Write-Host "  2. Vite Dev Server (port 5173)" -ForegroundColor Cyan
try {
    $vite = Invoke-WebRequest -Uri 'http://localhost:5173/' -TimeoutSec 3 -UseBasicParsing -ErrorAction Stop
    Write-Host "     [PASS] HTTP $($vite.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "     [FAIL] Not reachable" -ForegroundColor Red
    $allOk = $false
}

# 3. CLI Providers
Write-Host "  3. CLI Providers Installed" -ForegroundColor Cyan
try {
    $providers = Invoke-RestMethod -Uri 'http://127.0.0.1:9889/agents/providers' -TimeoutSec 3 -ErrorAction Stop
    $targets = @('codex', 'kiro_cli', 'gemini_cli')
    foreach ($name in $targets) {
        $p = $providers | Where-Object { $_.name -eq $name }
        if ($p -and $p.installed) {
            Write-Host "     [PASS] $name - installed ($($p.binary))" -ForegroundColor Green
        } else {
            Write-Host "     [FAIL] $name - NOT installed" -ForegroundColor Red
            $allOk = $false
        }
    }
} catch {
    Write-Host "     [FAIL] Could not query providers" -ForegroundColor Red
    $allOk = $false
}

# 4. Agent Profiles
Write-Host "  4. Agent Profiles Available" -ForegroundColor Cyan
try {
    $profiles = Invoke-RestMethod -Uri 'http://127.0.0.1:9889/agents/profiles' -TimeoutSec 3 -ErrorAction Stop
    foreach ($prof in $profiles) {
        Write-Host "     [PASS] $($prof.name) - $($prof.description)" -ForegroundColor Green
    }
} catch {
    Write-Host "     [FAIL] Could not query profiles" -ForegroundColor Red
    $allOk = $false
}

# 5. CLI Credentials in Container
Write-Host "  5. CLI Credentials Mounted" -ForegroundColor Cyan
$prevEAP = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
$credCheck = & wsl bash -c "docker exec agentverse-cao bash -c 'echo CODEX_CREDS=; ls -la /root/.codex/ 2>&1 | head -3; echo; echo KIRO_CREDS=; ls -la /root/.kiro/ 2>&1 | head -3; echo; echo GEMINI_CREDS=; ls -la /root/.gemini/ 2>&1 | head -3'" 2>&1 | Out-String
$ErrorActionPreference = $prevEAP

$credDirs = @(
    @{Name='Codex'; Pattern='CODEX_CREDS'; Dir='.codex'},
    @{Name='Kiro'; Pattern='KIRO_CREDS'; Dir='.kiro'},
    @{Name='Gemini/Antigravity'; Pattern='GEMINI_CREDS'; Dir='.gemini'}
)
foreach ($cred in $credDirs) {
    if ($credCheck -match "$($cred.Pattern)=`n.*total") {
        Write-Host "     [PASS] ~/$($cred.Dir) mounted" -ForegroundColor Green
    } elseif ($credCheck -match "$($cred.Pattern)=`n.*No such") {
        Write-Host "     [WARN] ~/$($cred.Dir) not found - auth may be needed" -ForegroundColor Yellow
    } else {
        Write-Host "     [WARN] ~/$($cred.Dir) status unclear" -ForegroundColor Yellow
    }
}

# 6. Docker Container Running
Write-Host "  6. Docker Container" -ForegroundColor Cyan
$containerCheck = & wsl bash -c "docker ps --filter name=agentverse-cao --format '{{.Status}}' 2>/dev/null" 2>&1 | Out-String
$containerCheck = $containerCheck.Trim()
$statusLine = ($containerCheck -split "`n" | Where-Object { $_ -match 'Up' } | Select-Object -First 1)
if ($statusLine) {
    Write-Host "     [PASS] agentverse-cao: $statusLine" -ForegroundColor Green
} else {
    Write-Host "     [FAIL] Container not running" -ForegroundColor Red
    $allOk = $false
}

# Summary
Write-Host ""
Write-Host "  -----------------------------------------------" -ForegroundColor DarkGray
if ($allOk) {
    Write-Host "  ALL CHECKS PASSED - You are ready to proceed!" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Next steps:" -ForegroundColor White
    Write-Host "    1. Open http://localhost:5173/" -ForegroundColor Gray
    Write-Host "    2. Click TEMPLATES -> pick a template" -ForegroundColor Gray
    Write-Host "    3. Set providers: kiro_cli, codex, gemini_cli" -ForegroundColor Gray
    Write-Host "    4. Set Working Directory and Deploy" -ForegroundColor Gray
} else {
    Write-Host "  SOME CHECKS FAILED - Fix the issues above" -ForegroundColor Red
}
Write-Host ""
