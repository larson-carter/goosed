param(
    [Parameter(Mandatory = $true)][string]$Api,
    [Parameter(Mandatory = $true)][string]$Token,
    [Parameter(Mandatory = $true)][string]$MachineId,
    [string]$BinaryPath = "C:\\Program Files\\Goosed\\goosed-agent.exe"
)

$ErrorActionPreference = "Stop"

function Invoke-GoosedFacts {
    param(
        [string]$Api,
        [string]$Token,
        [string]$MachineId
    )

    $os = Get-CimInstance -ClassName Win32_OperatingSystem
    $bios = Get-CimInstance -ClassName Win32_BIOS

    $snapshot = [ordered]@{
        os               = $os.Caption
        version          = $os.Version
        serial           = $bios.SerialNumber
        postinstall_done = $true
    }

    $payload = [ordered]@{
        machine_id = $MachineId
        snapshot   = $snapshot
    } | ConvertTo-Json -Depth 4

    $headers = @{ "Content-Type" = "application/json" }
    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $headers["Authorization"] = "Bearer $Token"
    }

    $endpoint = "$($Api.TrimEnd('/'))/v1/agents/facts"
    Invoke-RestMethod -Method Post -Uri $endpoint -Headers $headers -Body $payload | Out-Null
}

function Write-GoosedConfig {
    param(
        [string]$ConfigPath,
        [string]$Api,
        [string]$Token,
        [string]$MachineId
    )

    $configDir = Split-Path -Path $ConfigPath -Parent
    if (-not (Test-Path -Path $configDir)) {
        New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    }

    $config = [ordered]@{
        api        = $Api
        token      = $Token
        machine_id = $MachineId
    } | ConvertTo-Json -Depth 3

    Set-Content -Path $ConfigPath -Value $config -Encoding UTF8
}

function Register-GoosedService {
    param(
        [string]$BinaryPath
    )

    $serviceName = "GoosedAgent"
    $displayName = "Goosed Agent"

    if (-not (Test-Path -Path $BinaryPath)) {
        throw "Agent binary not found at $BinaryPath"
    }

    $existing = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($null -ne $existing) {
        if ($existing.Status -ne 'Stopped') {
            Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        }
        sc.exe delete $serviceName | Out-Null
        Start-Sleep -Seconds 2
    }

    $quotedBinary = '"' + $BinaryPath + '"'
    New-Service -Name $serviceName -BinaryPathName $quotedBinary -DisplayName $displayName -StartupType Automatic | Out-Null
    Start-Service -Name $serviceName
}

Invoke-GoosedFacts -Api $Api -Token $Token -MachineId $MachineId
Write-GoosedConfig -ConfigPath "C:\\ProgramData\\Goosed\\agent.conf" -Api $Api -Token $Token -MachineId $MachineId
Register-GoosedService -BinaryPath $BinaryPath
