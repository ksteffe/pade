#!/usr/bin/env bash
# Deterministic stand-in for the 1Password CLI (`op read`) used by Milestone 6 dogfood/CI.
# Not a secrets manager — returns fixed demo values for known op:// refs only.
set -euo pipefail

if [[ "${1:-}" != "read" ]]; then
  echo "fake-op: only 'read <op://...>' is supported" >&2
  exit 2
fi

ref="${2:-}"
case "$ref" in
  "op://pade-demo/google-analytics/property_id")
    printf '%s\n' "op-demo-property"
    ;;
  "op://pade-demo/google-analytics/credentials_path")
    printf '%s\n' "/tmp/op-ga.json"
    ;;
  "op://pade-demo/developers/alice/google-analytics/property_id")
    printf '%s\n' "alice-op-property"
    ;;
  "op://pade-demo/developers/alice/google-analytics/credentials_path")
    printf '%s\n' "/tmp/alice-op-ga.json"
    ;;
  "op://pade-demo/developers/bob/google-analytics/property_id")
    printf '%s\n' "bob-op-property"
    ;;
  "op://pade-demo/developers/bob/google-analytics/credentials_path")
    printf '%s\n' "/tmp/bob-op-ga.json"
    ;;
  *)
    echo "fake-op: unknown ref" >&2
    exit 1
    ;;
esac
