#!/usr/bin/env bash
# Deterministic stand-in for the 1Password CLI (`op read`) used by Milestone 6 dogfood/CI.
# Returns fixed demo tokens (pade-demo-* prefix) for known op:// refs only.
set -euo pipefail

if [[ "${1:-}" != "read" ]]; then
  echo "fake-op: only 'read <op://...>' is supported" >&2
  exit 2
fi

ref="${2:-}"
case "$ref" in
  "op://pade-demo/github/credential")
    printf '%s\n' "pade-demo-op-token"
    ;;
  "op://pade-demo/developers/alice/github/credential")
    printf '%s\n' "pade-demo-alice-op-token"
    ;;
  "op://pade-demo/developers/bob/github/credential")
    printf '%s\n' "pade-demo-bob-op-token"
    ;;
  *)
    echo "fake-op: unknown ref" >&2
    exit 1
    ;;
esac
