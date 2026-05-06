param(
    [string]$ServerIP = $env:SERVER_IP,
    [string]$SSHUser = $(if ($env:SSH_USER) { $env:SSH_USER } else { "root" }),
    [string]$SSHPassword = $env:SSH_PASSWORD,
    [string]$SSHHostKey = $env:SSH_HOST_KEY,
    [string]$ListenAddr = $(if ($env:LISTEN_ADDR) { $env:LISTEN_ADDR } else { ":49321" }),
    [string]$AdminUser = $(if ($env:ADMIN_USER) { $env:ADMIN_USER } else { "admin" }),
    [string]$AdminPassword = $(if ($env:ADMIN_PASSWORD) { $env:ADMIN_PASSWORD } else { "mSm-4Qb9-vY2-Zt7-Kp5" }),
    [string]$InstallDir = $(if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "/opt/mihomo-sub-manager" }),
    [string]$DataDir = $(if ($env:DATA_DIR) { $env:DATA_DIR } else { "/var/lib/mihomo-sub-manager" })
)

$ErrorActionPreference = "Stop"

if (-not $ServerIP -or -not $SSHPassword) {
    throw "ServerIP and SSHPassword are required. Pass -ServerIP/-SSHPassword or set SERVER_IP/SSH_PASSWORD."
}

$ProjectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$DistDir = Join-Path $ProjectRoot "dist"
$LocalPuttyDir = Join-Path $ProjectRoot ".tools\putty"
$PackageName = "mihomo-sub-manager-linux-amd64.tar.gz"
$BinaryPath = Join-Path $DistDir "mihomo-sub-manager"
$PackagePath = Join-Path $DistDir $PackageName
$RemotePackage = "/tmp/$PackageName"
$ServiceName = "mihomo-sub-manager"

function Find-CommandOrFail {
    param([string]$Name, [string]$InstallHint)
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if (-not $cmd) {
        throw "$Name not found. $InstallHint"
    }
    return $cmd.Source
}

function Find-PuttyTool {
    param([string]$Name)
    $local = Join-Path $LocalPuttyDir $Name
    if (Test-Path $local) {
        return $local
    }
    return Find-CommandOrFail $Name "Install PuTTY and make sure $Name is in PATH."
}

function ShellQuote {
    param([string]$Value)
    return "'" + ($Value -replace "'", "'\''") + "'"
}

$plink = Find-PuttyTool "plink.exe"
$pscp = Find-PuttyTool "pscp.exe"
$tar = Find-CommandOrFail "tar.exe" "Install Git for Windows or use Windows 10/11 built-in tar.exe."

New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

Write-Host "Building Linux AMD64 binary..."
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
Push-Location $ProjectRoot
try {
    go build -trimpath -ldflags="-s -w" -o $BinaryPath .
}
finally {
    Pop-Location
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
}

if (Test-Path $PackagePath) {
    Remove-Item $PackagePath -Force
}

Write-Host "Packaging $PackageName..."
Push-Location $DistDir
try {
    & $tar -czf $PackageName "mihomo-sub-manager"
}
finally {
    Pop-Location
}

Write-Host "Uploading package to $SSHUser@$ServerIP..."
$pscpArgs = @("-batch", "-pw", $SSHPassword)
if ($SSHHostKey) {
    $pscpArgs += @("-hostkey", $SSHHostKey)
}
$pscpArgs += @($PackagePath, "$SSHUser@$ServerIP`:$RemotePackage")
& $pscp @pscpArgs
if ($LASTEXITCODE -ne 0) {
    throw "Upload failed with exit code $LASTEXITCODE"
}

$quotedAdminUser = ShellQuote $AdminUser
$quotedAdminPassword = ShellQuote $AdminPassword
$quotedListenAddr = ShellQuote $ListenAddr
$quotedInstallDir = ShellQuote $InstallDir
$quotedDataDir = ShellQuote $DataDir
$quotedRemotePackage = ShellQuote $RemotePackage
$quotedSSHPassword = ShellQuote $SSHPassword

$remoteScript = @"
set -euo pipefail

SERVICE_NAME="mihomo-sub-manager"
INSTALL_DIR=$quotedInstallDir
DATA_DIR=$quotedDataDir
REMOTE_PACKAGE=$quotedRemotePackage
LISTEN_ADDR=$quotedListenAddr
ADMIN_USER=$quotedAdminUser
ADMIN_PASSWORD=$quotedAdminPassword
SSH_PASSWORD=$quotedSSHPassword
ENV_FILE="/etc/mihomo-sub-manager.env"

if [ "`$(id -u)" -eq 0 ]; then
  SUDO=""
else
  SUDO="sudo -S"
fi

run_root() {
  if [ -z "`$SUDO" ]; then
    sh -c "`$*"
  else
    printf '%s\n' "`$SSH_PASSWORD" | `$SUDO -p '' sh -c "`$*"
  fi
}

escape_env() {
  printf '%s' "`$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

echo "Installing service files..."
run_root "mkdir -p '`$INSTALL_DIR' '`$DATA_DIR'"
run_root "tar -xzf '`$REMOTE_PACKAGE' -C '`$INSTALL_DIR'"
run_root "chmod 0755 '`$INSTALL_DIR/mihomo-sub-manager'"

cat > /tmp/mihomo-sub-manager.env <<ENV
ADDR="`$(escape_env "`$LISTEN_ADDR")"
DB_PATH="`$(escape_env "`$DATA_DIR/mihomo-sub-manager.db")"
ADMIN_USER="`$(escape_env "`$ADMIN_USER")"
ADMIN_PASSWORD="`$(escape_env "`$ADMIN_PASSWORD")"
ENV

cat > /tmp/`$SERVICE_NAME.service <<UNIT
[Unit]
Description=Mihomo Subscription Manager
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=`$ENV_FILE
WorkingDirectory=`$INSTALL_DIR
ExecStart=`$INSTALL_DIR/mihomo-sub-manager
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT

run_root "mv '/tmp/mihomo-sub-manager.env' '`$ENV_FILE'"
run_root "chmod 0600 '`$ENV_FILE'"
run_root "mv '/tmp/`$SERVICE_NAME.service' '/etc/systemd/system/`$SERVICE_NAME.service'"
run_root "systemctl daemon-reload"
run_root "systemctl enable '`$SERVICE_NAME.service'"
run_root "systemctl restart '`$SERVICE_NAME.service'"
run_root "systemctl --no-pager --full status '`$SERVICE_NAME.service'"
"@

Write-Host "Installing and restarting systemd service..."
$plinkArgs = @("-batch", "-ssh", "-pw", $SSHPassword)
if ($SSHHostKey) {
    $plinkArgs += @("-hostkey", $SSHHostKey)
}
$plinkArgs += @("$SSHUser@$ServerIP", "bash -s")
$remoteScript | & $plink @plinkArgs
if ($LASTEXITCODE -ne 0) {
    throw "Remote deployment failed with exit code $LASTEXITCODE"
}

Write-Host "Deployed successfully."
Write-Host "URL: http://$ServerIP$ListenAddr"
Write-Host "Admin username: $AdminUser"
Write-Host "Admin password: $AdminPassword"
