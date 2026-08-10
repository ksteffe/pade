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
- `pade exec` best-effort redacts exact resolved secret values from child stdout/stderr. This is defense in depth for agent transcripts — **not** a security boundary.

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
make ci-unit   # fast: gofmt, vet, test, build
make ci-smoke  # env/identity/Vault/op/keeper/ksm dogfood + example validate
make ci   # local mirror of GitHub Actions (ci-unit then ci-smoke)
make dogfood   # examples/demo-project PADE smoke (no DevPod required)
make dogfood-identity   # Milestone 5 Alice/Bob identity separation smoke
make dogfood-vault   # Vault -dev capability resolution (+ Alice/Bob KV paths)
make dogfood-onepassword   # Milestone 6 1Password CLI adapter (fake-op shim)
make install-onepassword-cli  # install real `op` for live demos
make dogfood-onepassword-live   # local real 1Password + GitHub API (not CI)
make dogfood-keeper   # Milestone 7 Keeper Commander adapter (fake-keeper shim)
make install-keeper-cli  # install real `keeper` for live demos
make dogfood-keeper-live   # local real Keeper + GitHub API (not CI)
make dogfood-ksm   # Milestone 9 Keeper Secrets Manager (PADE_KSM_FAKE=1)
make dogfood-ksm-live   # local / Cursor Cloud: real KSM + GitHub API (not CI)
make dogfood-broker   # Phase 2 spike: fake OIDC + pade-broker + fake KSM
make dogfood-broker-stage-b   # Stage B: real Cursor OIDC + local broker (Cloud Agent; not CI)
make smoke-broker-container   # Docker pade-broker image smoke (requires docker)
make dogfood-ingress-teleport  # Milestone 8 Teleport Application Access (host; Docker optional)
make dogfood-ingress-teleport-down
```

DevPod lifecycle is documented in [docs/devpod-dogfood.md](docs/devpod-dogfood.md) and [examples/demo-project/README.md](examples/demo-project/README.md). Do not add a PADE wrapper that reimplements `devpod up`. Full DevPod proof runs locally via `make dogfood-devpod` and in CI via [`.github/workflows/devpod-dogfood.yml`](.github/workflows/devpod-dogfood.yml) (separate from the fast main CI). Teleport ingress composition is documented in [docs/teleport-ingress.md](docs/teleport-ingress.md); do not add a PADE wrapper that reimplements Teleport Application Access. Keeper Secrets Manager / Cursor Cloud composition is documented in [docs/keeper-secrets-manager-dogfood.md](docs/keeper-secrets-manager-dogfood.md) and [docs/cursor-cloud-dogfood.md](docs/cursor-cloud-dogfood.md). Phase 2 broker composition is documented in [docs/cursor-oidc-broker-dogfood.md](docs/cursor-oidc-broker-dogfood.md). Do not put Cursor-specific config into portable `pade.yaml`.

Treat [spec/pade.schema.json](spec/pade.schema.json) as the machine-readable contract. Update examples when the schema changes.

## Adding a provider later

Capability and runtime providers should be adapters behind small interfaces. Prefer CLI adapters when practical. The Keeper Secrets Manager adapter (`internal/binding/keepersm`) is an intentional exception that imports Keeper’s official Go SDK in the adapter package only — not in portable core packages. Do not import Vault/Google/Cursor/Keeper SDKs into portable packages. See [docs/go-reference.md](docs/go-reference.md).
