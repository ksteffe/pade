# PADE

**Portable Agent Development Environments**

PADE is an exploratory **interoperability specification** with a Go **reference implementation**. It defines how projects declare portable development **intent**, how **consumers** request capabilities, and how **brokers** authenticate workloads and materialize authorized authority—beside existing environment standards (Dev Containers, DevPod), without embedding credentials or coupling to a single AI vendor.

These specs are **draft / exploratory (v0.1)**. They are not an industry standard, and no vendor is claimed to support PADE today.

## Specification surfaces

| Spec | Role | Document |
|------|------|----------|
| **Intent** | Portable `pade.yaml`: what capabilities a project may need | [spec/intent.md](spec/intent.md) |
| **Consumer** | Software that reads intent and requests/uses capabilities | [spec/consumer.md](spec/consumer.md) |
| **Broker** | Authenticate workload, authorize, materialize | [spec/broker.md](spec/broker.md) |

Entry point: [spec/README.md](spec/README.md).

```text
pade.yaml (Intent)
   |
   v
Consumer
   |
   v
Broker
   |
   v
existing authority / service systems
```

A conforming ecosystem participant should not need to run binaries from this repository. Hypothetical future implementers (for example a Cursor-like Consumer or a Vercel-like Broker) would interoperate through the specifications—not by embedding the Go reference code.

## This repository’s reference implementation

| Piece | Role | Maturity |
|-------|------|----------|
| [`pade`](cmd/pade) | Reference Consumer / CLI | v0.1 capability path implemented |
| [`pade-broker`](cmd/pade-broker) | Reference Broker | **Experimental spike** (Phase 2) |
| [`spec/pade.schema.json`](spec/pade.schema.json) | Intent schema (machine-readable) | v0.1 stub |

Provider adapters (env, Vault, 1Password, Keeper, Keeper Secrets Manager, Cursor OIDC) are **reference implementation** integrations, not automatic parts of the PADE standard.

This repository is at **Milestone 9** with a **Phase 2 broker spike**: Keeper Secrets Manager direct mode plus Cursor OIDC + `pade-broker` so `KSM_CONFIG` can stay off the agent VM. Teleport ingress remains a Milestone 8 spike (not a PADE dependency).

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
make smoke-broker-container  # Docker image smoke: healthz + unauthenticated resolve deny
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

**DevPod-first environment lifecycle; PADE-owned capability interoperability.**

Research captured in the RFC and design docs shows DevPod already covers much of the portable workspace/runtime layer (Dev Container consumption, local/remote providers, lifecycle, SSH, port forwarding, prebuilds). PADE should not reimplement that.

The current target is:

1. A repository declares **Intent** (required capability names), not secrets.
2. A **Consumer** validates/plans and resolves capabilities for scoped executions (reference: `pade exec`).
3. Resolution may use local provider bindings **or** a **Broker** with workload identity (reference spike: `pade-broker` + Cursor OIDC).
4. **DevPod** (or an equivalent existing runtime) owns workspace creation and lifecycle.
5. Longer-term hypotheses (authenticated review URLs, resource/lease-shaped capabilities) remain open—see [spec/README.md](spec/README.md).

```text
Dev Container spec → DevPod → local/remote workspace
Intent (pade.yaml) → Consumer → [bindings | Broker] → authority systems → process-scoped use
```

## Historical / alternate path

Earlier sections of [DESIGN.md](DESIGN.md) and [docs/go-reference.md](docs/go-reference.md) describe a broader orchestration CLI (`pade up` / Dev Container provider, services, localhost ingress). That path remains documented as a learning artifact and fallback if a concrete gap appears that DevPod cannot cover. Prefer composition over reimplementation.

## What's in this repository

| Path | Purpose |
|------|---------|
| [spec/README.md](spec/README.md) | PADE specification entry (Intent / Consumer / Broker) |
| [spec/intent.md](spec/intent.md) | Intent Specification |
| [spec/consumer.md](spec/consumer.md) | Consumer Specification |
| [spec/broker.md](spec/broker.md) | Broker Specification (experimental protocol) |
| [spec/pade.schema.json](spec/pade.schema.json) | Intent JSON Schema (v0.1 stub) |
| [spec/examples/](spec/examples/) | Example manifests, bindings, broker policy |
| [RFC.md](RFC.md) | Problem statement and architectural evolution |
| [DESIGN.md](DESIGN.md) | Reference implementation design history |
| [docs/go-reference.md](docs/go-reference.md) | Go reference Consumer/Broker design notes |
| [AGENTS.md](AGENTS.md) | Guidance for coding agents working in this repo |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute |
| [SECURITY.md](SECURITY.md) | Trust boundaries and secret-handling invariants |
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
| [examples/demo-project](examples/demo-project) | DevPod-first dogfood project (+ `identities/`) |
| [examples/ingress-demo](examples/ingress-demo) | Teleport ingress dogfood (tiny Go HTTP app) |
| [cmd/pade](cmd/pade) | Reference Consumer CLI (`validate`, `plan`, `capabilities`, `exec`, `identity`) |
| [cmd/pade-broker](cmd/pade-broker) | Reference Broker spike (loopback / broker TLS / `-tls-termination=proxy`) |
| [internal/manifest](internal/manifest) | Intent load + schema/semantic validation |
| [internal/binding](internal/binding) | Local bindings + env/vault/onepassword/keeper/keepersm/broker providers |
| [internal/broker](internal/broker) | Broker policy, OIDC verify, resolve API |
| [internal/identity](internal/identity) | Workload token source seam (+ Cursor adapter) |
| [internal/execution](internal/execution) | Process-scoped capability injection + best-effort output redaction |
| [internal/planner](internal/planner) | Side-effect-free plan model |
| [internal/output](internal/output) | Human and JSON rendering |

## Manifest sketch (Intent)

Capabilities only — no secrets in the repo:

```yaml
version: "0.1"

capabilities:
  github.user.read:
    access: read
```

(Schema examples under `spec/examples/` may still show other capability names such as `google-analytics.read`.)

Environment construction stays in `.devcontainer/devcontainer.json` and is started with DevPod (for example `devpod up .`). See [spec/examples/web-app.yaml](spec/examples/web-app.yaml) for the capability-first shape and [spec/examples/web-app-orchestrated.yaml](spec/examples/web-app-orchestrated.yaml) for the earlier environment/services shape.

## Local bindings (reference Consumer)

`pade.yaml` declares capability names only. Bindings are local to the reference Consumer (or broker host), not portable Intent:

- `.pade/bindings.yaml` (gitignored via `.pade/`)
- `~/.config/pade/bindings.yaml`
- `PADE_BINDINGS` or `--bindings`

See [spec/examples/bindings.example.yaml](spec/examples/bindings.example.yaml). Plan/capabilities may show paths and env **names**, never secret values. Vault `-dev` and the 1Password dogfood shim are prototype-only.

## CLI (reference Consumer / Broker)

| Command | Status | Role |
|---------|--------|------|
| `pade validate` | Implemented | Validate Intent (`pade.yaml`) and referenced config |
| `pade plan` | Implemented | Side-effect-free plan including binding status |
| `pade capabilities` | Implemented | Show declared capabilities and binding probes |
| `pade exec --capability … -- <cmd>` | Implemented | Run a command with process-scoped capability injection |
| `pade identity --audience …` | Implemented | Inspect Cursor workload identity (safe claims; no raw JWT) |
| `pade-broker` | Experimental spike | OIDC-verified capability broker (Phase 2) |

Flags: `-f` / `--file`, `--bindings`, `--json` (validate/plan/capabilities), `--capability` / `-c` (exec, repeatable).

Workspace lifecycle: prefer `devpod up` / `devpod stop` directly. See [examples/demo-project](examples/demo-project) and [docs/devpod-dogfood.md](docs/devpod-dogfood.md).

## Design principles

- Compose Dev Containers, DevPod, and existing IAM/secrets systems; standardize only the missing capability interoperability boundary.
- Separate **environment**, **agent behavior**, and **runtime authority**.
- Separate **Intent**, **Consumer**, and **Broker** specification surfaces from the Go reference code.
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
| **9b** (spike) | Cursor OIDC token source + minimal `pade-broker` (server-side policy) — **implemented as experimental reference Broker** |
| **9+** | Further broker dogfood / deployment learning; optional release artifacts |
| Later | External validation of the Intent/Consumer/Broker specs; re-evaluate standalone packaging |

Details: [DESIGN.md](DESIGN.md), [spec/README.md](spec/README.md), and [docs/go-reference.md](docs/go-reference.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
