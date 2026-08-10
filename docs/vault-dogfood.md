# Vault dogfood (local `-dev`)

Prove that PADE can resolve declared capabilities from HashiCorp Vault without embedding secrets in `pade.yaml`, and that Alice/Bob identity paths resolve distinct material for the same capability name.

**Prototype only.** Vault `-dev` and root tokens are for seam validation. They are not production-safe.

## Ownership

| Concern | Owner |
|---------|--------|
| Portable capability names | `pade.yaml` |
| Vault paths / field maps | Local bindings (`--bindings`) |
| Secret values | Vault KV (never in repo / plan / capabilities output) |
| Workspace lifecycle | DevPod (unchanged) |

## Quick run

From the repository root:

```bash
make dogfood-vault
```

That will:

1. Download Vault into `.tools/vault` if needed (or use `vault` / `VAULT_BIN` on `PATH`)
2. Start `vault server -dev` on `http://127.0.0.1:8200` with token `pade-dev` (or reuse a matching server)
3. Seed shared + Alice/Bob KV secrets
4. Run `pade plan` / `capabilities` / `exec` with Vault bindings and **no ambient `GA_*` env**
5. Assert plan/capabilities JSON never contains seeded secret substrings

## Bindings

| File | Role |
|------|------|
| [examples/demo-project/bindings.vault.example.yaml](../examples/demo-project/bindings.vault.example.yaml) | Shared Vault path |
| [examples/demo-project/identities/alice.vault.bindings.yaml](../examples/demo-project/identities/alice.vault.bindings.yaml) | Alice KV path |
| [examples/demo-project/identities/bob.vault.bindings.yaml](../examples/demo-project/identities/bob.vault.bindings.yaml) | Bob KV path |
| [spec/examples/bindings.vault.example.yaml](../spec/examples/bindings.vault.example.yaml) | Spec example |

## Manual equivalent

```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=pade-dev
vault server -dev -dev-root-token-id=pade-dev -dev-listen-address=127.0.0.1:8200

vault kv put secret/pade/google-analytics \
  property_id=vault-demo-property \
  credentials_path=/tmp/vault-ga.json

# unset ambient env so resolution must come from Vault
unset GA_PROPERTY_ID GOOGLE_APPLICATION_CREDENTIALS

./bin/pade capabilities -f examples/demo-project/pade.yaml \
  --bindings examples/demo-project/bindings.vault.example.yaml
./bin/pade exec -f examples/demo-project/pade.yaml \
  --bindings examples/demo-project/bindings.vault.example.yaml \
  --capability google-analytics.read \
  -- ./examples/demo-project/scripts/ga-summary
```

## CI

GitHub Actions runs the same script in the main CI job after the env-based identity dogfood. Look for `dogfood-vault: ok` under **Validate example manifests**.

## Out of scope

- Production Vault auth (AppRole, JWT/OIDC, policies)
- Claiming PADE is a secrets manager
- Changing the portable schema for Vault-specific fields
