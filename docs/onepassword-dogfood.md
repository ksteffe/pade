# 1Password dogfood (Milestone 6)

Prove that the same portable `pade.yaml` can resolve capabilities through a **second credential-manager product** (1Password) without changing the repository capability declaration.

PADE shells out to the 1Password CLI (`op read`). For CI and local smoke without a real 1Password account, `PADE_OP_BIN` can point at [scripts/fake-op.sh](../scripts/fake-op.sh).

## Ownership

| Concern | Owner |
|---------|--------|
| Portable capability names | `pade.yaml` |
| `op://` references | Local bindings (`--bindings`) |
| Secret values | 1Password (or dogfood shim) |
| Workspace lifecycle | DevPod (unchanged) |

## Quick run

```bash
make dogfood-onepassword
```

That sets `PADE_OP_BIN=scripts/fake-op.sh`, runs plan/capabilities/exec for shared + Alice/Bob bindings, and asserts plan/capabilities JSON never contain seeded secret substrings.

## Bindings

```yaml
provider: onepassword
onepassword:
  refs:
    GA_PROPERTY_ID: "op://pade-demo/google-analytics/property_id"
    GOOGLE_APPLICATION_CREDENTIALS: "op://pade-demo/google-analytics/credentials_path"
```

| File | Role |
|------|------|
| [bindings.onepassword.example.yaml](../examples/demo-project/bindings.onepassword.example.yaml) | Shared refs |
| [alice.onepassword.bindings.yaml](../examples/demo-project/identities/alice.onepassword.bindings.yaml) | Alice refs |
| [bob.onepassword.bindings.yaml](../examples/demo-project/identities/bob.onepassword.bindings.yaml) | Bob refs |

## Real 1Password CLI

```bash
# install: https://developer.1password.com/docs/cli/
export PADE_OP_BIN=op   # or leave unset
op signin
# point bindings at your vault/item/field refs, then:
./bin/pade exec -f examples/demo-project/pade.yaml \
  --bindings /path/to/your.bindings.yaml \
  --capability google-analytics.read -- ./examples/demo-project/scripts/ga-summary
```

## CI

GitHub Actions runs this under the **Smoke** job in a dedicated **1Password dogfood** step. Look for `dogfood-onepassword: ok`.

## Out of scope

- Embedding the 1Password SDK in portable packages (CLI adapter only)
- Service-account / Connect server production auth
- Claiming PADE replaces 1Password
