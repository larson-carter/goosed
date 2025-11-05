#!/usr/bin/env bash
set -euo pipefail

CONFIG_PATH="/etc/goosed/agent.conf"
UNIT_PATH="/etc/systemd/system/goosed-agent.service"
BINARY_PATH="/usr/local/bin/goosed-agent-rhel"
STATE_DIR="/var/lib/goosed"

json_escape() {
    local value="$1"
    value="${value//\\/\\\\}"
    value="${value//\"/\\\"}"
    value="${value//$'\n'/\\n}"
    value="${value//$'\r'/\\r}"
    value="${value//$'\t'/\\t}"
    printf '%s' "$value"
}

get_cmdline_value() {
    local key="$1"
    local arg
    for arg in $(cat /proc/cmdline 2>/dev/null); do
        case "$arg" in
            "$key"=*)
                echo "${arg#*=}"
                return 0
                ;;
        esac
    done
    return 1
}

get_file_value() {
    local path="$1"
    if [[ -f "$path" ]]; then
        tr -d '\n' <"$path"
        return 0
    fi
    return 1
}

API=${API:-}
TOKEN=${TOKEN:-}
MACHINE_ID=${MACHINE_ID:-}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --api)
            API="$2"
            shift 2
            ;;
        --token)
            TOKEN="$2"
            shift 2
            ;;
        --machine|--machine-id)
            MACHINE_ID="$2"
            shift 2
            ;;
        --binary)
            BINARY_PATH="$2"
            shift 2
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

API="${API:-${API_URL:-${GOOSED_API:-}}}"
TOKEN="${TOKEN:-${API_TOKEN:-${GOOSED_TOKEN:-}}}"
MACHINE_ID="${MACHINE_ID:-${GOOSED_MACHINE_ID:-}}"

if [[ -z "$API" ]]; then
    for key in goosed.api goosed_api api API_URL GOOSED_API; do
        if value=$(get_cmdline_value "$key" 2>/dev/null); then
            API="$value"
            break
        fi
    done
fi

if [[ -z "$TOKEN" ]]; then
    for key in goosed.token goosed_token token TOKEN GOOSED_TOKEN; do
        if value=$(get_cmdline_value "$key" 2>/dev/null); then
            TOKEN="$value"
            break
        fi
    done
fi

if [[ -z "$MACHINE_ID" ]]; then
    if value=$(get_cmdline_value goosed.machine_id 2>/dev/null); then
        MACHINE_ID="$value"
    fi
fi

if [[ -z "$MACHINE_ID" ]]; then
    if value=$(get_file_value /etc/machine-id); then
        MACHINE_ID="$value"
    fi
fi

if [[ -z "$API" || -z "$TOKEN" || -z "$MACHINE_ID" ]]; then
    echo "API URL, token, and machine ID are required" >&2
    exit 1
fi

install -d -m 0755 "$(dirname "$CONFIG_PATH")"
install -d -m 0755 "$STATE_DIR"

API_ESCAPED=$(json_escape "$API")
TOKEN_ESCAPED=$(json_escape "$TOKEN")
MACHINE_ESCAPED=$(json_escape "$MACHINE_ID")

cat >"$CONFIG_PATH" <<EOF_CONF
{
  "api": "${API_ESCAPED}",
  "token": "${TOKEN_ESCAPED}",
  "machine_id": "${MACHINE_ESCAPED}"
}
EOF_CONF
chmod 0640 "$CONFIG_PATH"

cat >"$UNIT_PATH" <<EOF_UNIT
[Unit]
Description=Goosed RHEL Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$BINARY_PATH
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF_UNIT

chmod 0644 "$UNIT_PATH"

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable --now goosed-agent.service
else
    echo "systemctl not available; skipping enable/start" >&2
fi
