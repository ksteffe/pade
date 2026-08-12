# Examples

| Path | Purpose |
|------|---------|
| [demo-project](demo-project/) | DevPod-first dogfood (`github.user.read` + env/Vault/1Password + live GitHub API) |
| [ingress-demo](ingress-demo/) | Milestone 8 Teleport Application Access dogfood (tiny Go HTTP app) |

| [providers/](providers/) | Non-normative external providers (`provider: exec` contract dogfood) |

See [providers/README.md](providers/README.md) and [docs/provider-contract.md](../docs/provider-contract.md). Presence of reference providers does **not** make GitHub or Google Analytics part of the PADE standard. Prefer `examples/providers/` over an `extensions/` name. See [ROADMAP.md](../ROADMAP.md) Milestones B–G.

Spec-level YAML fixtures live under [spec/examples](../spec/examples/).
