#!/usr/bin/env bash
# Vault -dev dogfood: resolve capabilities from Vault without ambient env secrets.
# Prototype-only. Secret values must not appear in PADE plan/capabilities output.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PADE="${PADE:-$ROOT/bin/pade}"
MANIFEST="$ROOT/examples/demo-project/pade.yaml"
SHARED_BINDINGS="$ROOT/examples/demo-project/bindings.vault.example.yaml"
ALICE_BINDINGS="$ROOT/examples/demo-project/identities/alice.vault.bindings.yaml"
BOB_BINDINGS="$ROOT/examples/demo-project/identities/bob.vault.bindings.yaml"

VAULT_VERSION="${VAULT_VERSION:-1.17.6}"
VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-pade-dev}"
VAULT_BIN="${VAULT_BIN:-}"
TOOLS_DIR="$ROOT/.tools/vault"
VAULT_LOG="${VAULT_LOG:-/tmp/pade-vault-dev.log}"
VAULT_PID=""

export VAULT_ADDR VAULT_TOKEN

cleanup() {
  if [[ -n "${VAULT_PID}" ]] && kill -0 "${VAULT_PID}" 2>/dev/null; then
    kill "${VAULT_PID}" 2>/dev/null || true
    wait "${VAULT_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

die() {
  echo "error: $*" >&2
  exit 1
}

require_pade() {
  [[ -x "$PADE" ]] || die "pade binary not found at $PADE (run: make build)"
}

detect_os_arch() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) die "unsupported arch: $arch" ;;
  esac
  case "$os" in
    linux|darwin) ;;
    *) die "unsupported os: $os" ;;
  esac
  echo "${os}_${arch}"
}

ensure_vault_bin() {
  if [[ -n "$VAULT_BIN" && -x "$VAULT_BIN" ]]; then
    return
  fi
  if command -v vault >/dev/null 2>&1; then
    VAULT_BIN="$(command -v vault)"
    return
  fi
  if [[ -x "$TOOLS_DIR/vault" ]]; then
    VAULT_BIN="$TOOLS_DIR/vault"
    return
  fi

  local platform zip url
  platform="$(detect_os_arch)"
  mkdir -p "$TOOLS_DIR"
  zip="/tmp/vault_${VAULT_VERSION}_${platform}.zip"
  url="https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_${platform}.zip"
  echo "Downloading Vault ${VAULT_VERSION} (${platform})..."
  curl -fsSL -o "$zip" "$url"
  # unzip may be missing; use python as a portable fallback.
  if command -v unzip >/dev/null 2>&1; then
    unzip -o -q "$zip" vault -d "$TOOLS_DIR"
  else
    python3 - "$zip" "$TOOLS_DIR" <<'PY'
import sys, zipfile
zpath, outdir = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(zpath) as zf:
    zf.extract("vault", outdir)
PY
  fi
  chmod +x "$TOOLS_DIR/vault"
  VAULT_BIN="$TOOLS_DIR/vault"
}

wait_for_vault() {
  local i
  for i in $(seq 1 40); do
    if curl -fsS "${VAULT_ADDR}/v1/sys/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  echo "Vault failed to become ready. Log:" >&2
  cat "$VAULT_LOG" >&2 || true
  die "vault -dev did not become ready at ${VAULT_ADDR}"
}

start_vault_dev() {
  # Reuse an already-running local -dev if healthy and token matches.
  if curl -fsS "${VAULT_ADDR}/v1/sys/health" >/dev/null 2>&1; then
    if VAULT_TOKEN="$VAULT_TOKEN" "$VAULT_BIN" status >/dev/null 2>&1; then
      echo "Using existing Vault at ${VAULT_ADDR}"
      VAULT_PID=""
      return
    fi
    die "Vault is up at ${VAULT_ADDR} but VAULT_TOKEN=${VAULT_TOKEN} cannot authenticate"
  fi

  echo "Starting Vault -dev at ${VAULT_ADDR} (prototype only)..."
  "$VAULT_BIN" server -dev \
    -dev-root-token-id="$VAULT_TOKEN" \
    -dev-listen-address="${VAULT_ADDR#http://}" \
    >"$VAULT_LOG" 2>&1 &
  VAULT_PID=$!
  wait_for_vault
}

seed_secrets() {
  # KV v2 is mounted at secret/ in vault -dev.
  "$VAULT_BIN" kv put secret/pade/google-analytics \
    property_id="vault-demo-property" \
    credentials_path="/tmp/vault-ga.json" >/dev/null

  "$VAULT_BIN" kv put secret/pade/developers/alice/google-analytics \
    property_id="alice-vault-property" \
    credentials_path="/tmp/alice-vault-ga.json" >/dev/null

  "$VAULT_BIN" kv put secret/pade/developers/bob/google-analytics \
    property_id="bob-vault-property" \
    credentials_path="/tmp/bob-vault-ga.json" >/dev/null
}

assert_no_secret_leak() {
  local file="$1"
  if grep -E 'vault-demo-property|alice-vault-property|bob-vault-property|/tmp/vault-ga\.json|/tmp/alice-vault-ga\.json|/tmp/bob-vault-ga\.json' "$file" >/dev/null; then
    die "PADE output appears to contain Vault credential material: $file"
  fi
}

run_shared_vault() {
  echo "=== Vault shared binding ==="
  # Prove resolution does not depend on ambient GA_* env.
  unset GA_PROPERTY_ID GOOGLE_APPLICATION_CREDENTIALS GA_ACCESS_TOKEN || true

  "$PADE" validate -f "$MANIFEST"
  "$PADE" plan -f "$MANIFEST" --bindings "$SHARED_BINDINGS" --json >/tmp/pade-vault-plan.json
  assert_no_secret_leak /tmp/pade-vault-plan.json

  "$PADE" capabilities -f "$MANIFEST" --bindings "$SHARED_BINDINGS" --json >/tmp/pade-vault-capabilities.json
  assert_no_secret_leak /tmp/pade-vault-capabilities.json
  grep -Eq '"status"[[:space:]]*:[[:space:]]*"available"' /tmp/pade-vault-capabilities.json \
    || die "expected vault capability status available"

  local out
  out="$(
    "$PADE" exec \
      -f "$MANIFEST" \
      --bindings "$SHARED_BINDINGS" \
      --capability google-analytics.read \
      -- /bin/sh -c '
        test "$GA_PROPERTY_ID" = "vault-demo-property" || exit 1
        test "$GOOGLE_APPLICATION_CREDENTIALS" = "/tmp/vault-ga.json" || exit 1
        printf "vault-ok:shared\n"
      '
  )"
  test "$out" = "vault-ok:shared" || die "shared vault exec failed: $out"
}

run_identity_vault() {
  echo "=== Vault Alice/Bob identity paths ==="
  unset GA_PROPERTY_ID GOOGLE_APPLICATION_CREDENTIALS GA_ACCESS_TOKEN || true

  "$PADE" plan -f "$MANIFEST" --bindings "$ALICE_BINDINGS" --json >/tmp/pade-vault-alice-plan.json
  "$PADE" plan -f "$MANIFEST" --bindings "$BOB_BINDINGS" --json >/tmp/pade-vault-bob-plan.json
  assert_no_secret_leak /tmp/pade-vault-alice-plan.json
  assert_no_secret_leak /tmp/pade-vault-bob-plan.json

  local alice_out bob_out
  alice_out="$(
    "$PADE" exec \
      -f "$MANIFEST" \
      --bindings "$ALICE_BINDINGS" \
      --capability google-analytics.read \
      -- /bin/sh -c '
        test "$GA_PROPERTY_ID" = "alice-vault-property" || exit 1
        test "$GOOGLE_APPLICATION_CREDENTIALS" = "/tmp/alice-vault-ga.json" || exit 1
        printf "vault-ok:alice\n"
      '
  )"
  bob_out="$(
    "$PADE" exec \
      -f "$MANIFEST" \
      --bindings "$BOB_BINDINGS" \
      --capability google-analytics.read \
      -- /bin/sh -c '
        test "$GA_PROPERTY_ID" = "bob-vault-property" || exit 1
        test "$GOOGLE_APPLICATION_CREDENTIALS" = "/tmp/bob-vault-ga.json" || exit 1
        printf "vault-ok:bob\n"
      '
  )"

  test "$alice_out" = "vault-ok:alice" || die "alice vault exec failed: $alice_out"
  test "$bob_out" = "vault-ok:bob" || die "bob vault exec failed: $bob_out"
  test "$alice_out" != "$bob_out"
}

main() {
  require_pade
  ensure_vault_bin
  start_vault_dev
  seed_secrets
  run_shared_vault
  run_identity_vault
  echo "dogfood-vault: ok"
}

main "$@"
