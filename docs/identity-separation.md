# Identity separation (Milestone 5)

Prove that the same repository, `pade.yaml`, and development image can be used by two developers (or simulated identities) while each execution resolves **different** credential material — without changing the portable capability declaration.

## Ownership

| Concern | Owner |
|---------|--------|
| Portable capability names | `pade.yaml` (repo) |
| How *this* identity resolves a capability | Local bindings (`--bindings` / `.pade/bindings.yaml`) |
| Secret values | Credential manager / ambient env (never in repo) |
| Workspace lifecycle | DevPod (unchanged from Milestone 4) |

PADE does not become an identity provider. It only keeps capability declaration separate from per-developer binding and resolution.

## Acceptance criteria

1. One shared `pade.yaml` declares `google-analytics.read` (no secrets).
2. Two binding configs (Alice / Bob) map that capability to resolution paths appropriate to each identity.
3. `pade exec --capability google-analytics.read` under Alice’s ambient credentials yields Alice’s values; Bob’s yield Bob’s.
4. Plan / capabilities / injection notices never print secret values (names and providers only).
5. The portable repo files (`pade.yaml`, Dev Container, skills/scripts) are identical for both identities.

## Dogfood (local, no Vault required)

From the repository root:

```bash
make dogfood-identity
```

This runs [scripts/identity-dogfood.sh](../scripts/identity-dogfood.sh): same `examples/demo-project/pade.yaml`, Alice then Bob bindings + ambient env, child process asserts the expected identity label without echoing secret material into PADE logs.

Optional Vault simulation (local `-dev` only): use the commented vault paths in the identity binding examples so Alice and Bob point at different KV paths for the same capability name.

## Out of scope for M5

- Real IdP / OAuth / workload identity exchange
- Organization-wide binding distribution
- Changing the v0.1 schema for an `identity` field (ambient + bindings are enough for the proof)
- Second credential-manager *product* (that is Milestone 6)
