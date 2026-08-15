# Reference providers

In-tree providers exist for **dogfooding and illustrating** the draft external provider contract ([docs/provider-contract.md](../../docs/provider-contract.md)).

They are **non-normative architectural tests** and architecturally removable. Their presence does **not** make any vendor, API, or capability part of the PADE standard. Two structurally different providers (GitHub App; Google service-account OAuth) prove the broker-side derivation seam before `v0.1.0`—see [ROADMAP.md](../../ROADMAP.md#why-two-derived-token-providers-before-v010).

Prefer this `examples/providers/` layout over an `extensions/` name (avoids collision with CNCF Runtime Conditions terminology).

| Path | Role |
|------|------|
| [stub/](stub/) | Minimal contract dogfood |
| [github/](github/) | First architectural test (GitHub App → installation token; fake mode for CI) |
| [google-analytics/](google-analytics/) | Second structural test (service account → OAuth access token; fake mode for CI; not GA product support) |

Bindings use **broker-side** `provider: exec` with opaque `exec.config`; the Consumer uses `provider: broker`. Same-seam proof: `make dogfood-exec-provider-two`. See ROADMAP Milestones B–G and [SECURITY.md](../../SECURITY.md) (exec is broker-only).
