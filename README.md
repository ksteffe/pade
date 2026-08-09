# PADE

**Portable Agent Development Environments**

PADE is an exploratory specification and Go reference CLI for declaring **portable agent workspace capabilities** beside existing development-environment standards (Dev Containers, DevPod), without embedding credentials or coupling to a single AI vendor.

This repository is at **Milestone 0**: documentation, schema stub, and project scaffolding. The Go CLI is not implemented yet.

## Current direction (v0.1)

**DevPod-first, capability-focused.**

Research captured in the RFC and design docs shows DevPod already covers much of the portable workspace/runtime layer (Dev Container consumption, local/remote providers, lifecycle, SSH, port forwarding, prebuilds). PADE should not reimplement that.

The current prototype target is:

1. A repository declares **capabilities** (required authority), not secrets.
2. Developers/organizations bind those capabilities to approved credential managers (Vault, 1Password, env, etc.).
3. **DevPod** (or an equivalent existing runtime) owns workspace creation and lifecycle.
4. PADE validates, plans, and resolves capabilities for scoped executions (for example `pade exec --capability …`).
5. Longer-term: authenticated ephemeral review URLs for executable review with collaborators.

```text
Dev Container spec → DevPod → local/remote workspace
Capability declaration → PADE binding/resolution → credential manager → process-scoped authority
```

## Historical / alternate path

Earlier sections of [DESIGN.md](DESIGN.md) and [docs/go-reference.md](docs/go-reference.md) describe a broader orchestration CLI (`pade up` / Dev Container provider, services, localhost ingress). That path remains documented as a learning artifact and fallback if a concrete gap appears that DevPod cannot cover. Prefer composition over reimplementation.

## What's in this repository

| Path | Purpose |
|------|---------|
| [RFC.md](RFC.md) | Problem statement, architecture, security model, DevPod-first revision |
| [DESIGN.md](DESIGN.md) | v0.1 prototype design, CLI surface, milestones, capability/Vault experiments |
| [docs/go-reference.md](docs/go-reference.md) | Go package/design notes for the reference implementation |
| [AGENTS.md](AGENTS.md) | Guidance for coding agents working in this repo |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute |
| [SECURITY.md](SECURITY.md) | Secret-handling invariants and reporting |
| [LICENSE](LICENSE) | Apache License 2.0 |
| [spec/pade.schema.json](spec/pade.schema.json) | Normative JSON Schema for `pade.yaml` (v0.1 stub) |
| [spec/examples/](spec/examples/) | Example manifests |
| `examples/` | Dogfood apps (added as implementation proceeds) |
| `cmd/`, `internal/` | Go CLI (not present until Milestone 1+) |

## Manifest sketch

Capabilities only — no secrets in the repo:

```yaml
version: "0.1"

capabilities:
  google-analytics.read:
    access: read
  datadog.logs.read:
    access: read
```

Environment construction stays in `.devcontainer/devcontainer.json` and is started with DevPod (for example `devpod up .`). See [spec/examples/web-app.yaml](spec/examples/web-app.yaml) for a fuller illustrative manifest that also shows the earlier environment/services shape for comparison.

## Planned CLI (capability-focused v0.1)

| Command | Role |
|---------|------|
| `pade validate` | Validate `pade.yaml` and referenced config |
| `pade plan` | Side-effect-free resolution preview |
| `pade capabilities` | Show requested capabilities and binding status (never secret values) |
| `pade exec --capability … -- <cmd>` | Run a command with resolved capability injection |

Workspace lifecycle: prefer `devpod up` / `devpod stop` directly.

## Design principles

- Compose Dev Containers, DevPod, and existing IAM/secrets systems; standardize only the missing capability boundary.
- Separate **environment**, **agent behavior**, and **runtime authority**.
- Credentials must never appear in manifests, plan output, `.pade/` state, or normal logs.
- Downstream systems remain authoritative for authorization.
- If an existing tool already owns a concern, PADE should delegate or drop the abstraction.

## Roadmap (high level)

| Milestone | Focus |
|-----------|--------|
| **0** (this branch) | Repo, docs, license, schema stub, examples |
| **1** | `validate` / `plan` against schema |
| **2+** | Capability resolution (env → Vault demo), DevPod-oriented exec flow |
| Later | Authenticated review ingress, second runtime/credential providers, external validation |

Details: [DESIGN.md](DESIGN.md) (including DevPod-first revisions) and [docs/go-reference.md](docs/go-reference.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
