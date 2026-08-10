# Identity separation (Milestone 5)

Prove that the same repository, `pade.yaml`, and development image can be used by two developers (or simulated identities) while each execution resolves **different** credential material — without changing the portable capability declaration.

Demo capability: **`github.user.read`** → **`GITHUB_TOKEN`**.

## Ownership

| Concern | Owner |
|---------|--------|
| Portable capability names | `pade.yaml` (repo) |
| How *this* identity resolves a capability | Local bindings (`--bindings` / `.pade/bindings.yaml`) |
| Secret values | Credential manager / ambient env (never in repo) |
| Workspace lifecycle | DevPod (unchanged from Milestone 4) |

## Acceptance criteria

1. One shared `pade.yaml` declares `github.user.read` (no secrets).
2. Two binding configs (Alice / Bob) map that capability to resolution paths appropriate to each identity.
3. `pade exec --capability github.user.read` under Alice’s ambient credentials yields Alice’s values; Bob’s yield Bob’s.
4. Plan / capabilities / injection notices never print secret values (names and providers only).
5. The portable repo files (`pade.yaml`, Dev Container, skills/scripts) are identical for both identities.

## Dogfood

```bash
make dogfood-identity
```

Vault / 1Password / Keeper variants: `make dogfood-vault`, `make dogfood-onepassword`, `make dogfood-keeper`. Realistic GitHub API path (local only): `make dogfood-github-live`.

## Out of scope for M5

- Real IdP / OAuth / workload identity exchange
- Organization-wide binding distribution
- Changing the v0.1 schema for an `identity` field (ambient + bindings are enough for the proof)

Milestone 6 adds a second credential-manager product (1Password); see [onepassword-dogfood.md](onepassword-dogfood.md). Milestone 7 adds Keeper Commander; see [keeper-dogfood.md](keeper-dogfood.md).
