# Security

## Reporting

If you discover a security issue in PADE or its examples, please report it privately to the repository maintainers (GitHub Security Advisories preferred when available) rather than opening a public issue with exploit detail.

## Prototype invariants

Even for local demos (including Vault `-dev` mode):

- Never put secret values in `pade.yaml` or other committed config.
- Never print secret values in `pade plan`, status, logs, or errors.
- Never persist secrets in `.pade/` workspace state.
- Never bake resolved credentials into images or snapshots.
- Treat Vault/1Password/env adapters as **bindings**, not as part of the portable specification.

Local Vault development servers and root tokens are for proving seams only. They are not production-safe and must not be documented as recommended production practice.

## Scope

PADE does not replace OAuth, OIDC, IAM, SPIFFE, or resource-level authorization. Downstream systems remain authoritative.
