#!/usr/bin/env sh
set -eu

APP_NAME="msm-free"
PREFIX="/usr/local"
DATA_DIR="/opt/msm-free"
SERVICE_NAME="msm-free"
PURGE="0"

usage() {
  cat <<'EOF'
Usage: ./uninstall.sh [options]

Options:
  --prefix PATH        Binary prefix used during install (default: /usr/local)
  --data-dir PATH      Data directory used during install (default: /opt/msm-free)
  --service-name NAME  systemd service name (default: msm-free)
  --purge             Remove the data directory as well
  -h, --help           Show this help
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --prefix)
      PREFIX="${2:?missing value for --prefix}"
      shift 2
      ;;
    --data-dir)
      DATA_DIR="${2:?missing value for --data-dir}"
      shift 2
      ;;
    --service-name)
      SERVICE_NAME="${2:?missing value for --service-name}"
      shift 2
      ;;
    --purge)
      PURGE="1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "uninstall.sh must be run as root" >&2
  exit 1
fi

SERVICE_PATH="/etc/systemd/system/$SERVICE_NAME.service"
BIN_DEST="$PREFIX/bin/$APP_NAME"

if command -v systemctl >/dev/null 2>&1 && [ -f "$SERVICE_PATH" ]; then
  systemctl stop "$SERVICE_NAME" >/dev/null 2>&1 || true
  systemctl disable "$SERVICE_NAME" >/dev/null 2>&1 || true
  rm -f "$SERVICE_PATH"
  systemctl daemon-reload
fi

rm -f "$BIN_DEST"

if [ "$PURGE" = "1" ]; then
  rm -rf "$DATA_DIR"
  echo "removed $APP_NAME and purged $DATA_DIR"
else
  echo "removed $APP_NAME binary and service"
  echo "kept data directory: $DATA_DIR"
fi

