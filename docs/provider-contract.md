# Draft external provider contract (exec binding)

**Status:** Draft / dogfood — Milestone B–C toward D. Not a frozen standard.  
**Related:** [ROADMAP.md](../ROADMAP.md), [spec/broker.md](../spec/broker.md), Go adapters under [`internal/binding`](../internal/binding).

## Purpose

PADE core should understand only:

> An authorized DevelopmentSession requests a capability, and a configured provider fulfills it.

This document describes the **first dogfood binding** for independently implemented providers: **`provider: exec`**. The semantic abstraction is fulfill/derive a capability; exec/subprocess is one implementation binding—not “a generic shell hook” as the product definition.

In-tree reference providers under [`examples/providers/`](../examples/providers/) are **non-normative** and architecturally removable. Their presence does not make GitHub, Google Analytics, or any other vendor part of the PADE standard.

## Binding configuration (reference Consumer / Broker)

Local/admin bindings (never portable Intent):

```yaml
version: "0.1"
capabilities:
  demo.derived:
    provider: exec
    exec:
      command: ["./bin/pade-provider-stub"]
      # Opaque to PADE core — provider-defined keys only:
      config:
        tokenEnv: "DEMO_TOKEN"
        value: "stub-derived-token"
```

| Field | Meaning |
|-------|---------|
| `exec.command` | Argv to invoke (required). First element is the executable. |
| `exec.dir` | Optional working directory for the process. |
| `exec.config` | Optional opaque object. PADE core must not interpret vendor-specific keys. |

Durable authority (private keys, bootstrap secrets) stays on the broker/host side via ambient env, files, or secret-store refs that the **provider** understands. PADE core does not grow fields such as `githubInstallationId` or `googleServiceAccount` as protocol semantics.

## Process protocol

The reference Consumer/Broker invokes `command` with:

- **stdin:** one JSON object (`Request`)
- **stdout:** one JSON object (`Response`) — must not include unrelated chatter
- **stderr:** optional human diagnostics (must not include secret values)
- **exit code:** `0` on success; non-zero on failure

Environment: the child inherits the parent environment (so broker-side durable credentials can be ambient). Resolved Material is returned to PADE; bootstrap secrets should not be copied into Material unless that is the intended deliverable.

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

### Probe response

```json
{
  "status": "available",
  "message": "optional safe text",
  "meta": { "key": "safe-metadata-only" }
}
```

`status` is one of `available`, `unavailable`, `error`. Never put secret values in `message` or `meta`.

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
| `env` | Process environment entries to inject as `Material` (required for dogfood; may be empty only if a future mediation mode is defined) |
| `expiresAt` | Optional RFC3339 timestamp for derived credential lifetime |

## Relationship to existing Go `Provider` interface

[`internal/binding.Provider`](../internal/binding/provider.go) remains the in-process adapter surface for env/Vault/op/keeper/ksm/broker/**exec**. Third-party providers need not be written in Go; they speak this stdin/stdout JSON contract when invoked via `provider: exec`.

## Toward GitHub App / Google Analytics

- [`examples/providers/stub`](../examples/providers/stub) — contract dogfood only
- [`examples/providers/github`](../examples/providers/github) — first reference provider (fake mode now; real App derivation next)
- [`examples/providers/google-analytics`](../examples/providers/google-analytics) — planned second reference provider

If a second vendor forces new **normative** core fields, revisit this contract rather than leaking vendor semantics into PADE.
