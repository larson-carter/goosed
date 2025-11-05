#!/usr/bin/env bash
set -euo pipefail

if ! command -v git >/dev/null 2>&1; then
  echo "pre-push: git command not found" >&2
  exit 1
fi

if ! git diff --quiet --ignore-submodules HEAD; then
  echo "pre-push: aborting push because the working tree has uncommitted changes" >&2
  git status --short >&2
  exit 1
fi

if ! git diff --cached --quiet --ignore-submodules; then
  echo "pre-push: aborting push because there are staged but uncommitted changes" >&2
  git status --short >&2
  exit 1
fi

exit 0
