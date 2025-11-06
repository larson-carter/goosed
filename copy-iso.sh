#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage: ./copy-iso.sh [ISO_PATH]

Upload a Rocky Linux ISO to the SeaweedFS artifacts bucket.

Environment variables:
  S3_BUCKET        Name of the SeaweedFS bucket (required)
  S3_ENDPOINT      SeaweedFS S3 endpoint URL (e.g. http://localhost:8333) (required)
  S3_ACCESS_KEY    SeaweedFS S3 access key (required)
  S3_SECRET_KEY    SeaweedFS S3 secret key (required)
  S3_REGION        S3 region for aws CLI (optional; default: us-east-1)
  S3_DISABLE_TLS   Set to "true" if your endpoint is http (optional; aws ignores if endpoint is http)
  ISO_S3_KEY       Destination key inside the bucket (optional; defaults to artifacts/rocky/9/<iso-filename>)
  MC_ALIAS         mc alias name (optional; default: goose)
  MC_API           API preference for mc, space- or comma-separated (e.g. "S3v4 S3v2") (optional)

Examples:
  S3_BUCKET=goosed-artifacts S3_ENDPOINT=http://localhost:8333 \
  S3_ACCESS_KEY=goosed S3_SECRET_KEY=goosedsecret \
  ./copy-iso.sh ~/Downloads/Rocky-9.4-x86_64-minimal.iso
USAGE
}

if [[ "${1:-}" =~ ^(-h|--help)$ ]]; then usage; exit 0; fi

ISO_PATH=${1:-}
if [[ -z "$ISO_PATH" ]]; then
  read -r -p "Path to ISO: " ISO_PATH
fi
if [[ -z "$ISO_PATH" ]]; then echo "No ISO path provided" >&2; exit 1; fi
if [[ ! -f "$ISO_PATH" ]]; then echo "ISO not found: $ISO_PATH" >&2; exit 1; fi

: "${S3_BUCKET:?S3_BUCKET must be set}"
: "${S3_ENDPOINT:?S3_ENDPOINT must be set}"
: "${S3_ACCESS_KEY:?S3_ACCESS_KEY must be set}"
: "${S3_SECRET_KEY:?S3_SECRET_KEY must be set}"

# macOS-friendly absolute path (avoid realpath)
abspath() (
  cd "$(dirname "$1")" >/dev/null 2>&1
  printf "%s/%s\n" "$(pwd -P)" "$(basename "$1")"
)

ISO_PATH=$(abspath "$ISO_PATH")
ISO_FILENAME=$(basename "$ISO_PATH")
ISO_S3_KEY=${ISO_S3_KEY:-"artifacts/rocky/9/$ISO_FILENAME"}
MC_ALIAS=${MC_ALIAS:-goose}
MC_API_RAW=${MC_API:-}
S3_REGION=${S3_REGION:-us-east-1}

TMP_DIR=""
cleanup() { [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]] && rm -rf "$TMP_DIR"; }
trap cleanup EXIT

have_cmd() { command -v "$1" >/dev/null 2>&1; }

aws_env() {
  AWS_ACCESS_KEY_ID="$S3_ACCESS_KEY" \
  AWS_SECRET_ACCESS_KEY="$S3_SECRET_KEY" \
  AWS_DEFAULT_REGION="$S3_REGION" \
  "$@"
}

bucket_exists_with_aws() {
  aws_env aws --endpoint-url "$S3_ENDPOINT" s3 ls "s3://$S3_BUCKET" >/dev/null 2>&1
}

create_bucket_with_aws() {
  echo "Ensuring bucket s3://$S3_BUCKET exists (aws)…"
  aws_env aws --endpoint-url "$S3_ENDPOINT" s3 mb "s3://$S3_BUCKET" >/dev/null 2>&1 || true
}

upload_with_aws() {
  echo "Uploading via aws → s3://$S3_BUCKET/$ISO_S3_KEY"
  create_bucket_with_aws
  aws_env aws --endpoint-url "$S3_ENDPOINT" s3 cp "$ISO_PATH" "s3://$S3_BUCKET/$ISO_S3_KEY"
}

ensure_mc() {
  if have_cmd mc; then
    echo "Using system mc"
    echo "mc version: $(mc --version 2>/dev/null || true)"
    return 0
  fi
  echo "mc not found; downloading a temporary copy for macOS…"
  local os="darwin" arch url
  case "$(uname -m)" in
    arm64) arch="arm64" ;;
    x86_64|amd64) arch="amd64" ;;
    *) echo "Unsupported CPU arch $(uname -m) for mc bootstrap" >&2; return 1 ;;
  esac
  url="https://dl.min.io/client/mc/release/${os}-${arch}/mc"
  TMP_DIR="$(mktemp -d)"
  curl -fsSL "$url" -o "$TMP_DIR/mc"
  chmod +x "$TMP_DIR/mc"
  export PATH="$TMP_DIR:$PATH"
  echo "Downloaded mc to $TMP_DIR/mc"
}

mc_alias_set_and_check() {
  local api="$1"
  local args=()
  if [[ -n "$api" ]]; then
    echo "Configuring mc alias '$MC_ALIAS' for $S3_ENDPOINT (api=$api)…"
    args=(--api "$api")
  else
    echo "Configuring mc alias '$MC_ALIAS' for $S3_ENDPOINT…"
  fi

  if ! mc alias set "${args[@]}" "$MC_ALIAS" "$S3_ENDPOINT" "$S3_ACCESS_KEY" "$S3_SECRET_KEY" 2>&1; then
    return 2
  fi

  local out
  if ! out=$(mc ls "$MC_ALIAS/$S3_BUCKET" 2>&1); then
    if grep -qi 'InvalidAccessKeyId\|The access key ID you provided does not exist' <<<"$out"; then
      echo "mc credential validation failed: InvalidAccessKeyId. Check S3_ACCESS_KEY/S3_SECRET_KEY." >&2
      return 3
    fi
    if grep -qi 'SignatureDoesNotMatch' <<<"$out"; then
      echo "mc signature mismatch with API '$api'; will try another API…" >&2
      return 4
    fi
  fi
  return 0
}

create_bucket_with_mc() {
  if ! mc ls "$MC_ALIAS/$S3_BUCKET" >/dev/null 2>&1; then
    echo "Ensuring bucket $MC_ALIAS/$S3_BUCKET exists (mc)…"
    mc mb "$MC_ALIAS/$S3_BUCKET" >/dev/null 2>&1 || true
  fi
}

upload_with_mc() {
  ensure_mc

  local attempts=()
  if [[ -n "$MC_API_RAW" ]]; then
    IFS=', ' read -r -a attempts <<<"$MC_API_RAW"
  else
    attempts=(S3v2 S3v4 "")
  fi

  local api
  for api in "${attempts[@]}"; do
    if mc_alias_set_and_check "$api"; then
      create_bucket_with_mc
      echo "Uploading via mc → $MC_ALIAS/$S3_BUCKET/$ISO_S3_KEY"
      mc cp "$ISO_PATH" "$MC_ALIAS/$S3_BUCKET/$ISO_S3_KEY"
      return 0
    else
      case $? in
        3) return 1 ;;   # InvalidAccessKeyId -> don't keep flipping APIs; keys are wrong.
        4) : ;;          # Signature mismatch -> try next API
        *) : ;;          # generic -> try next API
      esac
    fi
  done

  echo "Unable to upload via mc after exhausting API options." >&2
  return 1
}

if have_cmd aws; then
  if upload_with_aws; then
    echo "Done: s3://$S3_BUCKET/$ISO_S3_KEY"
    exit 0
  fi
  echo "aws upload failed; falling back to mc…" >&2
fi

upload_with_mc
echo "Done: s3://$S3_BUCKET/$ISO_S3_KEY"
