#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-}"
if [[ -z "$TARGET" ]]; then
  echo "Usage: $0 <local|ssh-host>" >&2
  exit 1
fi

[[ -f go.mod ]] || { echo "Error: run from project root" >&2; exit 1; }

INSTALL_DIR="/opt/cscom"
SERVICE="cscom"

# ── detect or prompt for key ──────────────────────────────────────────────────
CSCOM_KEY=""
if [[ "$TARGET" == "local" ]]; then
  if [[ -f "${INSTALL_DIR}/.env" ]]; then
    CSCOM_KEY="$(sudo grep '^CSCOM_KEY=' "${INSTALL_DIR}/.env" | cut -d= -f2- || true)"
    echo "→ Using existing CSCOM_KEY from ${INSTALL_DIR}/.env"
  else
    read -rsp "CSCOM_KEY (leave empty to disable auth): " CSCOM_KEY
    echo
  fi
else
  if ssh "$TARGET" "test -f ${INSTALL_DIR}/.env" 2>/dev/null; then
    LINE="$(ssh "$TARGET" "grep '^CSCOM_KEY=' ${INSTALL_DIR}/.env || true")"
    CSCOM_KEY="${LINE#CSCOM_KEY=}"
    echo "→ Using existing CSCOM_KEY from remote ${INSTALL_DIR}/.env"
  else
    read -rsp "CSCOM_KEY (leave empty to disable auth): " CSCOM_KEY
    echo
  fi
fi

# ── detect target arch ────────────────────────────────────────────────────────
echo "→ Detecting architecture…"
if [[ "$TARGET" == "local" ]]; then
  RAW_ARCH="$(uname -m)"
else
  RAW_ARCH="$(ssh "$TARGET" uname -m)"
fi
case "$RAW_ARCH" in
  x86_64)        GOARCH="amd64" ;;
  aarch64|arm64) GOARCH="arm64" ;;
  *)             GOARCH="amd64" ;;
esac
echo "   linux/$GOARCH"

# ── build ─────────────────────────────────────────────────────────────────────
echo "→ Building binaries…"
mkdir -p bin
GOOS=linux GOARCH=$GOARCH go build -ldflags="-s -w" -o bin/cscom        ./cmd/cscom/
GOOS=linux GOARCH=$GOARCH go build -ldflags="-s -w" -o bin/cscom-serve  ./cmd/cscom-serve/

# ── generate config files ─────────────────────────────────────────────────────
TMP_ENV=$(mktemp)
TMP_SVC=$(mktemp)
trap 'rm -f "$TMP_ENV" "$TMP_SVC"' EXIT

printf 'PORT=4040\nCSCOM_KEY=%s\nCSCOM_WEB=%s/web\n' "$CSCOM_KEY" "$INSTALL_DIR" > "$TMP_ENV"

cat > "$TMP_SVC" <<SVCEOF
[Unit]
Description=CSCom - Control System Commander
After=network.target

[Service]
ExecStart=${INSTALL_DIR}/cscom-serve
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SVCEOF

# ── deploy: local ─────────────────────────────────────────────────────────────
if [[ "$TARGET" == "local" ]]; then
  echo "→ Installing to ${INSTALL_DIR}…"
  sudo mkdir -p "${INSTALL_DIR}/web"
  sudo install -m 755 bin/cscom bin/cscom-serve "${INSTALL_DIR}/"
  sudo cp web/index.html web/login.html "${INSTALL_DIR}/web/"
  sudo install -m 600 "$TMP_ENV" "${INSTALL_DIR}/.env"
  sudo cp "$TMP_SVC" "/etc/systemd/system/${SERVICE}.service"

  echo "→ Starting service…"
  sudo systemctl daemon-reload
  sudo systemctl enable  "${SERVICE}"
  sudo systemctl restart "${SERVICE}"
  sudo systemctl --no-pager status "${SERVICE}"

  sudo ln -sf "${INSTALL_DIR}/cscom" /usr/local/bin/cscom

  echo ""
  echo "✓ Done!"
  echo "  TUI        → cscom"
  echo "  Dashboard  → http://localhost:4040"
  echo "  API        → http://localhost:4040/api/metrics"
  echo "  Logs       → journalctl -fu ${SERVICE}"

# ── deploy: remote ────────────────────────────────────────────────────────────
else
  echo "→ Deploying to ${TARGET}…"
  # scp to /tmp first (always writable), then sudo-install to final locations
  scp -q bin/cscom bin/cscom-serve  "${TARGET}:/tmp/"
  scp -q web/index.html             "${TARGET}:/tmp/cscom-index.html"
  scp -q web/login.html             "${TARGET}:/tmp/cscom-login.html"
  scp -q "$TMP_ENV"                 "${TARGET}:/tmp/cscom.env"
  scp -q "$TMP_SVC"                 "${TARGET}:/tmp/${SERVICE}.service"
  ssh "$TARGET" "chmod 600 /tmp/cscom.env"

  echo "→ Installing and starting service…"
  ssh "$TARGET" bash -s <<REMOTE
set -e
sudo mkdir -p ${INSTALL_DIR}/web
sudo install -m 755 /tmp/cscom /tmp/cscom-serve ${INSTALL_DIR}/
sudo install -m 644 /tmp/cscom-index.html ${INSTALL_DIR}/web/index.html
sudo install -m 644 /tmp/cscom-login.html ${INSTALL_DIR}/web/login.html
sudo install -m 600 /tmp/cscom.env ${INSTALL_DIR}/.env
sudo mv /tmp/${SERVICE}.service /etc/systemd/system/${SERVICE}.service
sudo ln -sf ${INSTALL_DIR}/cscom /usr/local/bin/cscom
rm -f /tmp/cscom /tmp/cscom-serve /tmp/cscom-index.html /tmp/cscom-login.html /tmp/cscom.env
sudo systemctl daemon-reload
sudo systemctl enable  ${SERVICE}
sudo systemctl restart ${SERVICE}
sudo systemctl --no-pager status ${SERVICE}
REMOTE

  HOST_IP="$(ssh "$TARGET" "hostname -I 2>/dev/null | awk '{print \$1}' || echo $TARGET")"
  echo ""
  echo "✓ Done!"
  echo "  Dashboard  → http://${HOST_IP}:4040"
  echo "  API        → http://${HOST_IP}:4040/api/metrics"
  echo "  Logs       → ssh ${TARGET} journalctl -fu ${SERVICE}"
fi
