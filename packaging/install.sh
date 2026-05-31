#!/usr/bin/env sh
set -eu

APP_NAME="msm-free"
PREFIX="/usr/local"
DATA_DIR="/opt/msm-free"
HOST="0.0.0.0"
PORT="7777"
SERVICE_NAME="msm-free"
START_SERVICE="1"
ALIAS_NAME="msm"
INSTALL_ALIAS="1"

usage() {
  cat <<'EOF'
Usage: ./install.sh [options]

Options:
  --prefix PATH        Install binary under PATH/bin (default: /usr/local)
  --data-dir PATH      msm-free data directory (default: /opt/msm-free)
  --host HOST          HTTP listen host (default: 0.0.0.0)
  --port PORT          HTTP listen port (default: 7777)
  --service-name NAME  systemd service name (default: msm-free)
  --alias-name NAME    CLI alias to register under PATH/bin (default: msm)
  --no-alias           Do not register the msm compatibility alias
  --no-start           Install and enable the service, but do not start it
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
    --host)
      HOST="${2:?missing value for --host}"
      shift 2
      ;;
    --port)
      PORT="${2:?missing value for --port}"
      shift 2
      ;;
    --service-name)
      SERVICE_NAME="${2:?missing value for --service-name}"
      shift 2
      ;;
    --alias-name)
      ALIAS_NAME="${2:?missing value for --alias-name}"
      shift 2
      ;;
    --no-alias)
      INSTALL_ALIAS="0"
      shift
      ;;
    --no-start)
      START_SERVICE="0"
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
  echo "install.sh must be run as root" >&2
  exit 1
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
BIN_SRC="$SCRIPT_DIR/$APP_NAME"
if [ ! -f "$BIN_SRC" ]; then
  echo "missing bundled binary: $BIN_SRC" >&2
  exit 1
fi

BIN_DIR="$PREFIX/bin"
BIN_DEST="$BIN_DIR/$APP_NAME"
SERVICE_PATH="/etc/systemd/system/$SERVICE_NAME.service"

mkdir -p "$BIN_DIR" "$DATA_DIR"
install -m 0755 "$BIN_SRC" "$BIN_DEST"
"$BIN_DEST" init --config "$DATA_DIR"

if [ "$INSTALL_ALIAS" = "1" ] && [ -n "$ALIAS_NAME" ]; then
  ALIAS_DEST="$BIN_DIR/$ALIAS_NAME"
  if [ ! -e "$ALIAS_DEST" ] || [ -L "$ALIAS_DEST" ]; then
    ln -sfn "$BIN_DEST" "$ALIAS_DEST"
  else
    echo "skip alias: $ALIAS_DEST already exists and is not a symlink" >&2
  fi
fi

if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
  cat > "$SERVICE_PATH" <<EOF
[Unit]
Description=msm-free service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$DATA_DIR
Environment=MSM_FREE_DATA_DIR=$DATA_DIR
ExecStart=$BIN_DEST serve --config $DATA_DIR --host $HOST --port $PORT
Restart=on-failure
RestartSec=2
TimeoutStopSec=30
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
  chmod 0644 "$SERVICE_PATH"
  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME" >/dev/null
  if [ "$START_SERVICE" = "1" ]; then
    systemctl restart "$SERVICE_NAME"
  fi
  echo "installed $APP_NAME to $BIN_DEST"
  [ "$INSTALL_ALIAS" = "1" ] && [ -n "$ALIAS_NAME" ] && echo "cli alias: $BIN_DIR/$ALIAS_NAME"
  echo "data directory: $DATA_DIR"
  echo "service: $SERVICE_NAME"
  echo "web UI: http://$(hostname -I 2>/dev/null | awk '{print $1}'):$PORT"
else
  echo "installed $APP_NAME to $BIN_DEST"
  [ "$INSTALL_ALIAS" = "1" ] && [ -n "$ALIAS_NAME" ] && echo "cli alias: $BIN_DIR/$ALIAS_NAME"
  echo "systemd was not detected; start manually:"
  echo "  $BIN_DEST serve --config $DATA_DIR --host $HOST --port $PORT"
fi
