# Reference providers

In-tree providers exist for **dogfooding and illustrating** the draft external provider contract ([docs/provider-contract.md](../../docs/provider-contract.md)).

They are **non-normative** and architecturally removable. Their presence does **not** make any vendor, API, or capability part of the PADE standard.

Prefer this `examples/providers/` layout over an `extensions/` name (avoids collision with CNCF Runtime Conditions terminology).

| Path | Role |
|------|------|
| [stub/](stub/) | Minimal contract dogfood |
| [github/](github/) | First reference provider (GitHub App → installation token; fake mode for CI) |
| `google-analytics/` | Planned second reference provider (not created yet) |

Bindings use `provider: exec` with opaque `exec.config`. See ROADMAP Milestones B–G.
