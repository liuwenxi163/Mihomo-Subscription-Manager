#!/usr/bin/env bash
set -euo pipefail

SERVER_IP="${SERVER_IP:-}"
SSH_USER="${SSH_USER:-root}"
SSH_PASSWORD="${SSH_PASSWORD:-}"
LISTEN_ADDR="${LISTEN_ADDR:-:49321}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-mSm-4Qb9-vY2-Zt7-Kp5}"
INSTALL_DIR="${INSTALL_DIR:-/opt/mihomo-sub-manager}"
DATA_DIR="${DATA_DIR:-/var/lib/mihomo-sub-manager}"

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/deploy-ubuntu.sh --server-ip IP --ssh-password PASSWORD [options]

Options:
  --server-ip IP              Ubuntu server IP or hostname. Required.
  --ssh-user USER             SSH user. Default: root.
  --ssh-password PASSWORD     SSH password. Required when using password SSH.
  --listen-addr ADDR          Service listen address. Default: :49321.
  --admin-user USER           Web admin username. Default: admin.
  --admin-password PASSWORD   Web admin password. Default: mSm-4Qb9-vY2-Zt7-Kp5.
  --install-dir PATH          Remote install directory. Default: /opt/mihomo-sub-manager.
  --data-dir PATH             Remote database directory. Default: /var/lib/mihomo-sub-manager.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --server-ip) SERVER_IP="$2"; shift 2 ;;
    --ssh-user) SSH_USER="$2"; shift 2 ;;
    --ssh-password) SSH_PASSWORD="$2"; shift 2 ;;
    --listen-addr) LISTEN_ADDR="$2"; shift 2 ;;
    --admin-user) ADMIN_USER="$2"; shift 2 ;;
    --admin-password) ADMIN_PASSWORD="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --data-dir) DATA_DIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [ -z "$SERVER_IP" ] || [ -z "$SSH_PASSWORD" ]; then
  usage
  exit 1
fi

for cmd in go tar sshpass scp ssh; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "$cmd is required. On Ubuntu, install with: sudo apt-get install -y sshpass" >&2
    exit 1
  fi
done

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
PACKAGE_NAME="mihomo-sub-manager-linux-amd64.tar.gz"
BINARY_PATH="$DIST_DIR/mihomo-sub-manager"
PACKAGE_PATH="$DIST_DIR/$PACKAGE_NAME"
REMOTE_PACKAGE="/tmp/$PACKAGE_NAME"

mkdir -p "$DIST_DIR"

echo "Building Linux AMD64 binary..."
(
  cd "$PROJECT_ROOT"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$BINARY_PATH" .
)

echo "Packaging $PACKAGE_NAME..."
tar -C "$DIST_DIR" -czf "$PACKAGE_PATH" mihomo-sub-manager

SSH_OPTS=(-o StrictHostKeyChecking=accept-new)

echo "Uploading package to $SSH_USER@$SERVER_IP..."
sshpass -p "$SSH_PASSWORD" scp "${SSH_OPTS[@]}" "$PACKAGE_PATH" "$SSH_USER@$SERVER_IP:$REMOTE_PACKAGE"

echo "Installing and restarting systemd service..."
sshpass -p "$SSH_PASSWORD" ssh "${SSH_OPTS[@]}" "$SSH_USER@$SERVER_IP" bash -s -- \
  "$REMOTE_PACKAGE" "$INSTALL_DIR" "$DATA_DIR" "$LISTEN_ADDR" "$ADMIN_USER" "$ADMIN_PASSWORD" "$SSH_PASSWORD" <<'REMOTE'
set -euo pipefail

REMOTE_PACKAGE="$1"
INSTALL_DIR="$2"
DATA_DIR="$3"
LISTEN_ADDR="$4"
ADMIN_USER="$5"
ADMIN_PASSWORD="$6"
SSH_PASSWORD="$7"
SERVICE_NAME="mihomo-sub-manager"
ENV_FILE="/etc/mihomo-sub-manager.env"

if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
else
  SUDO="sudo -S"
fi

run_root() {
  if [ -z "$SUDO" ]; then
    sh -c "$*"
  else
    printf '%s\n' "$SSH_PASSWORD" | $SUDO -p '' sh -c "$*"
  fi
}

escape_env() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

run_root "mkdir -p '$INSTALL_DIR' '$DATA_DIR'"
run_root "tar -xzf '$REMOTE_PACKAGE' -C '$INSTALL_DIR'"
run_root "chmod 0755 '$INSTALL_DIR/mihomo-sub-manager'"

cat > /tmp/mihomo-sub-manager.env <<ENV
ADDR="$(escape_env "$LISTEN_ADDR")"
DB_PATH="$(escape_env "$DATA_DIR/mihomo-sub-manager.db")"
ADMIN_USER="$(escape_env "$ADMIN_USER")"
ADMIN_PASSWORD="$(escape_env "$ADMIN_PASSWORD")"
ENV

cat > /tmp/$SERVICE_NAME.service <<UNIT
[Unit]
Description=Mihomo Subscription Manager
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/mihomo-sub-manager
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT

run_root "mv '/tmp/mihomo-sub-manager.env' '$ENV_FILE'"
run_root "chmod 0600 '$ENV_FILE'"
run_root "mv '/tmp/$SERVICE_NAME.service' '/etc/systemd/system/$SERVICE_NAME.service'"
run_root "systemctl daemon-reload"
run_root "systemctl enable '$SERVICE_NAME.service'"
run_root "systemctl restart '$SERVICE_NAME.service'"
run_root "systemctl --no-pager --full status '$SERVICE_NAME.service'"
REMOTE

echo "Deployed successfully."
echo "URL: http://$SERVER_IP$LISTEN_ADDR"
echo "Admin username: $ADMIN_USER"
echo "Admin password: $ADMIN_PASSWORD"
