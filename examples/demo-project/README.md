# PADE demo-project (Milestone 4 dogfood)

Minimal repository used to dogfood the DevPod-first PADE flow:

1. **DevPod** owns workspace lifecycle (`devpod up` / `devpod stop`).
2. **PADE** owns capability declaration, binding probes, and process-scoped `exec`.
3. This repo never embeds secrets in `pade.yaml`.

## Layout

| Path | Role |
|------|------|
| `pade.yaml` | Portable capability declaration (+ optional Dev Container pointer) |
| `.devcontainer/devcontainer.json` | Environment definition for DevPod / Dev Containers |
| `bindings.example.yaml` | Example *local* bindings (copy; do not commit secrets) |
| `scripts/ga-summary` | Demo command that requires `google-analytics.read` |

## Prerequisites

- Go 1.22+ (to build `pade`), or a built `pade` binary on `PATH`
- [DevPod](https://devpod.sh/) for workspace lifecycle (optional for the PADE-only smoke below)
- Docker (or another DevPod provider) when using DevPod

## Easiest path (from repo root)

PADE-only smoke (no DevPod):

```bash
make dogfood
```

Full DevPod proof (Docker running + `devpod` on PATH):

```bash
make dogfood-devpod
```

GitHub Actions runs the same proof in [`.github/workflows/devpod-dogfood.yml`](../../.github/workflows/devpod-dogfood.yml) (separate from the fast main CI).

That will:

1. Ensure the docker provider is configured
2. `devpod up` this project
3. Cross-compile a **linux** `pade` for your Docker VM arch
4. Install it into the workspace with `docker cp` (fast; avoids SSH/base64 hangs)
5. Run `validate` + `exec` inside the workspace via `docker exec`

Step-by-step equivalents:

```bash
make dogfood-devpod-check
make dogfood-devpod-provider
make dogfood-devpod-up
make dogfood-devpod-install
make dogfood-devpod-smoke
make dogfood-devpod-down      # optional
make dogfood-devpod-delete    # optional
```

## PADE-only smoke (manual)

From the repository root:

```bash
export PATH="$(pwd)/.tools/go/bin:$PATH"   # if using the local toolchain
make dogfood
```

## DevPod dogfood (manual)

```bash
# from examples/demo-project
devpod provider add docker   # once
devpod provider use docker
devpod up . --ide none
```

Install a linux `pade` from the host (do **not** copy a macOS binary):

```bash
# from repo root
make dogfood-devpod-install
```

Or SSH in after install:

```bash
devpod ssh demo-project
export PATH="$HOME/bin:$PATH"
pade validate
GA_PROPERTY_ID=demo-property GOOGLE_APPLICATION_CREDENTIALS=/tmp/x \
  pade exec --bindings bindings.example.yaml \
  --capability google-analytics.read -- ./scripts/ga-summary
```

Stop / delete with DevPod (not PADE):

```bash
make dogfood-devpod-down
make dogfood-devpod-delete
```

## What success looks like

- Same `pade.yaml` + `devcontainer.json` on laptop and remote DevPod workspace
- Capability binding configured outside the portable manifest
- `pade exec --capability …` injects authority only into the child process
- DevPod remains the runtime owner; PADE never shells out to `devpod up` except via optional Make helpers that call the `devpod` CLI directly
