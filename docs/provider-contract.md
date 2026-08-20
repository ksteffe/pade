# External provider contract (semantic + exec adapter)

**Status:** Landed for dogfood (Milestones B–G); draft until Milestone H spec tighten / `v0.1.0`. Exec is an **experimental binding**, not the portable PADE standard.  
**Related:** [ROADMAP.md](../ROADMAP.md), [spec/broker.md](../spec/broker.md), Go adapters under [`internal/binding`](../internal/binding).

PADE core should understand only:

> An authorized DevelopmentSession requests a capability, and a configured provider fulfills it.

This document separates two layers that must not be collapsed:

1. **Semantic provider contract** — what fulfillment means (portable idea).
2. **Exec adapter** — one **broker-side** reference implementation mechanism (not normative PADE).

In-tree reference providers under [`examples/providers/`](../examples/providers/) are **non-normative** and architecturally removable. Their presence does not make GitHub, Google Analytics, or any other vendor part of the PADE standard.

## Semantic provider contract

A provider is a **fulfillment abstraction**:

> Given an authorized capability request and trusted broker-side context, produce bounded, validated Material or a structured failure.

Independent of transport or process mechanics, a provider:

| Concern | Expectation |
|---------|-------------|
| Input | Authorized capability identity; broker/operator context (not client-chosen executables) |
| Success | Bounded Material (env map for current dogfood) plus optional expiration metadata |
| Failure | Structured failure without secret leakage |
| Trust | Provider **code/config** is operator-trusted; provider **output** is untrusted until validated |

Portable `DevelopmentSession` / `pade.yaml` must not encode executable paths, argv, shell fragments, bootstrap credentials, or adapter choice. Those belong to broker/operator configuration.

Future implementations MAY fulfill the same semantic contract without subprocesses (for example in-process adapters, mature credential brokers, or workload-identity systems). Do not treat stdin/stdout as the definition of “provider.”

## Exec adapter (broker-side reference only)

**`provider: exec`** is a reference **Broker** materialization adapter: invoke an operator-installed process that speaks a JSON stdin/stdout protocol.

| Rule | Meaning |
|------|---------|
| Broker-side only | Consumer / workspace / user / `PADE_BINDINGS` / `--bindings` must not select `provider: exec` |
| Operator configured | Executable path and `exec.config` live in **server-side** broker bindings |
| Trusted executable | A broker administrator configuring exec is installing trusted credential/authorization plugin code |
| Untrusted output | Stdout/stderr are bounded and validated; raw stderr is not surfaced in user-facing errors |
| Not portable Intent | Never put command paths or bootstrap secrets in `pade.yaml` |
| Not normative | Subprocess invocation is experimental scaffolding—not the portable PADE standard |

### Server-side binding example

```yaml
# broker host bindings (operator-owned) — NOT Consumer/workspace bindings
version: "0.1"
capabilities:
  demo.derived:
    provider: exec
    exec:
      command: ["./bin/pade-provider-stub"]
      config:
        tokenEnv: "DEMO_TOKEN"
        value: "stub-derived-token"
```

Consumer-side bindings for the same capability use `provider: broker` (endpoint + audience only).

| Field | Meaning |
|-------|---------|
| `exec.command` | Argv to invoke (required). First element is the executable. No shell interpolation. |
| `exec.dir` | Optional working directory for the process. |
| `exec.config` | Optional opaque object. PADE core must not interpret vendor-specific keys. |

Durable authority (private keys, bootstrap secrets) stays on the broker/host side via ambient env, files, or secret-store refs that the **provider process** understands.

### Process protocol (exec adapter only)

The reference Broker invokes `command` with:

- **stdin:** one JSON object (`Request`)
- **stdout:** one JSON object (`Response`) — must not include unrelated chatter
- **stderr:** optional human diagnostics (must not include secret values)
- **exit code:** `0` on success; non-zero on failure
- **timeout:** process cancelled via context/`CommandContext`
- **environment:** deliberate allowlist (PATH/HOME and documented ambient auth prefixes such as `VAULT_*`, `OP_*`, `KSM_*`, `PADE_*`)—not the full caller environment
- **I/O bounds:** stdout and stderr each capped (reference: 1 MiB); oversized output fails closed

### Request

```json
{
  "capability": "demo.derived",
  "operation": "resolve",
  "config": { }
}
```

| Field | Values |
|-------|--------|
| `capability` | Portable capability id from the resolve request |
| `operation` | `probe` or `resolve` |
| `config` | Opaque object from `exec.config` (may be omitted/empty) |

Clients cannot supply `command`, argv, or executable paths on the broker resolve request.

### Probe response

```json
{
  "status": "available",
  "message": "optional safe text",
  "meta": { "key": "safe-metadata-only" }
}
```

`status` is one of `available`, `unavailable`, `error`. Never put secret values in `message` or `meta`.

Note: `pade plan` / `pade capabilities` do **not** probe providers (static inspection only).

### Resolve response

```json
{
  "env": {
    "DEMO_TOKEN": "…"
  },
  "expiresAt": "2026-08-12T22:00:00Z"
}
```

| Field | Meaning |
|-------|---------|
| `env` | Process environment entries to inject as `Material` |
| `expiresAt` | Optional RFC3339 timestamp for derived credential lifetime |

## Relationship to existing Go `Provider` interface

[`internal/binding.Provider`](../internal/binding/provider.go) remains the in-process adapter surface. Reference registries:

- Consumer: env / Vault / op / keeper / ksm / **broker** (no exec) — [`internal/providerset`](../internal/providerset)
- Broker: env / Vault / op / keeper / ksm / **exec** (no nested broker)

Third-party providers need not be written in Go; when using the exec adapter they speak this stdin/stdout JSON contract on the **broker host**.

## Reference providers (architectural tests)

In-tree providers under [`examples/providers/`](../examples/providers/) are **non-normative architectural tests**—not integration goals. Full rationale: [ROADMAP.md — Why two derived-token providers before v0.1.0](../ROADMAP.md#why-two-derived-token-providers-before-v010).

Dogfood runs through the broker:

| Target | Identity | Provider material | CI |
|--------|----------|-------------------|-----|
| `make dogfood-exec-provider` (stub) | Fake JWT | Fake stub token | yes |
| `make dogfood-exec-provider-github` | Fake JWT | Fake installation token + repo-meta | yes |
| `make dogfood-exec-provider-ga` | Fake JWT | Fake access token + property-meta | yes |
| `make dogfood-exec-provider-two` | Fake JWT | Both providers, one seam | yes |
| `make dogfood-broker-stage-b-exec` | Real Cursor OIDC | Fake providers by default (`PADE_PROVIDER_FAKE=1`) | no (Cloud Agent) |
| External broker E2E | Real Cursor OIDC | Real derived tokens | no (private deploy; ROADMAP J/K) |

```text
DevelopmentSession → Consumer (provider: broker) → Broker → exec adapter → trusted provider binary → Material
```

**Wire note:** Exec providers may return `expiresAt` in their subprocess JSON. The reference broker HTTP `/v1/resolve` response exposes **`env` only today**—expiry is not yet on the Consumer-visible wire. See [spec/broker.md](../spec/broker.md#resolve-response-success).

Milestone G proves GitHub + Google on the same **semantic** seam without vendor fields in PADE core. If a future vendor forces new **normative** core fields, revisit this contract rather than leaking vendor semantics into PADE.
