#!/usr/bin/env bash
# Milestone 6: resolve capabilities via the onepassword provider.
# Uses scripts/fake-op.sh by default so CI does not need a real 1Password account.
# Secret values must not appear in PADE plan/capabilities output.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PADE="${PADE:-$ROOT/bin/pade}"
MANIFEST="$ROOT/examples/demo-project/pade.yaml"
SHARED_BINDINGS="$ROOT/examples/demo-project/bindings.onepassword.example.yaml"
ALICE_BINDINGS="$ROOT/examples/demo-project/identities/alice.onepassword.bindings.yaml"
BOB_BINDINGS="$ROOT/examples/demo-project/identities/bob.onepassword.bindings.yaml"
FAKE_OP="${PADE_OP_BIN:-$ROOT/scripts/fake-op.sh}"

export PADE_OP_BIN="$FAKE_OP"

die() {
  echo "error: $*" >&2
  exit 1
}

[[ -x "$PADE" ]] || die "pade binary not found at $PADE (run: make build)"
chmod +x "$FAKE_OP"

assert_no_secret_leak() {
  local file="$1"
  if grep -E 'op-demo-property|alice-op-property|bob-op-property|/tmp/op-ga\.json|/tmp/alice-op-ga\.json|/tmp/bob-op-ga\.json' "$file" >/dev/null; then
    die "PADE output appears to contain 1Password credential material: $file"
  fi
}

echo "=== 1Password shared binding (via PADE_OP_BIN) ==="
unset GA_PROPERTY_ID GOOGLE_APPLICATION_CREDENTIALS GA_ACCESS_TOKEN || true

"$PADE" validate -f "$MANIFEST"
"$PADE" plan -f "$MANIFEST" --bindings "$SHARED_BINDINGS" --json >/tmp/pade-op-plan.json
assert_no_secret_leak /tmp/pade-op-plan.json

"$PADE" capabilities -f "$MANIFEST" --bindings "$SHARED_BINDINGS" --json >/tmp/pade-op-capabilities.json
assert_no_secret_leak /tmp/pade-op-capabilities.json
grep -Eq '"status"[[:space:]]*:[[:space:]]*"available"' /tmp/pade-op-capabilities.json \
  || die "expected onepassword capability status available"

shared_out="$(
  "$PADE" exec \
    -f "$MANIFEST" \
    --bindings "$SHARED_BINDINGS" \
    --capability google-analytics.read \
    -- /bin/sh -c '
      test "$GA_PROPERTY_ID" = "op-demo-property" || exit 1
      test "$GOOGLE_APPLICATION_CREDENTIALS" = "/tmp/op-ga.json" || exit 1
      printf "onepassword-ok:shared\n"
    '
)"
test "$shared_out" = "onepassword-ok:shared" || die "shared onepassword exec failed: $shared_out"

echo "=== 1Password Alice/Bob identity refs ==="
"$PADE" plan -f "$MANIFEST" --bindings "$ALICE_BINDINGS" --json >/tmp/pade-op-alice-plan.json
"$PADE" plan -f "$MANIFEST" --bindings "$BOB_BINDINGS" --json >/tmp/pade-op-bob-plan.json
assert_no_secret_leak /tmp/pade-op-alice-plan.json
assert_no_secret_leak /tmp/pade-op-bob-plan.json

alice_out="$(
  "$PADE" exec \
    -f "$MANIFEST" \
    --bindings "$ALICE_BINDINGS" \
    --capability google-analytics.read \
    -- /bin/sh -c '
      test "$GA_PROPERTY_ID" = "alice-op-property" || exit 1
      test "$GOOGLE_APPLICATION_CREDENTIALS" = "/tmp/alice-op-ga.json" || exit 1
      printf "onepassword-ok:alice\n"
    '
)"
bob_out="$(
  "$PADE" exec \
    -f "$MANIFEST" \
    --bindings "$BOB_BINDINGS" \
    --capability google-analytics.read \
    -- /bin/sh -c '
      test "$GA_PROPERTY_ID" = "bob-op-property" || exit 1
      test "$GOOGLE_APPLICATION_CREDENTIALS" = "/tmp/bob-op-ga.json" || exit 1
      printf "onepassword-ok:bob\n"
    '
)"

test "$alice_out" = "onepassword-ok:alice" || die "alice onepassword exec failed: $alice_out"
test "$bob_out" = "onepassword-ok:bob" || die "bob onepassword exec failed: $bob_out"
test "$alice_out" != "$bob_out"

echo "dogfood-onepassword: ok"
