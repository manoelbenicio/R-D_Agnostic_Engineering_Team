<#
.SYNOPSIS
Installs a durable, trusted HTTPS reverse proxy for OmniRoute on Windows.

.DESCRIPTION
Creates https://192.168.1.9/ and proxies it to the already-working
http://127.0.0.1:8128 WSL/SSH tunnel. This fixes OmniRoute's use of
crypto.subtle, which browsers disable on plain-HTTP LAN-IP origins.

The script installs a pinned official Caddy release as a Windows service,
uses Caddy's internal CA for the private IP certificate, trusts only that
local root certificate, and verifies HTTPS end to end.

It does NOT change OmniRoute, WSL, Tailscale, the SSH tunnel, netsh
portproxy, passwords, or existing firewall rules.

Run from an Administrator PowerShell:
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\setup-omniroute-https.ps1

Check status:
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\setup-omniroute-https.ps1 -Action Status

Rollback completely:
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\setup-omniroute-https.ps1 -Action Uninstall
#>

[CmdletBinding()]
param(
    [ValidateSet("Install", "Status", "Uninstall")]
    [string]$Action = "Install",

    [ValidatePattern('^\d{1,3}(\.\d{1,3}){3}$')]
    [string]$WindowsIp = "192.168.1.9",

    [ValidateRange(1, 65535)]
    [int]$BackendPort = 8128,

    [switch]$NoBrowser
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Dependency is intentionally pinned. The archive is verified against the
# checksum manifest from the same official GitHub release before execution.
$CaddyVersion = "2.11.4"
$ServiceName = "OmniRouteHttps"
$DisplayName = "OmniRoute HTTPS Proxy"
$InstallRoot = Join-Path $env:ProgramData "OmniRouteHttps"
$CaddyExe = Join-Path $InstallRoot "caddy.exe"
$Caddyfile = Join-Path $InstallRoot "Caddyfile"
$DataRoot = Join-Path $InstallRoot "data"
$ConfigRoot = Join-Path $InstallRoot "config"
$StateFile = Join-Path $InstallRoot "install-state.json"
$BackendUrl = "http://127.0.0.1:$BackendPort/login"
$HttpsUrl = "https://$WindowsIp/login"
$ReleaseBase = "https://github.com/caddyserver/caddy/releases/download/v$CaddyVersion"
$ArchiveName = "caddy_${CaddyVersion}_windows_amd64.zip"
$ChecksumsName = "caddy_${CaddyVersion}_checksums.txt"

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Fail {
    param([string]$Message)
    throw $Message
}

function Test-IsAdministrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Assert-Administrator {
    if (-not (Test-IsAdministrator)) {
        Fail "Run this file from an Administrator PowerShell window."
    }
}

function Invoke-HttpCheck {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [int]$TimeoutSeconds = 10
    )

    $response = Invoke-WebRequest -UseBasicParsing -Uri $Uri -TimeoutSec $TimeoutSeconds
    if ($response.StatusCode -ne 200) {
        Fail "$Uri returned HTTP $($response.StatusCode), expected 200."
    }
    return $response
}

function Get-ListeningProcessDescription {
    param([int]$Port)

    $listeners = @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue)
    if ($listeners.Count -eq 0) {
        return $null
    }

    $descriptions = foreach ($listener in $listeners) {
        $processName = "unknown"
        try {
            $processName = (Get-Process -Id $listener.OwningProcess -ErrorAction Stop).ProcessName
        } catch {}
        "$($listener.LocalAddress):$Port pid=$($listener.OwningProcess) process=$processName"
    }
    return ($descriptions -join "; ")
}

function Get-ExpectedChecksum {
    param(
        [Parameter(Mandatory = $true)][string]$ManifestPath,
        [Parameter(Mandatory = $true)][string]$FileName
    )

    $escapedName = [Regex]::Escape($FileName)
    foreach ($line in Get-Content -LiteralPath $ManifestPath) {
        if ($line -match "^(?<hash>[A-Fa-f0-9]{64})\s+\*?$escapedName$") {
            return $Matches.hash.ToLowerInvariant()
        }
        if ($line -match "^SHA256 \($escapedName\) = (?<hash>[A-Fa-f0-9]{64})$") {
            return $Matches.hash.ToLowerInvariant()
        }
    }

    Fail "The official checksum manifest does not contain $FileName."
}

function Download-And-VerifyCaddy {
    param([Parameter(Mandatory = $true)][string]$DestinationDirectory)

    $archivePath = Join-Path $DestinationDirectory $ArchiveName
    $checksumsPath = Join-Path $DestinationDirectory $ChecksumsName
    $sidecarArchive = Join-Path $PSScriptRoot $ArchiveName
    $sidecarChecksums = Join-Path $PSScriptRoot $ChecksumsName

    if ((Test-Path -LiteralPath $sidecarArchive) -and (Test-Path -LiteralPath $sidecarChecksums)) {
        Write-Step "Using Caddy artifacts stored beside this script."
        Copy-Item -LiteralPath $sidecarArchive -Destination $archivePath
        Copy-Item -LiteralPath $sidecarChecksums -Destination $checksumsPath
    } else {
        Write-Step "Downloading pinned Caddy v$CaddyVersion from the official GitHub release."
        $headers = @{ "User-Agent" = "OmniRouteHttpsSetup/1.0" }
        try {
            Invoke-WebRequest -UseBasicParsing -Headers $headers -Uri "$ReleaseBase/$ChecksumsName" -OutFile $checksumsPath
            Invoke-WebRequest -UseBasicParsing -Headers $headers -Uri "$ReleaseBase/$ArchiveName" -OutFile $archivePath
        } catch {
            Fail "Caddy download failed. No unverified binary was installed. Download $ChecksumsName and $ArchiveName from $ReleaseBase, place both beside this script, then run it again. Original error: $($_.Exception.Message)"
        }
    }

    $expected = Get-ExpectedChecksum -ManifestPath $checksumsPath -FileName $ArchiveName
    $actual = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        Fail "Caddy archive checksum mismatch. Expected $expected but received $actual."
    }

    Write-Success "Caddy archive SHA-256 matches the official release manifest."
    return $archivePath
}

function Get-ServiceEnvironmentRegistryPath {
    return "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName"
}

function Wait-ForServiceRunning {
    param([int]$TimeoutSeconds = 30)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        $service = Get-Service -Name $ServiceName -ErrorAction Stop
        if ($service.Status -eq "Running") {
            return
        }
        if ($service.Status -eq "Stopped") {
            Fail "The $ServiceName service stopped during startup."
        }
        Start-Sleep -Milliseconds 500
    } while ((Get-Date) -lt $deadline)

    Fail "Timed out waiting for the $ServiceName service to start."
}

function Wait-ForFile {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [int]$TimeoutSeconds = 30
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        if (Test-Path -LiteralPath $Path) {
            return
        }
        Start-Sleep -Milliseconds 500
    } while ((Get-Date) -lt $deadline)

    Fail "Timed out waiting for $Path."
}

function Wait-ForHttps {
    param([int]$TimeoutSeconds = 30)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $lastError = $null
    do {
        try {
            $null = Invoke-HttpCheck -Uri $HttpsUrl -TimeoutSeconds 5
            return
        } catch {
            $lastError = $_.Exception.Message
            Start-Sleep -Seconds 1
        }
    } while ((Get-Date) -lt $deadline)

    Fail "HTTPS verification failed for $HttpsUrl. Last error: $lastError"
}

function Show-Status {
    Write-Host ""
    Write-Host "OmniRoute HTTPS status" -ForegroundColor Cyan
    Write-Host "  Backend: $BackendUrl"
    Write-Host "  Browser: $HttpsUrl"

    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($null -eq $service) {
        Write-Host "  Service: not installed" -ForegroundColor Yellow
    } else {
        $color = if ($service.Status -eq "Running") { "Green" } else { "Yellow" }
        Write-Host "  Service: $($service.Status)" -ForegroundColor $color
    }

    try {
        $null = Invoke-HttpCheck -Uri $BackendUrl -TimeoutSeconds 5
        Write-Success "Existing WSL/SSH backend returned HTTP 200."
    } catch {
        Write-Host "[FAIL] Backend check: $($_.Exception.Message)" -ForegroundColor Red
    }

    try {
        $null = Invoke-HttpCheck -Uri $HttpsUrl -TimeoutSeconds 5
        Write-Success "Trusted Windows HTTPS endpoint returned HTTP 200."
    } catch {
        Write-Host "[FAIL] HTTPS check: $($_.Exception.Message)" -ForegroundColor Red
    }
}

function Install-OmniRouteHttps {
    Assert-Administrator

    if (-not [Environment]::Is64BitOperatingSystem) {
        Fail "This script requires 64-bit Windows."
    }

    if ($env:PROCESSOR_ARCHITECTURE -notmatch "^(AMD64|x86)$" -and $env:PROCESSOR_ARCHITEW6432 -ne "AMD64") {
        Fail "This pinned package targets Windows x64/AMD64; detected architecture '$($env:PROCESSOR_ARCHITECTURE)'."
    }

    $assignedIp = Get-NetIPAddress -AddressFamily IPv4 -IPAddress $WindowsIp -ErrorAction SilentlyContinue
    if ($null -eq $assignedIp) {
        Fail "$WindowsIp is not currently assigned to this Windows machine. No changes were made."
    }

    Write-Step "Checking the existing WSL/SSH tunnel at $BackendUrl."
    $null = Invoke-HttpCheck -Uri $BackendUrl
    Write-Success "Existing backend returned HTTP 200."

    if ($null -ne (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
        Fail "Service $ServiceName already exists. Run this script with -Action Status or -Action Uninstall."
    }

    if (Test-Path -LiteralPath $InstallRoot) {
        Fail "$InstallRoot already exists. Move or remove it after reviewing its contents, then run again."
    }

    $listener = Get-ListeningProcessDescription -Port 443
    if ($null -ne $listener) {
        Fail "TCP port 443 is already in use by: $listener. No changes were made."
    }

    $tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("omniroute-https-" + [Guid]::NewGuid().ToString("N"))
    $serviceCreated = $false
    $trustedThumbprint = $null
    $installDirectoryCreated = $false

    try {
        New-Item -ItemType Directory -Path $tempRoot -Force | Out-Null
        $archivePath = Download-And-VerifyCaddy -DestinationDirectory $tempRoot

        $expanded = Join-Path $tempRoot "expanded"
        Expand-Archive -LiteralPath $archivePath -DestinationPath $expanded -Force
        $downloadedExe = Join-Path $expanded "caddy.exe"
        if (-not (Test-Path -LiteralPath $downloadedExe)) {
            Fail "Verified archive does not contain caddy.exe."
        }

        New-Item -ItemType Directory -Path $InstallRoot, $DataRoot, $ConfigRoot -Force | Out-Null
        $installDirectoryCreated = $true

        # The internal CA private key is security-sensitive. Remove inherited
        # ACLs and grant access only to LocalSystem and built-in Administrators.
        & icacls.exe $InstallRoot /inheritance:r /grant:r '*S-1-5-18:(OI)(CI)F' '*S-1-5-32-544:(OI)(CI)F' | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Fail "Could not restrict permissions on $InstallRoot."
        }

        Copy-Item -LiteralPath $downloadedExe -Destination $CaddyExe

        $caddyConfig = @"
{
    auto_https disable_redirects
    skip_install_trust
}

https://$WindowsIp {
    bind $WindowsIp
    tls internal
    reverse_proxy 127.0.0.1:$BackendPort
}
"@
        Set-Content -LiteralPath $Caddyfile -Value $caddyConfig -Encoding ASCII

        # Use the same deterministic CA storage for validation and for the
        # LocalSystem service. Trust is installed explicitly and recorded
        # below, so rollback removes only the certificate created here.
        $env:XDG_DATA_HOME = $DataRoot
        $env:XDG_CONFIG_HOME = $ConfigRoot

        Write-Step "Validating the generated Caddy configuration."
        & $CaddyExe validate --config $Caddyfile --adapter caddyfile
        if ($LASTEXITCODE -ne 0) {
            Fail "Caddy rejected the generated configuration."
        }
        Write-Success "Caddy configuration is valid."

        $binaryPath = '"' + $CaddyExe + '" run --config "' + $Caddyfile + '" --adapter caddyfile'
        Write-Step "Creating the automatic Windows service $ServiceName."
        New-Service -Name $ServiceName `
            -BinaryPathName $binaryPath `
            -DisplayName $DisplayName `
            -Description "Trusted HTTPS proxy for the local OmniRoute WSL tunnel" `
            -StartupType Automatic `
            -DependsOn "iphlpsvc" | Out-Null
        $serviceCreated = $true

        # Give LocalSystem a stable, known Caddy data/config location. This is
        # where its unique local CA and private key are persisted.
        $serviceRegistryPath = Get-ServiceEnvironmentRegistryPath
        New-ItemProperty -Path $serviceRegistryPath -Name Environment -PropertyType MultiString -Value @(
            "XDG_DATA_HOME=$DataRoot",
            "XDG_CONFIG_HOME=$ConfigRoot"
        ) -Force | Out-Null
        New-ItemProperty -Path $serviceRegistryPath -Name DelayedAutoStart -PropertyType DWord -Value 1 -Force | Out-Null

        Start-Service -Name $ServiceName
        Wait-ForServiceRunning
        Write-Success "Windows HTTPS service is running."

        $rootCertificate = Join-Path $DataRoot "caddy\pki\authorities\local\root.crt"
        Wait-ForFile -Path $rootCertificate

        Write-Step "Trusting this installation's Caddy Local Authority in the Windows machine certificate store."
        $imported = Import-Certificate -FilePath $rootCertificate -CertStoreLocation "Cert:\LocalMachine\Root"
        $trustedThumbprint = $imported.Thumbprint
        if ([string]::IsNullOrWhiteSpace($trustedThumbprint)) {
            Fail "The local Caddy root certificate was not imported."
        }
        Write-Success "Trusted local root certificate $trustedThumbprint."

        $state = [ordered]@{
            schemaVersion = 1
            installedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
            caddyVersion = $CaddyVersion
            serviceName = $ServiceName
            windowsIp = $WindowsIp
            backendPort = $BackendPort
            rootCertificateThumbprint = $trustedThumbprint
            browserUrl = $HttpsUrl
        }
        $state | ConvertTo-Json | Set-Content -LiteralPath $StateFile -Encoding UTF8

        Write-Step "Verifying trusted HTTPS end to end."
        Wait-ForHttps
        Write-Success "$HttpsUrl returned HTTP 200 with a trusted certificate."

        Write-Host ""
        Write-Success "OmniRoute HTTPS installation completed."
        Write-Host "Open: $HttpsUrl" -ForegroundColor Green
        Write-Host ""
        Write-Host "This script did not modify WSL, Tailscale, the SSH tunnel, portproxy, OmniRoute, or firewall rules."

        if (-not $NoBrowser) {
            Start-Process $HttpsUrl
        }
    } catch {
        Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Rolling back changes made by this run..." -ForegroundColor Yellow

        if ($serviceCreated) {
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            & sc.exe delete $ServiceName | Out-Null
            Start-Sleep -Seconds 1
        }

        if (-not [string]::IsNullOrWhiteSpace($trustedThumbprint)) {
            Remove-Item -LiteralPath "Cert:\LocalMachine\Root\$trustedThumbprint" -Force -ErrorAction SilentlyContinue
        }

        if ($installDirectoryCreated -and (Test-Path -LiteralPath $InstallRoot)) {
            Remove-Item -LiteralPath $InstallRoot -Recurse -Force -ErrorAction SilentlyContinue
        }

        throw
    } finally {
        if (Test-Path -LiteralPath $tempRoot) {
            Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

function Uninstall-OmniRouteHttps {
    Assert-Administrator

    $state = $null
    if (Test-Path -LiteralPath $StateFile) {
        $state = Get-Content -LiteralPath $StateFile -Raw | ConvertFrom-Json
    }

    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($null -ne $service) {
        Write-Step "Stopping and deleting $ServiceName."
        if ($service.Status -ne "Stopped") {
            Stop-Service -Name $ServiceName -Force
        }
        & sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 1
        Write-Success "Windows service removed."
    }

    if ($null -ne $state -and -not [string]::IsNullOrWhiteSpace([string]$state.rootCertificateThumbprint)) {
        $certificatePath = "Cert:\LocalMachine\Root\$($state.rootCertificateThumbprint)"
        if (Test-Path -LiteralPath $certificatePath) {
            Write-Step "Removing only the local CA certificate installed by this script."
            Remove-Item -LiteralPath $certificatePath -Force
            Write-Success "Local CA trust removed."
        }
    } elseif (Test-Path -LiteralPath $InstallRoot) {
        Write-Warning "State file is missing; certificate trust was not removed automatically. Review the Windows certificate store for 'Caddy Local Authority'."
    }

    if (Test-Path -LiteralPath $InstallRoot) {
        Remove-Item -LiteralPath $InstallRoot -Recurse -Force
        Write-Success "$InstallRoot removed."
    }

    Write-Host ""
    Write-Success "Rollback complete. Existing WSL/tunnel/portproxy configuration was untouched."
}

try {
    # Ensure TLS 1.2 is enabled for Windows PowerShell downloads without
    # disabling any stronger protocols already selected by the machine.
    if ($null -ne ([Net.ServicePointManager]::SecurityProtocol)) {
        [Net.ServicePointManager]::SecurityProtocol =
            [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    }

    switch ($Action) {
        "Install" { Install-OmniRouteHttps }
        "Status" { Show-Status }
        "Uninstall" { Uninstall-OmniRouteHttps }
    }
} catch {
    Write-Host ""
    Write-Host "FAILED: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
