# Examples

| Path | Purpose |
|------|---------|
| [demo-project](demo-project/) | DevPod-first dogfood (`github.user.read` + env/Vault/1Password + live GitHub API) |
| [ingress-demo](ingress-demo/) | Milestone 8 Teleport Application Access dogfood (tiny Go HTTP app) |

**Planned (not created yet):**

- `providers/github/` — non-normative **first** reference provider (GitHub App → short-lived installation token); preferred pre-release derived GitHub dogfood
- `providers/google-analytics/` — non-normative **second** reference provider; proves the contract is not GitHub-specific

Presence of reference providers does **not** make GitHub or Google Analytics part of the PADE standard. Prefer `examples/providers/` over an `extensions/` name (avoids CNCF Runtime Conditions terminology collision). See [ROADMAP.md](../ROADMAP.md) Milestones B–G.

Spec-level YAML fixtures live under [spec/examples](../spec/examples/).
