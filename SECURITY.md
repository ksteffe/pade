# Security

## Reporting

If you discover a security issue in PADE or its examples, please report it privately to the repository maintainers (GitHub Security Advisories preferred when available) rather than opening a public issue with exploit detail.

## Trust boundaries

PADE’s three specification surfaces (Intent / Consumer / Broker) imply **four security boundaries** when provider systems and downstream resources are included. Repository Intent never creates authority by itself.

```text
Intent
  request only
     |
     v
Consumer
  authenticated workload
     |
     v
Broker
  PADE authorization
     |
     v
Provider / authority system
  materialization
     |
     v
Downstream resource
  final authorization
```

### Intent boundary

Repository Intent (`pade.yaml`) is **untrusted input**. A declaration *requests* a capability. It does **not** grant authority. A malicious repository must not gain authority merely by listing capability names.

### Consumer boundary

The Consumer (reference: `pade`):

- validates Intent;
- obtains workload identity where the resolution path requires it;
- requests approved authority (via local providers or a broker);
- protects returned material;
- limits propagation as much as practical (process-scoped injection; no casual logging or persistence of secrets).

The Consumer **cannot** create authorization merely because a repository asks for a capability. Best-effort stdout/stderr redaction and child env omission are defense in depth—not a sandbox. See [spec/consumer.md](spec/consumer.md).

### Broker boundary

The Broker (reference: `pade-broker`) is an **authorization boundary**. It:

- authenticates the workload;
- validates relevant identity context;
- applies **server-owned** authorization policy;
- materializes only permitted capabilities.

It must not authorize a capability merely because Intent declared it or a client requested it. See [spec/broker.md](spec/broker.md).

### Provider / resource boundary

Credential managers, IAM systems, service providers, and downstream APIs retain their own authorization responsibilities. PADE does not replace downstream authorization.

Draft specs: [spec/README.md](spec/README.md), [spec/intent.md](spec/intent.md), [spec/consumer.md](spec/consumer.md), [spec/broker.md](spec/broker.md).

## Prototype invariants

Even for local demos (including Vault `-dev` mode):

- Never put secret values in `pade.yaml` or other committed config.
- Never print secret values in `pade plan`, status, logs, or errors.
- Never persist secrets in `.pade/` workspace state.
- Never bake resolved credentials into images or snapshots.
- Treat Vault/1Password/env/Keeper/KSM/broker adapters as **bindings / providers**, not as part of the portable Intent Specification.

**Materialization preference (roadmap):** Brokers SHOULD prefer session-scoped, short-lived, or otherwise derived credentials over delivering durable source credentials when the configured provider supports such derivation. Direct durable-secret materialization remains a valid interoperability mechanism (and is what current dogfood uses). Details and maturity stages: [ROADMAP.md](ROADMAP.md) (fulfillment maturity; Milestone I).

Local Vault development servers and root tokens are for proving seams only. They are not production-safe and must not be documented as recommended production practice.

## Process-scoped execution

`pade exec` (reference Consumer) injects resolved material only into the child process. After exit, the reference Consumer clears its in-memory material maps.

Best-effort behaviors (defense in depth, **not** security boundaries):

- Exact-match redaction of resolved secret values on child stdout/stderr before they reach the caller (helps cloud-agent transcripts). Encoded, transformed, hashed, or substring-altered values are not recognized. A child that possesses a credential can still intentionally exfiltrate it.
- Providers may omit ambient bootstrap env keys from the child (for example `keeper-secrets-manager` omits `KSM_CONFIG`) after resolution.

A process that has been given a credential can still observe, transform, encode, or exfiltrate it. Any process that can read ambient bootstrap credentials (for example `KSM_CONFIG`) can also call the secret manager directly, bypassing PADE.

## Milestone 9 — direct Keeper Secrets Manager mode

```text
agent VM possesses KSM_CONFIG
→ any process in the VM may potentially use it
→ reference Consumer reduces accidental propagation (child omit + output redaction)
→ the reference Consumer does not sandbox the VM
```

Use a narrowly scoped Keeper Secrets Manager Application so possession of `KSM_CONFIG` is not equivalent to whole-vault access.

Keeper Secrets Manager is one **reference materialization provider**. It is not part of the portable Intent Specification.

## Phase 2 — Cursor OIDC broker mode (spike)

```text
agent VM has no KSM_CONFIG
→ workload can mint Cursor OIDC identity (local socket)
→ pade-broker verifies JWT (issuer, audience, signature, nbf/exp)
→ server-side policy authorizes subject + repo_urls + capability
→ Keeper bootstrap stays on the broker host
→ only requested env material returns to the agent
```

Important distinctions:

- Cursor OIDC authenticates the **Cloud Agent workload**, not an individual subprocess. It is one workload-identity adapter used by the reference Consumer/Broker dogfood—not a mandatory PADE identity mechanism.
- Any process able to reach Cursor’s local identity socket can mint an identity token for that workload.
- The security boundary for capability resolution therefore lives at **broker authorization**.
- A capability name in `pade.yaml` is a **request**, not authorization. The broker must not trust capability names merely because a client asks or a repo declares them.
- For single-repo confinement, require complete `repo_urls` attestation. Missing `repo_urls` means unknown, not single-repo. Do not authorize from `repo_url` alone. Managed Cloud Agents have been observed with `repo_url` but without `repo_urls`; until complete attestation exists, broker dogfood uses subject + capability (`requireRepoURLs: false`) rather than weakening policy to trust `repo_url`.
- Broker logs must contain identity/capability decision metadata only — never JWTs or resolved credentials.
- JTI replay tracking is deferred; this spike relies on short-lived tokens and exact audience binding.
- The PADE contract still does not replace resource-level authorization (GitHub, IAM, databases, etc.).

### Broker transport modes

`pade-broker` supports three listener models. Defaults stay fail-closed. The Broker Specification requires protected transport for non-local use at the abstract level; these modes are the **reference** deployment shapes.

**1. Local development (loopback)**

Plain HTTP on a loopback bind (`127.0.0.1`, `localhost`, `::1`) is allowed. No TLS files required.

**2. Broker-managed TLS (direct exposure)**

If the broker process is directly reachable on a non-loopback interface, `pade-broker` must terminate TLS itself:

```bash
./bin/pade-broker -listen 0.0.0.0:8787 -tls-cert … -tls-key … -policy … -bindings …
```

**3. Trusted upstream TLS termination**

`pade-broker` may serve plaintext HTTP on a non-loopback interface **only** when the operator explicitly opts in with `-tls-termination=proxy` and ensures the plaintext listener is reachable only inside the trusted deployment boundary (for example Cloud Run’s internal container network, Kubernetes behind an HTTPS ingress, or an HTTPS load balancer):

```text
Internet
   |
 HTTPS
   v
trusted reverse proxy / managed platform
   |
 internal platform transport
   v
pade-broker (plaintext on 0.0.0.0:$PORT)
```

```bash
# Prefer explicit -listen, or omit it and set PORT (container platforms).
# PORT alone does not enable proxy TLS mode — still pass -tls-termination=proxy.
./bin/pade-broker \
  -tls-termination=proxy \
  -policy … -bindings …
# with PORT=8080 → listens on 0.0.0.0:8080
```

`-tls-termination=proxy` is a **deployment assertion**, not cryptographic verification by `pade-broker`. The reference Broker cannot prove that Cloud Run, Kubernetes, or a load balancer was configured correctly. Do not treat arbitrary plaintext non-loopback deployment as safe. Cloud Run is a deployment example, not part of the Broker Specification.

TLS termination does **not** replace Cursor OIDC, broker policy, Keeper authorization, or downstream API authorization. Non-loopback plaintext without `-tls-termination=proxy` or broker-managed cert/key is still rejected.

### Container image

The repo-root `Dockerfile` builds a minimal `pade-broker` image (`gcr.io/distroless/static-debian12:nonroot`):

- The image contains only the broker binary (and base CA certificates). It does **not** contain Go toolchains, shells, curl, source, policy, bindings, or secrets.
- `KSM_CONFIG`, Cursor tokens, and deployment policy/bindings are **runtime** inputs (env / mounts / secret stores). Never bake them into the image or Docker build args.
- The process runs as the distroless `nonroot` user. Non-root reduces container privileges; it is **not** a security sandbox.
- Trusted-upstream-TLS mode (`-tls-termination=proxy` + `PORT`) assumes the operator has established a secure deployment boundary.
- Cursor OIDC and broker authorization remain required regardless of container transport.
- No Docker `HEALTHCHECK`: distroless has no HTTP client; platforms should probe `/healthz`.

## Scope

PADE does not replace OAuth, OIDC, IAM, SPIFFE, or resource-level authorization. Downstream systems remain authoritative.
