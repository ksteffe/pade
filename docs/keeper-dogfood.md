# Keeper Commander dogfood (Milestone 7)

Prove that the same portable `pade.yaml` can resolve capabilities through **Keeper Commander** without changing the repository capability declaration.

PADE shells out to Keeper Commander (`keeper get --format=password <UID>`).

The demo capability is **`github.user.read`** → env **`GITHUB_TOKEN`**.

## Ways to try it

| Goal | Command | CI? |
|------|---------|-----|
| Fake-keeper smoke (no account) | `make dogfood-keeper` | Yes (Smoke job) |

Live Keeper login / real GitHub PAT paths are **out of scope** for Milestone 7 (mirror the 1Password fake-op CI path first).

## Fake-keeper smoke (CI)

```bash
make dogfood-keeper
```

Uses `scripts/fake-keeper.sh` and `pade-demo-*` stub tokens (no network call to Keeper or GitHub).

Also exercises Alice/Bob identity separation with distinct `keeper://` UIDs.

## Bindings shape

```yaml
provider: keeper
keeper:
  refs:
    GITHUB_TOKEN: "keeper://RECORD_UID"
```

- Handles only: values must start with `keeper://`.
- v0.1 resolves the **password** field of the record only.

## Override binary

```bash
export PADE_KEEPER_BIN=/path/to/keeper   # or scripts/fake-keeper.sh
```

## Out of scope

- Embedding the Keeper SDK in portable packages (CLI adapter only)
- Live Commander install/login helpers and real GitHub API dogfood
- Non-password custom fields / Keeper Secrets Manager
- Claiming PADE replaces Keeper
