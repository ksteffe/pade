# GitHub App reference provider (skeleton)

**Non-normative.** In-tree for dogfooding the generic [`provider: exec`](../../../docs/provider-contract.md) seam. Removable from PADE core. Does **not** make GitHub part of the PADE standard.

## Intended flow (Milestone D–E)

```text
GitHub App private key  (broker-side only)
        ↓
PADE broker
        ↓
this provider
        ↓
short-lived installation token → DevelopmentSession
        ↓
repository-scoped GitHub operation (not /user whoami)
```

All GitHub-specific behavior (App JWT, installation id, permissions, expiry) belongs **here**, in opaque `exec.config` / ambient broker env—not in PADE Intent or core protocol fields.

## Current status

| Mode | Behavior |
|------|----------|
| `PADE_PROVIDER_FAKE=1` | Returns a fake installation-token-shaped `GITHUB_TOKEN` + `expiresAt` (contract dogfood) |
| unset | Exits with an error pointing at real App derivation (next implementation slice) |

```bash
go build -o ../../../bin/pade-provider-github .
PADE_PROVIDER_FAKE=1 make dogfood-exec-provider-github
```

## Opaque config examples (provider-local)

These keys are **not** PADE core fields; they are only meaningful to this binary:

```yaml
exec:
  command: ["./bin/pade-provider-github"]
  config:
    tokenEnv: GITHUB_TOKEN
    # Future real mode may use paths/refs understood only by this provider:
    # appId: "..."
    # installationId: "..."
    # privateKeyPath: "/run/secrets/github-app.pem"
```
