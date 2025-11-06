#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./copy-iso.sh [ISO_PATH]

Upload a Rocky Linux ISO to the SeaweedFS artifacts bucket.

Environment variables:
  S3_BUCKET        Name of the SeaweedFS bucket (required)
  S3_ENDPOINT      SeaweedFS S3 endpoint URL (required for aws CLI, optional for mc)
  ISO_S3_KEY       Destination key inside the bucket (optional, defaults to artifacts/rocky/9/<iso-filename>)
  MC_ALIAS         Alias configured for the SeaweedFS endpoint when using mc (optional, defaults to "goose")
  MC_API           Override the mc S3 API list, e.g. "S3v2" or "S3v4 S3v2" (optional; defaults to auto-detect)
  S3_ACCESS_KEY    SeaweedFS S3 access key (required when the script needs to bootstrap mc)
  S3_SECRET_KEY    SeaweedFS S3 secret key (required when the script needs to bootstrap mc)

Examples:
  S3_BUCKET=goosed-artifacts S3_ENDPOINT=http://localhost:8333 ./copy-iso.sh ./Rocky-9.4-x86_64-minimal.iso
  ISO_S3_KEY=artifacts/rocky/9/custom.iso ./copy-iso.sh
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -gt 1 ]]; then
  echo "Too many arguments" >&2
  usage
  exit 1
fi

ISO_PATH=${1:-}
if [[ -z "$ISO_PATH" ]]; then
  read -r -p "Path to ISO: " ISO_PATH
fi

if [[ -z "$ISO_PATH" ]]; then
  echo "No ISO path provided" >&2
  exit 1
fi

if [[ ! -f "$ISO_PATH" ]]; then
  echo "ISO not found: $ISO_PATH" >&2
  exit 1
fi

if [[ -z "${S3_BUCKET:-}" ]]; then
  echo "S3_BUCKET environment variable must be set" >&2
  exit 1
fi

cleanup() {
  if [[ -n "${TMP_MC_DIR:-}" && -d "${TMP_MC_DIR:-}" ]]; then
    rm -rf "$TMP_MC_DIR"
  fi
}

trap cleanup EXIT

ISO_PATH=$(realpath "$ISO_PATH")
ISO_FILENAME=$(basename "$ISO_PATH")
ISO_S3_KEY=${ISO_S3_KEY:-"artifacts/rocky/9/$ISO_FILENAME"}
MC_ALIAS=${MC_ALIAS:-goose}
MC_BIN=${MC_BIN:-mc}
MC_API_CANDIDATES=()

upload_with_aws() {
  if [[ -z "${S3_ENDPOINT:-}" ]]; then
    echo "S3_ENDPOINT must be set to use the aws CLI" >&2
    return 1
  fi

  echo "Uploading $ISO_PATH to s3://$S3_BUCKET/$ISO_S3_KEY via aws..."
  aws --endpoint-url "$S3_ENDPOINT" \
    s3 cp "$ISO_PATH" "s3://$S3_BUCKET/$ISO_S3_KEY"
}

mc_alias_exists() {
  local mc_bin="$1"
  "$mc_bin" alias list "$MC_ALIAS" >/dev/null 2>&1
}

mc_supports_api_flag() {
  local mc_bin="$1"
  "$mc_bin" alias set --help 2>&1 | grep -q -- '--api'
}

mc_api_candidates() {
  local mc_bin="$1"

  MC_API_CANDIDATES=()

  if ! mc_supports_api_flag "$mc_bin"; then
    return
  fi

  if [[ -n "${MC_API:-}" ]]; then
    local token
    for token in ${MC_API//,/ }; do
      if [[ -n "$token" ]]; then
        MC_API_CANDIDATES+=("$token")
      fi
    done
  else
    MC_API_CANDIDATES=(S3v2 S3v4)
  fi
}

configure_mc_alias() {
  local mc_bin="$1"
  local api="$2"

  if [[ -z "${S3_ENDPOINT:-}" ]]; then
    echo "S3_ENDPOINT must be set to configure the mc alias" >&2
    return 1
  fi

  if [[ -z "${S3_ACCESS_KEY:-}" || -z "${S3_SECRET_KEY:-}" ]]; then
    echo "S3_ACCESS_KEY and S3_SECRET_KEY must be set to configure the mc alias" >&2
    return 1
  fi

  if [[ -n "$api" ]]; then
    echo "Configuring mc alias '$MC_ALIAS' for $S3_ENDPOINT (api=$api)..."
    "$mc_bin" alias set --api "$api" "$MC_ALIAS" "$S3_ENDPOINT" "$S3_ACCESS_KEY" "$S3_SECRET_KEY"
  else
    echo "Configuring mc alias '$MC_ALIAS' for $S3_ENDPOINT..."
    "$mc_bin" alias set "$MC_ALIAS" "$S3_ENDPOINT" "$S3_ACCESS_KEY" "$S3_SECRET_KEY"
  fi
}

download_mc() {
  local os
  local arch

  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)

  case "$os" in
    darwin)
      case "$arch" in
        arm64|aarch64) arch="arm64" ;;
        x86_64|amd64) arch="amd64" ;;
        *) echo "Unsupported architecture for macOS: $arch" >&2; return 1 ;;
      esac
      os="darwin"
      ;;
    linux)
      case "$arch" in
        arm64|aarch64) arch="arm64" ;;
        x86_64|amd64) arch="amd64" ;;
        *) echo "Unsupported architecture for Linux: $arch" >&2; return 1 ;;
      esac
      os="linux"
      ;;
    *)
      echo "Automatic mc download is not supported on $os" >&2
      return 1
      ;;
  esac

  local url="https://dl.min.io/client/mc/release/${os}-${arch}/mc"

  if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to download mc automatically" >&2
    return 1
  fi

  TMP_MC_DIR=$(mktemp -d)
  local mc_path="$TMP_MC_DIR/mc"

  echo "Downloading mc from $url..."
  if ! curl -fsSL "$url" -o "$mc_path"; then
    echo "Failed to download mc from $url" >&2
    return 1
  fi

  chmod +x "$mc_path"
  export MC_CONFIG_DIR="${MC_CONFIG_DIR:-$TMP_MC_DIR/.mc}"
  MC_BIN="$mc_path"
}

ensure_mc_available() {
  if command -v mc >/dev/null 2>&1; then
    MC_BIN=$(command -v mc)
    return 0
  fi

  download_mc
}

try_mc_copy() {
  local mc_bin="$1"
  local target="$2"
  local descriptor="$3"

  if [[ -n "$descriptor" ]]; then
    descriptor=" $descriptor"
  fi

  echo "Uploading $ISO_PATH to $target via $mc_bin$descriptor..."
  "$mc_bin" cp "$ISO_PATH" "$target"
}

upload_with_mc() {
  local mc_bin="$1"
  local target="$MC_ALIAS/$S3_BUCKET/$ISO_S3_KEY"

  if mc_alias_exists "$mc_bin"; then
    if try_mc_copy "$mc_bin" "$target" "(existing alias)"; then
      return 0
    fi

    echo "mc upload failed using existing alias '$MC_ALIAS'. Retrying with alternate API configurations..." >&2
  fi

  if [[ -z "${S3_ENDPOINT:-}" || -z "${S3_ACCESS_KEY:-}" || -z "${S3_SECRET_KEY:-}" ]]; then
    echo "Unable to retry mc upload automatically. S3_ENDPOINT, S3_ACCESS_KEY, and S3_SECRET_KEY must be set." >&2
    return 1
  fi

  mc_api_candidates "$mc_bin"
  local apis=("${MC_API_CANDIDATES[@]}")

  local attempts=()
  if (( ${#apis[@]} > 0 )); then
    attempts=("${apis[@]}")
    local has_default=0
    local candidate
    for candidate in "${apis[@]}"; do
      if [[ -z "$candidate" ]]; then
        has_default=1
        break
      fi
    done
    if (( ! has_default )); then
      attempts+=("")
    fi
  else
    attempts=("")
  fi

  local api
  for api in "${attempts[@]}"; do
    if ! configure_mc_alias "$mc_bin" "$api"; then
      return 1
    fi

    local descriptor=""
    if [[ -n "$api" ]]; then
      descriptor="(api=$api)"
    fi

    if try_mc_copy "$mc_bin" "$target" "$descriptor"; then
      return 0
    fi

    if [[ -n "$api" ]]; then
      echo "mc upload failed using API '$api'." >&2
    else
      echo "mc upload failed using the default API settings." >&2
    fi
  done

  echo "Unable to upload $ISO_PATH via mc after exhausting all API options." >&2
  return 1
}

if command -v aws >/dev/null 2>&1; then
  upload_with_aws && exit 0
fi

if ensure_mc_available; then
  upload_with_mc "$MC_BIN"
  exit 0
fi

echo "Unable to locate or bootstrap mc. Install aws CLI or mc to upload the ISO." >&2
exit 1
