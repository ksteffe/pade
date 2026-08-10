# AGENTS.md

Guidance for coding agents and humans working on PADE.

## Purpose

PADE explores a portable **capability declaration** layer for agent development environments. It sits beside Dev Containers and DevPod. It is not a new container runtime, IAM platform, secrets manager, MCP replacement, or AI agent framework.

## Current v0.1 intent

Prefer the **DevPod-first** direction described in the README and the later sections of [RFC.md](RFC.md) / [DESIGN.md](DESIGN.md):

- DevPod (or equivalent) owns workspace lifecycle.
- PADE owns capability declaration, validation, planning, and credential binding/resolution.
- Do not reimplement Dev Container semantics, port forwarding, SSH, or prebuilds unless a concrete gap is proven.

Earlier `pade up` / Dev Container orchestration designs are historical context, not the default implementation target.

## Architecture boundaries

| Concern | Owner |
|---------|--------|
| Reproducible toolchain/image | Dev Containers (`devcontainer.json`) |
| Workspace lifecycle / providers | DevPod (or peer runtime) |
| What authority is needed | PADE capability declaration |
| How credentials are stored | User/org credential manager binding |
| Final authorization | Downstream resource (API, IAM, DB, etc.) |
| How to use a capability | Repo skills/scripts/MCP — not PADE core |

## Design principles

1. Compose mature tools; do not reimplement them.
2. The specification is the product boundary; the Go CLI is a reference implementation.
3. Environment, agent behavior, and runtime authority stay separate.
4. Providers own provider-specific behavior; keep adapters outside the portable core.
5. Secure defaults should be easier than insecure workarounds.
6. Optimize early code for learning and replaceability, not feature completeness.

Before adding a feature, ask:

1. Does an existing standard already represent this?
2. Does an existing CLI or API already implement this?
3. Can PADE delegate to it?
4. Is the remaining gap actually required for portability?

## Secrets and logging (non-negotiable)

- Secret values MUST NOT appear in `pade.yaml`, plan output, `.pade/` state, normal logs, or error messages.
- Workspace images/snapshots MUST NOT contain resolved credentials.
- Prefer distinct types for capability metadata vs credential material to reduce accidental logging.
- Document that code inside a workspace that has been given secrets can still observe them; PADE does not claim otherwise.

## Working in this repo

- Plan before changing cross-cutting interfaces or the schema.
- When behavior changes, update `spec/`, examples, and design docs in the same change when practical.
- Keep v0.1 schemas small; every field creates compatibility pressure.
- Prefer clear package boundaries over speculative abstractions.

## Tests and commands

Requires Go 1.22+. If `go version` shows 1.13 (or similar), put a current SDK first on `PATH` (for example `$(pwd)/.tools/go/bin`) or run `make test`.

```bash
export PATH="$(pwd)/.tools/go/bin:$PATH"
go test ./...
go run ./cmd/pade validate -f spec/examples/web-app.yaml
go run ./cmd/pade plan -f spec/examples/web-app.yaml --json
go run ./cmd/pade capabilities -f spec/examples/web-app.yaml --bindings spec/examples/bindings.example.yaml
GA_PROPERTY_ID=demo GOOGLE_APPLICATION_CREDENTIALS=/tmp/x \
  go run ./cmd/pade exec -f spec/examples/web-app.yaml \
  --bindings spec/examples/bindings.example.yaml \
  --capability google-analytics.read -- /bin/sh -c 'test -n "$GA_PROPERTY_ID" && echo ok'
make ci   # local mirror of GitHub Actions
make dogfood   # examples/demo-project PADE smoke (no DevPod required)
```

DevPod lifecycle is documented in [docs/devpod-dogfood.md](docs/devpod-dogfood.md) and [examples/demo-project/README.md](examples/demo-project/README.md). Do not add a PADE wrapper that reimplements `devpod up`. Full DevPod proof runs locally via `make dogfood-devpod` and in CI via [`.github/workflows/devpod-dogfood.yml`](.github/workflows/devpod-dogfood.yml) (separate from the fast main CI).

Treat [spec/pade.schema.json](spec/pade.schema.json) as the machine-readable contract. Update examples when the schema changes.

## Adding a provider later

Capability and runtime providers should be adapters behind small interfaces. Core code must not import Vault/Google/Cursor-specific SDKs directly into portable packages. See [docs/go-reference.md](docs/go-reference.md).
