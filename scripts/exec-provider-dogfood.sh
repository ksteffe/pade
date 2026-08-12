#!/usr/bin/env bash
# Dogfood provider: exec (Milestone B–C). Optional GitHub fake provider toward D.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
PADE="${PADE:-$ROOT/bin/pade}"
GO="${GO:-go}"
MODE="${1:-stub}" # stub | github

die() {
  echo "error: $*" >&2
  exit 1
}

[[ -x "$PADE" ]] || die "pade binary not found at $PADE (run: make build)"

STUB_OUT="${ROOT}/bin/pade-provider-stub"
GH_OUT="${ROOT}/bin/pade-provider-github"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/pade-exec-provider.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "${ROOT}/bin"

case "$MODE" in
  stub)
    "$GO" build -o "${STUB_OUT}" ./examples/providers/stub
    COMMAND_PATH="${STUB_OUT}"
    CAP="demo.derived"
    CHECK_ENV="DEMO_TOKEN"
    EXPECT="stub-ok"
    CONFIG_YAML="        tokenEnv: DEMO_TOKEN
        value: stub-ok"
    unset PADE_PROVIDER_FAKE || true
    ;;
  github)
    "$GO" build -o "${GH_OUT}" ./examples/providers/github
    COMMAND_PATH="${GH_OUT}"
    CAP="github.repo.read"
    CHECK_ENV="GITHUB_TOKEN"
    EXPECT="ghs_pade_fake_installation_token"
    CONFIG_YAML="        tokenEnv: GITHUB_TOKEN"
    export PADE_PROVIDER_FAKE=1
    ;;
  *)
    die "usage: $0 [stub|github]"
    ;;
esac

cat >"${TMP}/pade.yaml" <<EOF
apiVersion: pade.local/v1alpha1
kind: DevelopmentSession
metadata:
  name: exec-provider-dogfood
spec:
  capabilities:
    ${CAP}:
      access: use
EOF

cat >"${TMP}/bindings.yaml" <<EOF
version: "0.1"
capabilities:
  ${CAP}:
    provider: exec
    exec:
      command:
        - ${COMMAND_PATH}
      config:
${CONFIG_YAML}
EOF

echo "=== exec provider dogfood (${MODE}): validate ==="
"$PADE" validate -f "${TMP}/pade.yaml"

echo "=== exec provider dogfood (${MODE}): capabilities ==="
"$PADE" capabilities -f "${TMP}/pade.yaml" --bindings "${TMP}/bindings.yaml"

echo "=== exec provider dogfood (${MODE}): exec (assert env; do not print secret) ==="
"$PADE" exec -f "${TMP}/pade.yaml" --bindings "${TMP}/bindings.yaml" \
  --capability "${CAP}" --quiet -- \
  /bin/sh -c "test \"\$${CHECK_ENV}\" = '${EXPECT}'"

echo "ok: exec provider dogfood (${MODE})"
