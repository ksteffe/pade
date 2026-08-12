# Examples

| Path | Purpose |
|------|---------|
| [demo-project](demo-project/) | DevPod-first dogfood (`github.user.read` + env/Vault/1Password + live GitHub API) |
| [ingress-demo](ingress-demo/) | Milestone 8 Teleport Application Access dogfood (tiny Go HTTP app) |

**Planned (not created yet):** `providers/google-analytics/` — non-normative in-tree **reference provider** to dogfood the generic provider contract and broker-side derived credentials before `v0.1.0`. Presence of reference providers does **not** make Google Analytics part of the PADE standard. Prefer `examples/providers/` over an `extensions/` name (avoids CNCF Runtime Conditions terminology collision). See [ROADMAP.md](../ROADMAP.md) Milestones B–E.

Spec-level YAML fixtures live under [spec/examples](../spec/examples/).
