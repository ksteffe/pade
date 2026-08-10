# PADE

**Portable Agent Development Environments**

PADE is an exploratory specification and Go reference CLI for declaring **portable agent workspace capabilities** beside existing development-environment standards (Dev Containers, DevPod), without embedding credentials or coupling to a single AI vendor.

This repository is at **Milestone 9** with a **Phase 2 spike**: Keeper Secrets Manager direct mode plus a minimal Cursor OIDC capability broker (`pade-broker`) that keeps `KSM_CONFIG` off the agent VM. Earlier milestones remain dogfoodable via `make dogfood*` targets. Teleport ingress remains a Milestone 8 spike.

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
go run ./cmd/pade capabilities -f spec/examples/web-app.yaml --bindings spec/examples/bindings.example.yaml
GA_PROPERTY_ID=demo GOOGLE_APPLICATION_CREDENTIALS=/tmp/x \
  go run ./cmd/pade exec -f spec/examples/web-app.yaml \
  --bindings spec/examples/bindings.example.yaml \
  --capability google-analytics.read -- /bin/sh -c 'test -n "$GA_PROPERTY_ID" && echo ok'
```

Or use Make (auto-selects `.tools/go` when present):

```bash
make test
make validate
make plan
make ci          # local mirror of GitHub Actions checks
make dogfood     # PADE smoke against examples/demo-project
make dogfood-identity  # Milestone 5: Alice/Bob bindings against the same pade.yaml
make dogfood-vault     # Vault -dev resolution (+ Alice/Bob KV paths; prototype only)
make dogfood-onepassword  # Milestone 6: 1Password CLI adapter (fake-op shim in CI)
make install-onepassword-cli  # install real `op` (Homebrew or .tools/op)
make dogfood-onepassword-live  # local only: real 1Password + real GitHub API
make dogfood-keeper       # Milestone 7: Keeper Commander adapter (fake-keeper shim in CI)
make install-keeper-cli   # install real `keeper` (Homebrew or .tools/keeper-venv)
make dogfood-keeper-live  # local only: real Keeper + real GitHub API
make dogfood-ksm          # Milestone 9: Keeper Secrets Manager (PADE_KSM_FAKE=1 in CI)
make dogfood-ksm-live     # local / Cursor Cloud: real KSM + real GitHub API
make dogfood-broker       # Phase 2 spike: fake Cursor OIDC + pade-broker + fake KSM
make dogfood-broker-stage-b  # Stage B: real Cursor OIDC + local broker (Cloud Agent only)
make dogfood-ingress-teleport  # Milestone 8 spike: Teleport Application Access (host; Docker optional)
make dogfood-ingress-teleport-down
make dogfood-devpod  # optional: full DevPod proof (needs docker + devpod)
```

CI runs on pushes to `main` and on pull requests via [`.github/workflows/ci.yml`](.github/workflows/ci.yml):

- **Unit tests** — `gofmt`, `go vet`, `go test`, build (fast feedback)
- **Smoke** — example validate/plan/exec, identity dogfood, Vault `-dev` dogfood, 1Password dogfood, Keeper dogfood, KSM dogfood, broker OIDC dogfood (needs the unit job)

A separate [DevPod dogfood](.github/workflows/devpod-dogfood.yml) workflow boots a real DevPod workspace and runs PADE inside it (path-filtered / manual `workflow_dispatch`).

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
| [docs/devpod-dogfood.md](docs/devpod-dogfood.md) | DevPod composition rules for Milestone 4 |
| [docs/identity-separation.md](docs/identity-separation.md) | Milestone 5 identity-separation dogfood |
| [docs/vault-dogfood.md](docs/vault-dogfood.md) | Vault `-dev` capability resolution dogfood |
| [docs/onepassword-dogfood.md](docs/onepassword-dogfood.md) | 1Password provider + `make dogfood-onepassword-live` |
| [docs/keeper-dogfood.md](docs/keeper-dogfood.md) | Keeper Commander provider / Milestone 7 dogfood |
| [docs/keeper-secrets-manager-dogfood.md](docs/keeper-secrets-manager-dogfood.md) | Keeper Secrets Manager provider / Milestone 9 |
| [docs/cursor-cloud-dogfood.md](docs/cursor-cloud-dogfood.md) | Cursor Cloud Agent + KSM composition (vendor-specific) |
| [docs/cursor-oidc-broker-dogfood.md](docs/cursor-oidc-broker-dogfood.md) | Phase 2 Cursor OIDC broker spike |
| [docs/teleport-ingress.md](docs/teleport-ingress.md) | Teleport Application Access / Milestone 8 ingress spike |
| [spec/pade.schema.json](spec/pade.schema.json) | Normative JSON Schema for `pade.yaml` (v0.1 stub) |
| [spec/examples/](spec/examples/) | Example manifests |
| [examples/demo-project](examples/demo-project) | DevPod-first dogfood project (+ `identities/`) |
| [examples/ingress-demo](examples/ingress-demo) | Teleport ingress dogfood (tiny Go HTTP app) |
| [cmd/pade](cmd/pade) | CLI entrypoint (`validate`, `plan`, `capabilities`, `exec`, `identity`) |
| [cmd/pade-broker](cmd/pade-broker) | Phase 2 spike: OIDC-verified capability broker (loopback / broker TLS / `-tls-termination=proxy`) |
| [internal/manifest](internal/manifest) | Load + schema/semantic validation |
| [internal/binding](internal/binding) | Local bindings + env/vault/onepassword/keeper/keepersm/broker providers |
| [internal/broker](internal/broker) | Broker policy, OIDC verify, resolve API |
| [internal/identity](internal/identity) | Workload token source seam (+ Cursor adapter) |
| [internal/execution](internal/execution) | Process-scoped capability injection + best-effort output redaction |
| [internal/planner](internal/planner) | Side-effect-free plan model |
| [internal/output](internal/output) | Human and JSON rendering |

## Manifest sketch

Capabilities only — no secrets in the repo:

```yaml
version: "0.1"

capabilities:
  github.user.read:
    access: read
```

(Schema examples under `spec/examples/` may still show other capability names such as `google-analytics.read`.)

Environment construction stays in `.devcontainer/devcontainer.json` and is started with DevPod (for example `devpod up .`). See [spec/examples/web-app.yaml](spec/examples/web-app.yaml) for the capability-first shape and [spec/examples/web-app-orchestrated.yaml](spec/examples/web-app-orchestrated.yaml) for the earlier environment/services shape.

## Local bindings

`pade.yaml` declares capability names only. Bindings are local:

- `.pade/bindings.yaml` (gitignored via `.pade/`)
- `~/.config/pade/bindings.yaml`
- `PADE_BINDINGS` or `--bindings`

See [spec/examples/bindings.example.yaml](spec/examples/bindings.example.yaml). Plan/capabilities may show paths and env **names**, never secret values. Vault `-dev` and the 1Password dogfood shim are prototype-only.

## CLI (capability-focused v0.1)

| Command | Status | Role |
|---------|--------|------|
| `pade validate` | Implemented | Validate `pade.yaml` and referenced config |
| `pade plan` | Implemented | Side-effect-free plan including binding status |
| `pade capabilities` | Implemented | Show declared capabilities and binding probes |
| `pade exec --capability … -- <cmd>` | Implemented | Run a command with process-scoped capability injection |
| `pade identity --audience …` | Implemented | Inspect Cursor workload identity (safe claims; no raw JWT) |
| `pade-broker` | Spike | OIDC-verified capability broker (Phase 2) |

Flags: `-f` / `--file`, `--bindings`, `--json` (validate/plan/capabilities), `--capability` / `-c` (exec, repeatable).

Workspace lifecycle: prefer `devpod up` / `devpod stop` directly. See [examples/demo-project](examples/demo-project) and [docs/devpod-dogfood.md](docs/devpod-dogfood.md).

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
| **1** | `validate` / `plan` against schema |
| **2** | Local bindings (`env`, `vault`) + `capabilities` |
| **3** | Scoped `pade exec --capability` |
| **4** | DevPod dogfood (`examples/demo-project`) |
| **5** | Identity separation (same repo, distinct credentials) |
| **5b** | Vault `-dev` validation of bindings / identity paths |
| **6** | Second credential provider (1Password CLI adapter) |
| **7** | Keeper Commander CLI binding provider (fake-keeper CI) |
| **8** | Local Teleport authenticated ingress (`examples/ingress-demo`) |
| **9** | Keeper Secrets Manager + Cursor Cloud dogfood (`KSM_CONFIG`, exec redaction) |
| **9b** (spike) | Cursor OIDC token source + minimal `pade-broker` (server-side policy) |
| **9+** | Real Cursor OIDC + hosted/tunneled broker dogfood; optional release artifacts |
| Later | External validation / re-evaluate standalone PADE |

Details: [DESIGN.md](DESIGN.md) (including DevPod-first revisions) and [docs/go-reference.md](docs/go-reference.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
