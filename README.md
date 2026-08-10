# PADE

**Portable Agent Development Environments**

PADE is an exploratory specification and Go reference CLI for declaring **portable agent workspace capabilities** beside existing development-environment standards (Dev Containers, DevPod), without embedding credentials or coupling to a single AI vendor.

This repository is at **Milestone 1**: `pade validate` and `pade plan` against the v0.1 schema. Capability binding and scoped `exec` come next.

## Quick start

Requires **Go 1.22+**. macOS Homebrew `go` 1.13 will fail with errors like `cannot load embed` — that toolchain predates the `embed` standard library.

If you already have a local SDK under `.tools/go` (or after installing one from https://go.dev/dl/):

```bash
export PATH="$(pwd)/.tools/go/bin:$PATH"
go version   # expect go1.22+
go test ./...
go run ./cmd/pade validate -f spec/examples/web-app.yaml
go run ./cmd/pade plan -f spec/examples/web-app.yaml
go run ./cmd/pade plan -f spec/examples/web-app.yaml --json
```

Or use Make (auto-selects `.tools/go` when present):

```bash
make test
make validate
make plan
make ci          # local mirror of GitHub Actions checks
```

CI runs on pushes to `main` and on pull requests via [`.github/workflows/ci.yml`](.github/workflows/ci.yml) (`gofmt`, `go vet`, tests, build, example validate/plan).

Dependabot keeps Go modules and GitHub Actions on a weekly cadence via [`.github/dependabot.yml`](.github/dependabot.yml).

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
| [cmd/pade](cmd/pade) | CLI entrypoint (`validate`, `plan`) |
| [internal/manifest](internal/manifest) | Load + schema/semantic validation |
| [internal/planner](internal/planner) | Side-effect-free plan model |
| [internal/output](internal/output) | Human and JSON rendering |

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

Environment construction stays in `.devcontainer/devcontainer.json` and is started with DevPod (for example `devpod up .`). See [spec/examples/web-app.yaml](spec/examples/web-app.yaml) for the capability-first shape and [spec/examples/web-app-orchestrated.yaml](spec/examples/web-app-orchestrated.yaml) for the earlier environment/services shape.

## CLI (capability-focused v0.1)

| Command | Status | Role |
|---------|--------|------|
| `pade validate` | Implemented | Validate `pade.yaml` and referenced config |
| `pade plan` | Implemented | Side-effect-free plan (never prints secrets) |
| `pade capabilities` | Planned | Show requested capabilities and binding status |
| `pade exec --capability … -- <cmd>` | Planned | Run a command with resolved capability injection |

Flags: `-f` / `--file` for manifest path; `--json` for machine-readable output.

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
| **0** | Repo, docs, license, schema stub, examples |
| **1** (current) | `validate` / `plan` against schema |
| **2+** | Capability resolution (env → Vault demo), DevPod-oriented exec flow |
| Later | Authenticated review ingress, second runtime/credential providers, external validation |

Details: [DESIGN.md](DESIGN.md) (including DevPod-first revisions) and [docs/go-reference.md](docs/go-reference.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
