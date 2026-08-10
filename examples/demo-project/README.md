# PADE demo-project

Minimal repository used to dogfood the DevPod-first PADE flow:

1. **DevPod** owns workspace lifecycle (`devpod up` / `devpod stop`).
2. **PADE** owns capability declaration, binding probes, and process-scoped `exec`.
3. This repo never embeds secrets in `pade.yaml`.

Demo capability: **`github.user.read`** (env: **`GITHUB_TOKEN`**).

## Layout

| Path | Role |
|------|------|
| `pade.yaml` | Portable capability declaration (+ optional Dev Container pointer) |
| `.devcontainer/devcontainer.json` | Environment definition for DevPod / Dev Containers |
| `bindings.example.yaml` | Example *local* env bindings (copy; do not commit secrets) |
| `bindings.vault.example.yaml` | Example Vault `-dev` bindings (prototype only) |
| `bindings.onepassword.example.yaml` | Example 1Password bindings (fake-op CI path) |
| `identities/` | Alice/Bob env, Vault, and 1Password binding fixtures |
| `scripts/github-whoami` | Calls GitHub `/user` (or stub for `pade-demo-*` tokens) |

## Easiest path (from repo root)

```bash
make dogfood                 # stub token injection (CI-friendly)
make dogfood-identity
make dogfood-vault
make dogfood-onepassword     # fake-op shim
make dogfood-github-live     # local only: real op + real GitHub API
```

See [docs/onepassword-dogfood.md](../../docs/onepassword-dogfood.md) for storing a real PAT in 1Password.

## DevPod

```bash
make dogfood-devpod
```

Details: [docs/devpod-dogfood.md](../../docs/devpod-dogfood.md).
