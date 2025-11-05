#!/usr/bin/env bash

set -euo pipefail

export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.0}"

MODE="write"
if [[ ${1:-} == "--check" ]]; then
  MODE="check"
  shift || true
fi

if ! command -v gofmt >/dev/null 2>&1; then
  echo "gofmt is required but not installed" >&2
  exit 1
fi

mapfile -t GO_FILES < <(git ls-files '*.go')

FILTERED=()
for file in "${GO_FILES[@]}"; do
  [[ -s "$file" ]] && FILTERED+=("$file")
done

if [[ ${#FILTERED[@]} -eq 0 ]]; then
  exit 0
fi

if [[ "$MODE" == "check" ]]; then
  UNFORMATTED=$(gofmt -l "${FILTERED[@]}")
  if [[ -n "$UNFORMATTED" ]]; then
    echo "The following files are not gofmt'ed:" >&2
    echo "$UNFORMATTED" >&2
    exit 1
  fi
else
  gofmt -w "${FILTERED[@]}"
fi
