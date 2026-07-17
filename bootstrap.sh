#!/usr/bin/env bash
#
# KP Restic Wrap - bootstrap installer
# Downloads the latest release binary and installs it to /usr/local/bin
#
# Copyright (c) 2026 Kevin Pirnie
# Licensed under the MIT License. See LICENSE for details.

set -euo pipefail

REPO="kpirnie/kp-restic-wrap"
INSTALL_PATH="/usr/local/bin/kp"

# must be root to write the binary and later the config
if [ "$(id -u)" -ne 0 ]; then
    echo "Run as root: sudo $0" >&2
    exit 1
fi

# map the machine arch to the release asset
case "$(uname -m)" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

ASSET="kp-linux-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

# fetch and install
echo "Downloading ${ASSET}..."
TMP="$(mktemp)"
trap 'rm -f "${TMP}"' EXIT
curl -fsSL -o "${TMP}" "${URL}"
install -m 0755 "${TMP}" "${INSTALL_PATH}"

echo "Installed $(${INSTALL_PATH} help 2>/dev/null | head -1 || echo kp) to ${INSTALL_PATH}"

# verify restic is present, warn if not
if ! command -v restic >/dev/null 2>&1; then
    echo ""
    echo "WARNING: restic is not installed or not in PATH."
    echo "  Debian/Ubuntu: apt install restic fuse3"
    echo "  RHEL/Fedora:   dnf install restic fuse3"
fi

# next steps
cat << 'EOF'

Next steps:
  1. sudo kp configure     - create your configuration and initialize repositories
  2. sudo kp backup        - run your first backup
  3. Automate it           - see the readme for systemd timer and cron examples

EOF