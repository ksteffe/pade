#!/usr/bin/env bash
# Deterministic stand-in for Keeper Commander (`keeper get --format=password`) used by Milestone 7 dogfood/CI.
# Returns fixed demo tokens (pade-demo-* prefix) for known keeper:// UIDs only.
set -euo pipefail

if [[ "${1:-}" != "get" || "${2:-}" != "--format=password" ]]; then
  echo "fake-keeper: only 'get --format=password <uid>' is supported" >&2
  exit 2
fi

uid="${3:-}"
case "$uid" in
  "pade-demo-github")
    printf '%s\n' "pade-demo-keeper-token"
    ;;
  "pade-demo-alice-github")
    printf '%s\n' "pade-demo-alice-keeper-token"
    ;;
  "pade-demo-bob-github")
    printf '%s\n' "pade-demo-bob-keeper-token"
    ;;
  *)
    echo "fake-keeper: unknown uid" >&2
    exit 1
    ;;
esac
