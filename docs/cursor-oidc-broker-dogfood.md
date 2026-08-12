# Cursor OIDC broker dogfood (Phase 2 spike)

Prove a clean evolutionary path toward **removing `KSM_CONFIG` from the Cursor Cloud Agent VM**:

```text
Cursor Cloud Agent
  → mint short-lived Cursor OIDC JWT
  → PADE broker (server-side policy)
  → Keeper Secrets Manager (broker-side KSM_CONFIG only)
  → process-scoped material via pade exec
```

> **Spec context:** Cursor OIDC is one **workload identity** adapter used by the reference Consumer/Broker dogfood. It is **not** a provider adapter and is **not** required by the PADE specification. `pade-broker` is an experimental reference Broker; the draft wire protocol is documented in [../spec/broker.md](../spec/broker.md). See also [../spec/consumer.md](../spec/consumer.md).

The portable `DevelopmentSession` in `pade.yaml` keeps provider-specific details out of `spec.capabilities`. Direct `keeper-secrets-manager` (Milestone 9) remains available for local/Cloud Agents that still use ambient `KSM_CONFIG`.

Broker policy and agent bindings use their own `version: "0.1"` configuration format—that is **not** the Intent API version (`pade.local/v1alpha1`).

## Fake CI dogfood (no Cursor / no Keeper)

```bash
make dogfood-broker
```

Starts an in-process JWKS + `pade-broker` with `PADE_KSM_FAKE=1`, mints a test JWT matching broker policy, and runs `pade exec` through `provider: broker`. Included in `make ci-smoke`.

## Stage A — Cursor identity proof (real Cloud Agent)

One-time: build PADE in the Cloud Agent environment (`make build` or `go build -o bin/pade ./cmd/pade`).

From a Cursor Cloud Agent VM (identity socket present):

```bash
./bin/pade identity --audience https://pade-broker.example
./bin/pade identity --audience https://pade-broker.example --json
```

Expected: safe claims (`subject`, `cloud_agent`, `runtime`, `repos`, `expires`). **Raw JWT is never printed.**

No Keeper or broker required for Stage A.

Notes:

- Socket: `CURSOR_AGENT_SOCKET` or `/run/cursor/api.sock`
- Token authenticates the **Cloud Agent workload**, not a subprocess
- Prefer `subject` / `owner_user_id` over email for policy
- **Observed on managed Cloud Agents:** `repo_url` (primary) may be present while `repo_urls` / `repo_count` are absent. Missing `repo_urls` means the complete repository set is **unknown** — not proof of single-repo confinement. Do not authorize from `repo_url` alone. Policies with `requireRepoURLs: true` fail closed until Cursor attests the complete set.

## Stage B — real OIDC + local broker (Cloud Agent)

Prove the broker path on a Cloud Agent VM **without** putting `KSM_CONFIG` on the agent:

```text
real Cursor OIDC (identity socket)
  → local pade-broker on 127.0.0.1 (real JWKS from api.cursor.com)
  → fake KSM on the broker process only (PADE_KSM_FAKE=1)
  → pade exec injects process-scoped material
```

```bash
make dogfood-broker-stage-b
# optional pin: PADE_STAGE_B_SUBJECT=user:<id> make dogfood-broker-stage-b
```

Not included in CI (requires a live Cursor identity socket). The script:

1. Mints identity via `pade identity` (safe claims only; no raw JWT).
2. Writes a **session-local** broker policy for the attested `subject` + `github.user.read` with **`requireRepoURLs: false`** (see Stage A finding).
3. Starts `pade-broker` on loopback with server-side fake KSM bindings.
4. Runs `pade` plan/capabilities/exec through `provider: broker` with **no** `PADE_BROKER_FAKE_JWT` and **no** agent `KSM_CONFIG`.

Example policy shape: [`spec/examples/broker-policy.stage-b.example.yaml`](../spec/examples/broker-policy.stage-b.example.yaml).

Production policies should be **static allowlists**, not generated from the current token. Stage B generates policy only to dogfood the path.

## Stage C — external broker + real KSM (manual host)

Next live step after Stage B: keep the same OIDC + policy path, but run the broker **outside** the agent VM with real Keeper Secrets Manager.

### Broker host (outside the agent VM)

1. Narrowly scoped Keeper Secrets Manager Application + `KSM_CONFIG` on the **broker host only**.
2. Server-side policy (example: [`spec/examples/broker-policy.example.yaml`](../spec/examples/broker-policy.example.yaml)). Use `requireRepoURLs: true` only when Cursor attests complete `repo_urls`; otherwise follow the Stage B subject + capability pattern until that attestation exists.

```yaml
version: "0.1"
oidc:
  issuer: https://api.cursor.com
  audience: https://pade-broker.example
policies:
  - subject: "user:<your-cursor-user-id>"
    requireRepoURLs: true
    repositories:
      - github.com/ksteffe/pade
    capabilities:
      - github.user.read
```

3. Server-side bindings (reuses existing providers; never commit real UIDs with secrets):

```yaml
version: "0.1"
capabilities:
  github.user.read:
    provider: keeper-secrets-manager
    keeperSecretsManager:
      refs:
        GITHUB_TOKEN: "keeper://RECORD_UID/field/password"
```

4. Run (choose one transport model):

**Loopback / tunnel to the broker host** (plaintext OK on loopback only):

```bash
export KSM_CONFIG="$(base64 -w0 ksm-config.json)"
./bin/pade-broker \
  -policy /path/to/broker-policy.yaml \
  -bindings /path/to/broker-bindings.yaml \
  -listen 127.0.0.1:8787
```

**Broker-managed TLS** (directly exposed non-loopback listener):

```bash
./bin/pade-broker \
  -policy /path/to/broker-policy.yaml \
  -bindings /path/to/broker-bindings.yaml \
  -listen 0.0.0.0:8787 \
  -tls-cert /path/to/cert.pem \
  -tls-key /path/to/key.pem
```

**Trusted upstream TLS termination** (Cloud Run, Kubernetes ingress, HTTPS load balancer, etc.):

```bash
# Container-friendly: omit -listen and set PORT (listens on 0.0.0.0:$PORT).
# Still requires explicit -tls-termination=proxy — PORT alone is not a TLS opt-in.
export PORT=8080
./bin/pade-broker \
  -policy /path/to/broker-policy.yaml \
  -bindings /path/to/broker-bindings.yaml \
  -tls-termination=proxy
```

`-tls-termination=proxy` is an operator assertion that TLS is terminated by a trusted deployment boundary and that plaintext is not reachable outside it. PADE does not verify Cloud Run / ingress / LB wiring. Non-loopback plaintext without this flag or broker-managed cert/key is rejected. See [SECURITY.md](../SECURITY.md).

### Container image (pade-broker)

The repository root [`Dockerfile`](../Dockerfile) builds a production-oriented **pade-broker-only** image (distroless static, non-root). No secrets or policy are baked in.

```bash
docker build -t pade-broker:ci .
make smoke-broker-container   # build + /healthz + unauthenticated /v1/resolve → 401
```

Example run (trusted upstream TLS termination; mounts are deployment-specific):

```bash
docker run --rm -p 8080:8080 -e PORT=8080 \
  -v "$PWD/policy.yaml:/config/policy.yaml:ro" \
  -v "$PWD/bindings.yaml:/config/bindings.yaml:ro" \
  pade-broker:ci \
  -tls-termination=proxy \
  -policy /config/policy.yaml \
  -bindings /config/bindings.yaml
# Image defaults PORT=8080; -e PORT=8080 shown for Cloud Run parity.
```

Probe `GET /healthz` (no auth). There is no Docker `HEALTHCHECK` (distroless has no curl); Cloud Run / Kubernetes should use the HTTP probe.

The same image is intended for Cloud Run, Kubernetes ingress, reverse proxies, or local Docker. Full Cloud Run deployment (registry, Secret Manager, custom domain) is a later milestone.

Google Cloud Run is a planned composition example (container listens via `PORT` with plaintext inside the platform; Cloud Run terminates external HTTPS). It is not a normative PADE dependency.

Reachability from a Cloud Agent (public HTTPS URL, SSH tunnel, Tailscale, etc.) is a **deployment concern**. Temporary tunnels are composition options, not PADE architecture.

### Agent bindings (runtime / org local)

```yaml
version: "0.1"
capabilities:
  github.user.read:
    provider: broker
    broker:
      endpoint: https://pade-broker.example
      audience: https://pade-broker.example
```

Example: [`spec/examples/bindings.broker.example.yaml`](../spec/examples/bindings.broker.example.yaml).

Agent VM should **not** have `KSM_CONFIG` in this mode.

```bash
./bin/pade exec \
  -f examples/demo-project/pade.yaml \
  --bindings /path/to/agent-broker.bindings.yaml \
  --capability github.user.read -- \
  examples/demo-project/scripts/github-whoami
```

### Next dogfood (Cloud Run — not implemented in this repo yet)

```text
Cursor Cloud Agent
  → Cursor OIDC
  → public HTTPS Cloud Run endpoint
  → pade-broker (PORT + -tls-termination=proxy)
  → broker-side Keeper Secrets Manager
```

Remaining work for that milestone: push the proven container image to a registry; deploy to Cloud Run; static policy/bindings; `KSM_CONFIG` via Secret Manager; external HTTPS URL as OIDC audience + agent broker endpoint; invoke from a real Cloud Agent with no agent `KSM_CONFIG`.

## Trust boundaries

| Mode | Where KSM_CONFIG lives | Authorization |
|------|------------------------|---------------|
| Milestone 9 direct KSM | Agent VM | PADE reduces accidental propagation; VM is not sandboxed |
| Stage B local broker | Broker process only (fake KSM OK) | Server policy on Cursor subject + capability (`requireRepoURLs` only when complete `repo_urls` attested) |
| Stage C external broker | Broker host / platform only | Same; prefer complete `repo_urls` when available |

Transport (separate from authorization):

| Listener | TLS |
|----------|-----|
| Loopback | Plain HTTP OK |
| Non-loopback direct | Broker `-tls-cert` / `-tls-key` |
| Non-loopback behind trusted proxy | `-tls-termination=proxy` (deployment assertion) |

- Cursor OIDC authenticates the Cloud Agent **workload**, not an individual subprocess.
- Any process that can reach Cursor’s identity socket can mint a token for that workload.
- A capability name in `pade.yaml` is a **request**, not authorization.
- Broker must not trust capability names merely because the client asks or the repo declares them.
- Broker logs: identity/capability decisions only — never JWTs or resolved credentials.
- Downstream APIs remain authoritative for final authorization.
- JTI replay tracking is **deferred**; this spike relies on short-lived tokens (5m) + exact audience binding.
- TLS termination (broker or upstream) does **not** replace Cursor OIDC or broker policy.

## Deferred

Multi-user admin UI, hosted SaaS, DB-backed policy, SPIFFE / GitHub Actions OIDC, other agent IdPs, OAuth consent, leasing/rotation, OPA, HA, normative tunnel products, Cloud Run deployment automation, replacing direct KSM mode.