# <img src="https://c.kcp.im/logos/kevinpirnie-favicon-initials.svg" alt="kp ~ Restic Backup Wrapper" width="64" valign="middle"> KP Restic Wrap

[![Latest Release](https://img.shields.io/github/v/release/kpirnie/kp-restic-wrap?style=for-the-badge&labelColor=000)](https://github.com/kpirnie/kp-restic-wrap/releases/latest)
[![Last Commit](https://img.shields.io/github/last-commit/kpirnie/kp-restic-wrap?style=for-the-badge&labelColor=000)](https://github.com/kpirnie/kp-restic-wrap/commits/main)
[![License: MIT](https://img.shields.io/badge/License-MIT-orange.svg?style=for-the-badge&logo=opensourceinitiative&logoColor=white&labelColor=000)](LICENSE)

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white&style=for-the-badge&labelColor=000)](https://go.dev/)
[![Alpine](https://img.shields.io/badge/Base-Alpine%20Linux-0D597F?logo=alpinelinux&logoColor=white&style=for-the-badge&labelColor=000)](https://www.alpinelinux.org/)
[![Kevin Pirnie](https://img.shields.io/badge/-KevinPirnie.com-000d2d?style=for-the-badge&labelColor=000&logoColor=white&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjgiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLWxpbmVqb2luPSJyb3VuZCI+CiAgPGNpcmNsZSBjeD0iMTIiIGN5PSIxMiIgcj0iMTAiLz4KICA8ZWxsaXBzZSBjeD0iMTIiIGN5PSIxMiIgcng9IjQuNSIgcnk9IjEwIi8+CiAgPGxpbmUgeDE9IjIiIHkxPSIxMiIgeDI9IjIyIiB5Mj0iMTIiLz4KICA8bGluZSB4MT0iNC41IiB5MT0iNi41IiB4Mj0iMTkuNSIgeTI9IjYuNSIvPgogIDxsaW5lIHgxPSI0LjUiIHkxPSIxNy41IiB4Mj0iMTkuNSIgeTI9IjE3LjUiLz4KPC9zdmc+Cg==)](https://kevinpirnie.com/)
[![Support](https://img.shields.io/badge/Support-Available-28a745?logo=handshake&logoColor=white&style=for-the-badge&labelColor=000)](https://github.com/kpirnie/kp-restic-wrap/issues)

---

[Requirements](#requirements) | [Installation](#installation) | [Configuration](#configuration) | [Commands](#commands) | [Automation](#automation)

A configuration-driven CLI wrapper around [restic](https://restic.net/) for backup, restore, and mount operations against S3-compatible storage. Native Linux binary, no container required.

---

## Requirements

[Requirements](#requirements) | [Installation](#installation) | [Configuration](#configuration) | [Commands](#commands) | [Automation](#automation)

- Linux
- `restic` in PATH
- `fusermount` for the mount command (`fuse3` package on Debian/Ubuntu)
- An S3-compatible bucket

[Back To Top](#top)

---

## Installation

[Requirements](#requirements) | [Installation](#installation) | [Configuration](#configuration) | [Commands](#commands) | [Automation](#automation)

### Bootstrap (recommended)

Downloads the latest release binary for your architecture (amd64/arm64) and installs it to `/usr/local/bin/kp`:

    ```bash
    curl -fsSL https://raw.githubusercontent.com/kpirnie/kp-restic-wrap/main/bootstrap.sh | sudo bash
    ```

### From source

    ```bash
    git clone https://github.com/kpirnie/kp-restic-wrap.git
    cd kp-restic-wrap
    go build -o kp .
    sudo mv kp /usr/local/bin/
    ```

### Releases

Prebuilt binaries and checksums are published on every version tag: <https://github.com/kpirnie/kp-restic-wrap/releases>

[Back To Top](#top)

---

## Configuration

[Requirements](#requirements) | [Installation](#installation) | [Configuration](#configuration) | [Commands](#commands) | [Automation](#automation)

All configuration lives in `/etc/kp/config.yaml` (override with `--config`). Create and manage it interactively:

```bash
sudo kp configure
```

The walk covers every setting, offers to initialize any repository that doesn't exist yet, and rotates repository keys if you change the encryption password. The file is written with `0600` permissions.

### Settings

| Setting | Description | Default |
| ------- | ----------- | ------- |
| `s3.endpoint` | S3 endpoint | `s3.amazonaws.com` |
| `s3.key` | S3 API key | |
| `s3.secret` | S3 API secret | |
| `s3.bucket` | S3 bucket name | |
| `s3.region` | S3 region | `us-east-1` |
| `temp_path` | Disk-backed staging path for restic | `/var/tmp/kp` |
| `encryption.password` | Repository encryption password (required, never auto-generated) | |
| `backups[].name` | Repository name for this backup set | hostname |
| `backups[].start_path` | Path to back up | `/home` |
| `backups[].retention_days` | Days of snapshots to keep | `30` |
| `backups[].excludes` | Restic exclude patterns | |

### Example

```yaml
s3:
  endpoint: s3.amazonaws.com
  key: AKIAIOSFODNN7EXAMPLE
  secret: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  bucket: my-backup-bucket
  region: us-east-1
temp_path: /var/tmp/kp
encryption:
  password: my-super-secret-encryption-password
backups:
  - name: webserver-01
    start_path: /home
    retention_days: 30
    excludes:
      - "**/node_modules"
      - "**/.cache"
  - name: etc-configs
    start_path: /etc
    retention_days: 90
    excludes: []
```

Each backup name gets its own restic repository under the bucket. `temp_path` exists because tmpfs-backed `/tmp` can corrupt restores larger than available RAM — point it at real disk.

[Back To Top](#top)

---

## Commands

[Requirements](#requirements) | [Installation](#installation) | [Configuration](#configuration) | [Commands](#commands) | [Automation](#automation)

### configure

Interactive create/edit of the configuration, repository initialization, and key rotation on password change.

```bash
sudo kp configure
```

### backup

Backs up every configured entry, then applies retention (`--keep-within`) and prunes, per entry. Entries run concurrently (capped at half the CPU count) with output grouped per entry. Exits non-zero only if a backup fully failed; unreadable-file warnings (restic exit 3) are logged but the snapshot still lands.

```bash
sudo kp backup
```

### restore

Fully interactive: pick the backup, pick the snapshot by date/time, enter the target path, optionally limit to specific paths within the snapshot.

```bash
sudo kp restore
```

The full original path structure is recreated under the target (e.g. restoring to `/tmp/restore` produces `/tmp/restore/home/...`).

### mount

Fully interactive: pick the backup, enter the mountpoint. Holds the FUSE mount in the foreground; Ctrl-C unmounts cleanly with a `fusermount -u` fallback.

```bash
sudo kp mount
```

Browse snapshots at `<mountpoint>/snapshots/`.

[Back To Top](#top)

---

## Automation

[Requirements](#requirements) | [Installation](#installation) | [Configuration](#configuration) | [Commands](#commands) | [Automation](#automation)

### systemd

**`/etc/systemd/system/kp-backup.service`:**

```ini
[Unit]
Description=KP Restic Backup
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/kp backup
```

**`/etc/systemd/system/kp-backup.timer`:**

```ini
[Unit]
Description=Run KP Backup Daily at 2 AM

[Timer]
OnCalendar=*-*-* 02:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kp-backup.timer
```

### cron

```
0 2 * * * /usr/local/bin/kp backup >> /var/log/kp-backup.log 2>&1
```

[Back To Top](#top)

---

## License

MIT License — see [LICENSE](https://github.com/kpirnie/kp-restic-wrap/blob/main/LICENSE) for details.

## Support

- GitHub: <https://github.com/kpirnie/kp-restic-wrap>
- Issues: <https://github.com/kpirnie/kp-restic-wrap/issues>

[Back To Top](#top)