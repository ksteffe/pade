# Cursor OIDC broker dogfood (Phase 2 spike)

Prove a clean evolutionary path toward **removing `KSM_CONFIG` from the Cursor Cloud Agent VM**:

```text
Cursor Cloud Agent
  → mint short-lived Cursor OIDC JWT
  → PADE broker (server-side policy)
  → Keeper Secrets Manager (broker-side KSM_CONFIG only)
  → process-scoped material via pade exec
```

Portable `pade.yaml` stays capability-only. Direct `keeper-secrets-manager` (Milestone 9) remains available for local/Cloud Agents that still use ambient `KSM_CONFIG`.

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

## Stage B — real broker (manual host)

### Broker host (outside the agent VM)

1. Narrowly scoped Keeper Secrets Manager Application + `KSM_CONFIG` on the **broker host only**.
2. Server-side policy (example: [`spec/examples/broker-policy.example.yaml`](../spec/examples/broker-policy.example.yaml)):

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

4. Run:

```bash
export KSM_CONFIG="$(base64 -w0 ksm-config.json)"
./bin/pade-broker \
  -policy /path/to/broker-policy.yaml \
  -bindings /path/to/broker-bindings.yaml \
  -listen 127.0.0.1:8787
```

Non-loopback listen requires `-tls-cert` / `-tls-key`. Plain HTTP is for localhost tests only.

Reachability from a Cloud Agent is a **deployment concern** (SSH tunnel, Tailscale, reverse proxy, etc.). Temporary tunnels are composition options, not PADE architecture.

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

## Trust boundaries

| Mode | Where KSM_CONFIG lives | Authorization |
|------|------------------------|---------------|
| Milestone 9 direct KSM | Agent VM | PADE reduces accidental propagation; VM is not sandboxed |
| Phase 2 broker | Broker host only | Server policy on Cursor subject + `repo_urls` + capability allowlist |

- Cursor OIDC authenticates the Cloud Agent **workload**, not an individual subprocess.
- Any process that can reach Cursor’s identity socket can mint a token for that workload.
- A capability name in `pade.yaml` is a **request**, not authorization.
- Broker must not trust capability names merely because the client asks or the repo declares them.
- Broker logs: identity/capability decisions only — never JWTs or resolved credentials.
- Downstream APIs remain authoritative for final authorization.
- JTI replay tracking is **deferred**; this spike relies on short-lived tokens (5m) + exact audience binding.

## Deferred

Multi-user admin UI, hosted SaaS, DB-backed policy, SPIFFE / GitHub Actions OIDC, other agent IdPs, OAuth consent, leasing/rotation, OPA, HA, normative tunnel products, replacing direct KSM mode.
