#!/usr/bin/env bash
set -euo pipefail

NAMESPACE=${NAMESPACE:-goose}
KUBECTL_CONTEXT=${KUBECTL_CONTEXT:-}
TIMEOUT=${TIMEOUT:-30}

SERVICES=(
  "goosed-api:8080:18080"
  "goosed-bootd:8080:18081"
  "goosed-blueprints:8080:18082"
  "goosed-orchestrator:8080:18083"
  "goosed-inventory:8080:18084"
  "goosed-artifacts-gw:8080:18085"
)

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[smoke] missing required command: $1" >&2
    exit 1
  fi
}

cleanup() {
  for pid in "${PORT_FORWARD_PIDS[@]-}"; do
    if [[ -n "$pid" ]]; then
      kill "$pid" >/dev/null 2>&1 || true
      wait "$pid" 2>/dev/null || true
    fi
  done
  for log in "${PORT_FORWARD_LOGS[@]-}"; do
    [[ -f "$log" ]] && rm -f "$log"
  done
}

port_forward_and_check() {
  local svc="$1"
  local remote_port="$2"
  local local_port="$3"
  local log_file

  log_file="$(mktemp -t goosed-smoke-${svc}.XXXXXX.log)"
  PORT_FORWARD_LOGS+=("$log_file")

  local args=("-n" "$NAMESPACE" "port-forward" "svc/${svc}" "${local_port}:${remote_port}")
  if [[ -n "$KUBECTL_CONTEXT" ]]; then
    args=("--context" "$KUBECTL_CONTEXT" "${args[@]}")
  fi

  kubectl "${args[@]}" >"$log_file" 2>&1 &
  local pid=$!
  PORT_FORWARD_PIDS+=("$pid")

  local waited=0
  until grep -q "Forwarding from" "$log_file"; do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      echo "[smoke] port-forward for ${svc} exited early" >&2
      cat "$log_file" >&2
      return 1
    fi
    if (( waited >= TIMEOUT )); then
      echo "[smoke] timed out waiting for port-forward on ${svc}" >&2
      cat "$log_file" >&2
      return 1
    fi
    sleep 1
    waited=$((waited + 1))
  done

  if ! curl -fsS "http://127.0.0.1:${local_port}/healthz" >/dev/null; then
    echo "[smoke] healthz check failed for ${svc}" >&2
    cat "$log_file" >&2
    return 1
  fi

  echo "[smoke] ${svc} /healthz OK"
  kill "$pid" >/dev/null 2>&1 || true
  wait "$pid" 2>/dev/null || true
}

main() {
  require_cmd kubectl
  require_cmd curl

  trap cleanup EXIT

  PORT_FORWARD_PIDS=()
  PORT_FORWARD_LOGS=()

  for entry in "${SERVICES[@]}"; do
    IFS=":" read -r svc remote_port local_port <<<"$entry"
    port_forward_and_check "$svc" "$remote_port" "$local_port"
  done

  echo "[smoke] all services responded with HTTP 200"
}

main "$@"
