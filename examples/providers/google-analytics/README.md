# Google Analytics reference provider

**Non-normative.** In-tree for dogfooding the generic [`provider: exec`](../../../docs/provider-contract.md) seam. Removable from PADE core. Does **not** make Google Analytics part of the PADE standard.

## Flow (Milestone F)

```text
Google service account  (broker-side only)
        ↓
this provider (JWT assertion → OAuth2 access token)
        ↓
short-lived access token (+ optional GA_PROPERTY_ID) → DevelopmentSession Material
        ↓
minimal Analytics API operation (property metadata)
```

All Google-specific authentication behavior belongs **here**, in opaque `exec.config` / ambient broker env—not in PADE Intent or core protocol fields.

## Modes

| Mode | Behavior |
|------|----------|
| `PADE_PROVIDER_FAKE=1` | Returns a fake access-token-shaped `GA_ACCESS_TOKEN` + `GA_PROPERTY_ID` + `expiresAt` |
| unset (real) | Mints a service-account JWT and exchanges it at Google's token endpoint |

```bash
go test ./examples/providers/google-analytics/
go build -o ../../../bin/pade-provider-google-analytics .
PADE_PROVIDER_FAKE=1 make dogfood-exec-provider-ga
```

Live Google credentials are **not** required for CI. For a real install:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/run/secrets/ga-sa.json
export GA_PROPERTY_ID=properties/123456789
# unset PADE_PROVIDER_FAKE
pade exec -f pade.yaml --bindings bindings.yaml --capability google-analytics.read -- \
  ./examples/demo-project/scripts/ga-property-meta
```

## Opaque config (provider-local)

These keys are **not** PADE core fields:

| Key | Env fallback | Meaning |
|-----|--------------|---------|
| `serviceAccountFile` | `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account JSON (preferred) |
| `serviceAccountJSON` | `GOOGLE_SERVICE_ACCOUNT_JSON` | Inline SA JSON (fallback) |
| `clientEmail` | `GOOGLE_CLIENT_EMAIL` | SA email when not using JSON file |
| `privateKey` | `GOOGLE_PRIVATE_KEY` | PEM when not using JSON file |
| `tokenURL` | `GOOGLE_TOKEN_URL` | Default `https://oauth2.googleapis.com/token` |
| `scope` | `GOOGLE_OAUTH_SCOPE` | Default `https://www.googleapis.com/auth/analytics.readonly` |
| `subject` | — | Optional domain-wide delegation subject |
| `tokenEnv` | — | Env var for access token (default `GA_ACCESS_TOKEN`) |
| `propertyId` | `GA_PROPERTY_ID` | Non-secret property id injected into Material |
| `propertyEnv` | — | Env var name for property id (default `GA_PROPERTY_ID`) |

```yaml
exec:
  command: ["./bin/pade-provider-google-analytics"]
  config:
    tokenEnv: GA_ACCESS_TOKEN
    propertyId: properties/123456789
    serviceAccountFile: /run/secrets/ga-sa.json
    scope: https://www.googleapis.com/auth/analytics.readonly
```

## Security notes

- Durable service account private key stays broker/host-side; only the access token (and non-secret property id) is returned as Material.
- Errors must not echo response bodies or private key material.
- Prefer [`ga-property-meta`](../../demo-project/scripts/ga-property-meta) for dogfood validation.
