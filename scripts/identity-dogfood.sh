#!/usr/bin/env bash
# Milestone 5: same portable pade.yaml, two simulated identities, different material.
# Secret values must not appear in PADE plan/capabilities output; the child only
# prints identity-ok:<name> labels after a private match.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PADE="${PADE:-$ROOT/bin/pade}"
MANIFEST="$ROOT/examples/demo-project/pade.yaml"
ALICE_BINDINGS="$ROOT/examples/demo-project/identities/alice.bindings.yaml"
BOB_BINDINGS="$ROOT/examples/demo-project/identities/bob.bindings.yaml"

if [[ ! -x "$PADE" ]]; then
  echo "error: pade binary not found at $PADE (run: make build)" >&2
  exit 1
fi

"$PADE" validate -f "$MANIFEST"

# Binding files differ only by documented identity; capability names match.
"$PADE" plan -f "$MANIFEST" --bindings "$ALICE_BINDINGS" --json >/tmp/pade-identity-alice-plan.json
"$PADE" plan -f "$MANIFEST" --bindings "$BOB_BINDINGS" --json >/tmp/pade-identity-bob-plan.json

# Refuse if plan JSON accidentally contains demo secret substrings.
for f in /tmp/pade-identity-alice-plan.json /tmp/pade-identity-bob-plan.json; do
  if grep -E 'alice-ga-property|bob-ga-property|alice-ga\.json|bob-ga\.json' "$f" >/dev/null; then
    echo "error: plan output appears to contain credential material: $f" >&2
    exit 1
  fi
done

run_identity() {
  local name="$1"
  local bindings="$2"
  local property_id="$3"
  local creds_path="$4"

  GA_PROPERTY_ID="$property_id" \
  GOOGLE_APPLICATION_CREDENTIALS="$creds_path" \
  "$PADE" exec \
    -f "$MANIFEST" \
    --bindings "$bindings" \
    --capability google-analytics.read \
    -- /bin/sh -c '
      expected_property="$1"
      expected_creds="$2"
      label="$3"
      if [ "$GA_PROPERTY_ID" != "$expected_property" ]; then
        echo "identity-mismatch:property" >&2
        exit 1
      fi
      if [ "$GOOGLE_APPLICATION_CREDENTIALS" != "$expected_creds" ]; then
        echo "identity-mismatch:creds" >&2
        exit 1
      fi
      # Parent shell must not leave ambient demo values for the other identity.
      printf "identity-ok:%s\n" "$label"
    ' sh "$property_id" "$creds_path" "$name"
}

alice_out="$(run_identity alice "$ALICE_BINDINGS" "alice-ga-property" "/tmp/alice-ga.json")"
bob_out="$(run_identity bob "$BOB_BINDINGS" "bob-ga-property" "/tmp/bob-ga.json")"

test "$alice_out" = "identity-ok:alice"
test "$bob_out" = "identity-ok:bob"
test "$alice_out" != "$bob_out"

echo "dogfood-identity: ok"
