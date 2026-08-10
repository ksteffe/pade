# Security

## Reporting

If you discover a security issue in PADE or its examples, please report it privately to the repository maintainers (GitHub Security Advisories preferred when available) rather than opening a public issue with exploit detail.

## Prototype invariants

Even for local demos (including Vault `-dev` mode):

- Never put secret values in `pade.yaml` or other committed config.
- Never print secret values in `pade plan`, status, logs, or errors.
- Never persist secrets in `.pade/` workspace state.
- Never bake resolved credentials into images or snapshots.
- Treat Vault/1Password/env/Keeper/KSM adapters as **bindings**, not as part of the portable specification.

Local Vault development servers and root tokens are for proving seams only. They are not production-safe and must not be documented as recommended production practice.

## Process-scoped execution

`pade exec` injects resolved material only into the child process. After exit, PADE clears its in-memory material maps.

Best-effort behaviors (defense in depth, **not** security boundaries):

- Exact-match redaction of resolved secret values on child stdout/stderr before they reach the caller (helps cloud-agent transcripts). Encoded, transformed, hashed, or substring-altered values are not recognized. A child that possesses a credential can still intentionally exfiltrate it.
- Providers may omit ambient bootstrap env keys from the child (for example `keeper-secrets-manager` omits `KSM_CONFIG`) after resolution.

A process that has been given a credential can still observe, transform, encode, or exfiltrate it. Any process that can read ambient bootstrap credentials (for example `KSM_CONFIG`) can also call the secret manager directly, bypassing PADE.

## Scope

PADE does not replace OAuth, OIDC, IAM, SPIFFE, or resource-level authorization. Downstream systems remain authoritative.
