# Mihomo Subscription Manager

A small self-hosted web app that turns individual proxy links into tokenized Mihomo subscription YAML.

It is intended for personal deployments where you have one or more raw proxy links, but your Mihomo client expects a subscription URL with proxy groups and rules.

## Features

- Pure local conversion, no external subscription conversion service.
- Mihomo-focused YAML output.
- Admin web UI with login.
- Public tokenized subscription URLs without login.
- Multiple named proxy nodes per subscription.
- Built-in default rules plus selectable custom rule groups.
- JSON export and import for server migration.
- SQLite storage through GORM.
- Single Go binary with embedded HTML.

## Supported Link Schemes

The converter supports common fields for:

- `vless`
- `vmess`
- `trojan`
- `ss`
- `ssr`
- `hysteria`
- `hysteria2`
- `hy2`

Protocol ecosystems have many client-specific URL variants. If a specific provider format is not parsed correctly, add a failing test in `converter_test.go` before changing `converter.go`.

## Build And Run

```bash
go test -count=1 ./...
go build ./...
```

Run locally:

```bash
ADDR=":49321" \
DB_PATH="mihomo-sub-manager.db" \
ADMIN_USER="admin" \
ADMIN_PASSWORD="mSm-4Qb9-vY2-Zt7-Kp5" \
go run .
```

Open:

```text
http://127.0.0.1:49321
```

## Configuration

The application reads these environment variables:

| Name | Default | Description |
| --- | --- | --- |
| `ADDR` | `:49321` | HTTP listen address |
| `DB_PATH` | `mihomo-sub-manager.db` | SQLite database path |
| `ADMIN_USER` | `admin` | Admin login username |
| `ADMIN_PASSWORD` | `mSm-4Qb9-vY2-Zt7-Kp5` | Admin login password |

Change `ADMIN_PASSWORD` for any real deployment.

## Admin UI

Subscriptions contain named proxy nodes. Each node has:

- Node name: used as the Mihomo proxy name.
- Proxy URL: the raw `vless://`, `vmess://`, `trojan://`, `ss://`, `ssr://`, `hysteria://`, `hysteria2://`, or `hy2://` link.

If the node name is blank, the converter falls back to the URL fragment name, then to `proxy-1`, `proxy-2`, and so on.

## Deploy To Ubuntu

### From Windows PowerShell

The PowerShell script requires `plink.exe` and `pscp.exe` from PuTTY. It also supports putting them in `.tools/putty/`.

```powershell
.\scripts\deploy-ubuntu.ps1 `
  -ServerIP "203.0.113.10" `
  -SSHUser "root" `
  -SSHPassword "your-ssh-password" `
  -ListenAddr ":49321" `
  -AdminUser "admin" `
  -AdminPassword "replace-this-password"
```

Optional:

```powershell
-SSHHostKey "ssh-ed25519 255 SHA256:..."
```

The PowerShell script also accepts environment variables with the same uppercase names, such as `SERVER_IP`, `SSH_PASSWORD`, `ADMIN_USER`, and `ADMIN_PASSWORD`.

### From Linux Or macOS

The shell script requires `sshpass`, `ssh`, `scp`, `go`, and `tar`.

```bash
./scripts/deploy-ubuntu.sh \
  --server-ip "203.0.113.10" \
  --ssh-user "root" \
  --ssh-password "your-ssh-password" \
  --listen-addr ":49321" \
  --admin-user "admin" \
  --admin-password "replace-this-password"
```

The shell script also accepts the same values from environment variables, for example:

```bash
SERVER_IP="203.0.113.10" \
SSH_PASSWORD="your-ssh-password" \
ADMIN_PASSWORD="replace-this-password" \
./scripts/deploy-ubuntu.sh
```

The scripts install:

- Binary: `/opt/mihomo-sub-manager/mihomo-sub-manager`
- Database: `/var/lib/mihomo-sub-manager/mihomo-sub-manager.db`
- Environment file: `/etc/mihomo-sub-manager.env`
- systemd unit: `mihomo-sub-manager.service`

Check the service:

```bash
systemctl status mihomo-sub-manager.service
journalctl -u mihomo-sub-manager.service -f
```

## Data Migration

Use the admin UI:

- Export data: downloads all subscriptions and rule groups as JSON.
- Import data: replaces current subscriptions and rule groups with the uploaded JSON.

## Development Notes

- Keep `/sub/{token}.yaml` public and unauthenticated.
- Keep admin APIs protected by the login session.
- Prefer tests for converter changes because provider URL formats vary.
- The SQLite driver is pure Go, so Linux builds can use `CGO_ENABLED=0`.
