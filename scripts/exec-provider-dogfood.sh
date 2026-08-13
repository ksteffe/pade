#!/usr/bin/env bash
# Dogfood provider: exec (B–C), GitHub App (D–E), Google Analytics (F), two-provider seam (G).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
PADE="${PADE:-$ROOT/bin/pade}"
GO="${GO:-go}"
MODE="${1:-stub}" # stub | github | ga | two

die() {
  echo "error: $*" >&2
  exit 1
}

[[ -x "$PADE" ]] || die "pade binary not found at $PADE (run: make build)"

STUB_OUT="${ROOT}/bin/pade-provider-stub"
GH_OUT="${ROOT}/bin/pade-provider-github"
GA_OUT="${ROOT}/bin/pade-provider-google-analytics"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/pade-exec-provider.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "${ROOT}/bin"

run_two_provider_seam() {
  echo "=== Milestone G: two-provider same-seam architectural test ==="
  echo "=== github + google-analytics unit tests ==="
  "$GO" test ./examples/providers/github/ ./examples/providers/google-analytics/ -count=1
  "$GO" build -o "${GH_OUT}" ./examples/providers/github
  "$GO" build -o "${GA_OUT}" ./examples/providers/google-analytics
  export PADE_PROVIDER_FAKE=1

  cat >"${TMP}/pade.yaml" <<EOF
apiVersion: pade.local/v1alpha1
kind: DevelopmentSession
metadata:
  name: two-provider-seam-dogfood
spec:
  capabilities:
    github.repo.read:
      access: use
    google-analytics.read:
      access: read
EOF

  cat >"${TMP}/bindings.yaml" <<EOF
version: "0.1"
capabilities:
  github.repo.read:
    provider: exec
    exec:
      command:
        - ${GH_OUT}
      config:
        tokenEnv: GITHUB_TOKEN
        repositories:
          - ksteffe/pade
        permissions:
          metadata: read
          contents: read
  google-analytics.read:
    provider: exec
    exec:
      command:
        - ${GA_OUT}
      config:
        tokenEnv: GA_ACCESS_TOKEN
        propertyId: properties/000000000
EOF

  echo "=== two-provider: validate ==="
  "$PADE" validate -f "${TMP}/pade.yaml"

  echo "=== two-provider: capabilities (both on provider: exec) ==="
  out="$("$PADE" capabilities -f "${TMP}/pade.yaml" --bindings "${TMP}/bindings.yaml")"
  echo "$out"
  echo "$out" | grep -q 'provider: exec' || die "expected provider: exec"
  echo "$out" | grep -q 'github.repo.read' || die "missing github.repo.read"
  echo "$out" | grep -q 'google-analytics.read' || die "missing google-analytics.read"

  chmod +x \
    "${ROOT}/examples/demo-project/scripts/github-repo-meta" \
    "${ROOT}/examples/demo-project/scripts/ga-property-meta"

  echo "=== two-provider: resolve github.repo.read ==="
  "$PADE" exec -f "${TMP}/pade.yaml" --bindings "${TMP}/bindings.yaml" \
    --capability github.repo.read --quiet -- \
    env GITHUB_REPOSITORY=ksteffe/pade "${ROOT}/examples/demo-project/scripts/github-repo-meta"

  echo "=== two-provider: resolve google-analytics.read ==="
  "$PADE" exec -f "${TMP}/pade.yaml" --bindings "${TMP}/bindings.yaml" \
    --capability google-analytics.read --quiet -- \
    "${ROOT}/examples/demo-project/scripts/ga-property-meta"

  echo "ok: two-provider same-seam dogfood (no vendor fields in PADE core)"
}

case "$MODE" in
  two)
    run_two_provider_seam
    exit 0
    ;;
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
    echo "=== github provider unit tests (JWT + installation token httptest) ==="
    "$GO" test ./examples/providers/github/ -count=1
    "$GO" build -o "${GH_OUT}" ./examples/providers/github
    COMMAND_PATH="${GH_OUT}"
    CAP="github.repo.read"
    CHECK_ENV="GITHUB_TOKEN"
    EXPECT="ghs_pade_fake_installation_token"
    CONFIG_YAML="        tokenEnv: GITHUB_TOKEN
        repositories:
          - ksteffe/pade
        permissions:
          metadata: read
          contents: read"
    export PADE_PROVIDER_FAKE=1
    ;;
  ga)
    echo "=== google-analytics provider unit tests (SA JWT + token exchange httptest) ==="
    "$GO" test ./examples/providers/google-analytics/ -count=1
    "$GO" build -o "${GA_OUT}" ./examples/providers/google-analytics
    COMMAND_PATH="${GA_OUT}"
    CAP="google-analytics.read"
    CHECK_ENV="GA_ACCESS_TOKEN"
    EXPECT="ya29.pade_fake_access_token"
    CONFIG_YAML="        tokenEnv: GA_ACCESS_TOKEN
        propertyId: properties/000000000"
    export PADE_PROVIDER_FAKE=1
    ;;
  *)
    die "usage: $0 [stub|github|ga|two]"
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

if [[ "$MODE" == "github" ]]; then
  REPO_META="${ROOT}/examples/demo-project/scripts/github-repo-meta"
  chmod +x "$REPO_META"
  echo "=== exec provider dogfood (github): repo-scoped validation (Milestone E; fake skips network) ==="
  "$PADE" exec -f "${TMP}/pade.yaml" --bindings "${TMP}/bindings.yaml" \
    --capability "${CAP}" --quiet -- \
    env GITHUB_REPOSITORY=ksteffe/pade "$REPO_META"
fi

if [[ "$MODE" == "ga" ]]; then
  GA_META="${ROOT}/examples/demo-project/scripts/ga-property-meta"
  chmod +x "$GA_META"
  echo "=== exec provider dogfood (ga): property validation (Milestone F; fake skips network) ==="
  "$PADE" exec -f "${TMP}/pade.yaml" --bindings "${TMP}/bindings.yaml" \
    --capability "${CAP}" --quiet -- \
    "$GA_META"
fi

echo "ok: exec provider dogfood (${MODE})"
