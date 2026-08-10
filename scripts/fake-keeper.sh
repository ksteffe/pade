#!/usr/bin/env bash
# Deterministic stand-in for Keeper Commander used by Milestone 7 dogfood/CI.
# Supports:
#   get --format=password [--unmask] <uid>
#   find-password <uid>
#   clipboard-copy --output stdout <uid>
#   get --format=json --unmask <uid>
# Returns fixed demo tokens (pade-demo-* prefix) for known UIDs only.
set -euo pipefail

uid=""
mode=""

if [[ "${1:-}" == "get" && "${2:-}" == "--format=password" ]]; then
  mode="password"
  if [[ "${3:-}" == "--unmask" ]]; then
    uid="${4:-}"
  else
    uid="${3:-}"
  fi
elif [[ "${1:-}" == "get" && "${2:-}" == "--format=json" ]]; then
  mode="json"
  if [[ "${3:-}" == "--unmask" ]]; then
    uid="${4:-}"
  else
    uid="${3:-}"
  fi
elif [[ "${1:-}" == "find-password" ]]; then
  mode="password"
  uid="${2:-}"
elif [[ "${1:-}" == "clipboard-copy" && "${2:-}" == "--output" && "${3:-}" == "stdout" ]]; then
  mode="password"
  uid="${4:-}"
else
  echo "fake-keeper: unsupported invocation: $*" >&2
  exit 2
fi

token=""
case "$uid" in
  "pade-demo-github") token="pade-demo-keeper-token" ;;
  "pade-demo-alice-github") token="pade-demo-alice-keeper-token" ;;
  "pade-demo-bob-github") token="pade-demo-bob-keeper-token" ;;
  *)
    echo "fake-keeper: unknown uid" >&2
    exit 1
    ;;
esac

if [[ "$mode" == "json" ]]; then
  printf '{"uid":"%s","type":"login","fields":[{"type":"password","value":["%s"]}]}\n' "$uid" "$token"
else
  printf '%s\n' "$token"
fi
