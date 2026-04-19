# cloud-init-portal

Single-binary Go web app for generating and serving cloud-init files for bare-metal provisioning.

## What it does

- Runs an HTTP UI on `http://0.0.0.0:8080`
- Loads cloud-init templates from `./templates/*.yaml` at startup
- Lets field tech choose template + box type and enter host/network details
- Renders user-data via Go `text/template`
- Serves:
  - `GET /` UI
  - `POST /provision` generate current config
  - `POST /consume` manual consume/clear
  - `POST /force-replace` discard active config and allow immediate replacement
  - `GET /status` current provisioning status as JSON
  - `GET /user-data` rendered cloud-init user-data
  - `GET /meta-data` rendered cloud-init meta-data
- Keeps only one active config at a time (in-memory by default; optional file-backed state via `STATE_FILE`)
- Logs events to `./provision.log`

Operational note: use log rotation for `provision.log` in long-running environments.

## Run

```bash
go run .
```

Environment options:

- `PUBLIC_BASE_URL` (example: `https://portal.example.com`) — overrides generated user-data/meta-data URLs in UI output.
- `TRUST_PROXY_HEADERS=true` — trust `X-Forwarded-Proto` and `X-Forwarded-Host` when building URLs (use only behind trusted proxy).
- `STATE_FILE=/var/lib/cloud-init-portal/state.json` — enable optional file-backed persistence for active/consumed state.
- `STATUS_RATE_LIMIT_PER_SEC=6` — per-client fixed-window rate limit for `GET /status`.
- `WRITE_RATE_LIMIT_PER_SEC=3` — per-client fixed-window rate limit for write endpoints (`/provision`, `/consume`, `/force-replace`).

Operational behavior:
- responses include `X-Request-ID` for traceability.
- lightweight fixed-window rate limiting is applied per client IP.

## Build

```bash
go build -o cloud-init-portal .
# or
make build
```

Then open: `http://127.0.0.1:8080`

The HTTP server runs with conservative timeouts and graceful shutdown support on `SIGINT`/`SIGTERM`.

## Versioning (SemVer)

This repo uses `VERSION` with strict SemVer core format:

```text
MAJOR.MINOR.PATCH
```

Current version is stored in the root `VERSION` file.

### Auto bump via GitHub label

Workflow: `.github/workflows/version-bump.yml`

On merged PRs, the workflow inspects labels and bumps `VERSION` on the default branch:

- `major` -> `X+1.0.0`
- `minor` -> `X.Y+1.0`
- `patch` -> `X.Y.Z+1`

Rules:
- Exactly one of `major`, `minor`, `patch` should be present.
- If none are present, bump is skipped.
- If multiple are present, workflow fails.

## Add templates

Put `.yaml` files in `./templates/`.

Template placeholders available:

- `{{.Hostname}}`
- `{{.BootstrapInterface}}`
- `{{.ProductionInterface}}`
- `{{.ProductionAddress}}` (example `192.168.50.10/24`)
- `{{.Gateway}}`
- `{{range .DNS}}...{{end}}`

The app parses templates at startup with `missingkey=error`.

Input validation:
- `hostname` must be RFC1123-ish label (letters/numbers/dash, max 63)
- `static_ip` and optional `gateway` must be valid IP addresses
- each DNS entry must be a valid IP address
- `cidr` must be in range `1-32`

## Example rendered user-data (from templates/example.yaml)

```yaml
#cloud-config
hostname: edge-001
manage_etc_hosts: true

write_files:
  - path: /etc/systemd/system/systemd-networkd-wait-online.service.d/override.conf
    permissions: '0644'
    owner: root:root
    content: |
      [Service]
      ExecStart=
      ExecStart=/usr/lib/systemd/systemd-networkd-wait-online --any --timeout=20

runcmd:
  - [systemctl, daemon-reload]
  - [systemctl, restart, systemd-networkd-wait-online.service]

network:
  version: 2
  renderer: networkd
  ethernets:
    enp1s0:
      dhcp4: true
    enp2s0:
      dhcp4: false
      addresses:
        - 192.168.50.10/24
      routes:
        - to: default
          via: 192.168.50.1
      nameservers:
        addresses:
          - 1.1.1.1
          - 8.8.8.8
```
