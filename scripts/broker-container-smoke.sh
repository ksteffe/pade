#!/usr/bin/env bash
# Smoke-test the Docker-built pade-broker image (packaging proof).
# Does not require Cursor, Keeper, KSM_CONFIG, or GCP.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${PADE_BROKER_IMAGE:-pade-broker:ci}"
CONTAINER_NAME="${PADE_BROKER_CONTAINER:-pade-broker-smoke-$$}"
HOST_PORT="${PADE_BROKER_SMOKE_PORT:-18087}"
CONTAINER_PORT=8080
WORK="$(mktemp -d "${TMPDIR:-/tmp}/pade-broker-container-smoke.XXXXXX")"

cleanup() {
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

die() {
  echo "error: $*" >&2
  if docker inspect "$CONTAINER_NAME" >/dev/null 2>&1; then
    echo "--- container logs ---" >&2
    docker logs "$CONTAINER_NAME" >&2 || true
  fi
  exit 1
}

command -v docker >/dev/null || die "docker is required"
command -v curl >/dev/null || die "curl is required"

echo "building pade-broker image (${IMAGE})..."
docker build -t "$IMAGE" "$ROOT"
echo "building pade-broker image... ok"

# Minimal policy/bindings: healthz + unauthenticated resolve only (no KSM/Cursor).
cat >"$WORK/policy.yaml" <<'EOF'
version: "0.1"
oidc:
  issuer: https://api.cursor.com
  audience: https://pade-broker.local
policies:
  - subject: "user:container-smoke"
    requireRepoURLs: false
    capabilities:
      - github.user.read
EOF

cat >"$WORK/bindings.yaml" <<'EOF'
version: "0.1"
capabilities:
  github.user.read:
    provider: env
    env:
      - GITHUB_TOKEN
EOF

echo "starting broker container..."
docker run -d \
  --name "$CONTAINER_NAME" \
  -p "127.0.0.1:${HOST_PORT}:${CONTAINER_PORT}" \
  -e "PORT=${CONTAINER_PORT}" \
  -v "$WORK/policy.yaml:/config/policy.yaml:ro" \
  -v "$WORK/bindings.yaml:/config/bindings.yaml:ro" \
  "$IMAGE" \
  -tls-termination=proxy \
  -policy /config/policy.yaml \
  -bindings /config/bindings.yaml \
  >/dev/null

# Light packaging assertions
user="$(docker inspect -f '{{.Config.User}}' "$CONTAINER_NAME")"
[[ -n "$user" && "$user" != "0" && "$user" != "root" && "$user" != "0:0" ]] \
  || die "expected non-root container user, got ${user:-empty}"

ready=0
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${HOST_PORT}/healthz" >/dev/null 2>&1; then
    ready=1
    break
  fi
  if ! docker inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null | grep -q true; then
    die "container exited before becoming ready"
  fi
  sleep 1
done
[[ "$ready" -eq 1 ]] || die "timed out waiting for /healthz"

echo "starting broker container... ok"

hz="$(curl -fsS -w '\n%{http_code}' "http://127.0.0.1:${HOST_PORT}/healthz")"
hz_body="$(printf '%s' "$hz" | head -n 1)"
hz_code="$(printf '%s' "$hz" | tail -n 1)"
[[ "$hz_code" == "200" && "$hz_body" == "ok" ]] || die "GET /healthz unexpected: code=$hz_code body=$hz_body"
echo "GET /healthz... 200"

code="$(
  curl -sS -o "$WORK/resolve.body" -w '%{http_code}' \
    -X POST "http://127.0.0.1:${HOST_PORT}/v1/resolve" \
    -H 'Content-Type: application/json' \
    -d '{"capability":"github.user.read"}'
)"
[[ "$code" == "401" ]] || die "POST /v1/resolve without bearer expected 401, got $code body=$(cat "$WORK/resolve.body")"
echo "POST /v1/resolve without bearer token... rejected as expected (401)"

echo "container smoke... PASS"
