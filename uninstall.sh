#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-}"
if [[ -z "$TARGET" ]]; then
  echo "Usage: $0 <local|ssh-host>" >&2
  exit 1
fi

SERVICE="cscom"
INSTALL_DIR="/opt/cscom"

# ── uninstall: local ──────────────────────────────────────────────────────────
if [[ "$TARGET" == "local" ]]; then
  echo "→ Stopping service…"
  sudo systemctl stop    "${SERVICE}" 2>/dev/null || true
  sudo systemctl disable "${SERVICE}" 2>/dev/null || true
  sudo rm -f "/etc/systemd/system/${SERVICE}.service"
  sudo systemctl daemon-reload

  echo "→ Removing files…"
  sudo rm -rf "${INSTALL_DIR}"
  sudo rm -f /usr/local/bin/cscom

  echo "✓ CSCom removed."

# ── uninstall: remote ─────────────────────────────────────────────────────────
else
  echo "→ Removing from ${TARGET}…"
  ssh "$TARGET" bash -s <<REMOTE
set -e
sudo systemctl stop    ${SERVICE} 2>/dev/null || true
sudo systemctl disable ${SERVICE} 2>/dev/null || true
sudo rm -f /etc/systemd/system/${SERVICE}.service
sudo systemctl daemon-reload
sudo rm -rf ${INSTALL_DIR}
sudo rm -f /usr/local/bin/cscom
echo "✓ CSCom removed."
REMOTE
fi
