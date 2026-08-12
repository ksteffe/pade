# PADE Manifest Conventions

**Status:** Draft / Exploratory  
**Applies to:** Intent documents (`pade.yaml`) shaped as `DevelopmentSession` (`pade.local/v1alpha1`)

PADE borrows Kubernetes-style declarative API-object grammar. It does **not** require Kubernetes, CRDs, controllers, or API-server metadata.

## Field meanings

| Field | Role |
|-------|------|
| `apiVersion` | Wire-format / version contract (`pade.local/v1alpha1`) |
| `kind` | Resource semantic type (`DevelopmentSession`) |
| `metadata` | Portable identity (`name`) plus optional labels/annotations |
| `spec` | Desired development-session intent (capability requests) |
| `status` | Reserved for a possible future **runtime-produced** observed state — **not** v1alpha1 input |

Do not invent Kubernetes ObjectMeta fields (`uid`, `resourceVersion`, `generation`, `managedFields`, timestamps, `ownerReferences`, `finalizers`, …).

## Division of responsibility

```text
Repository
    |
    v
DevelopmentSession.spec
    |
    | portable intent
    v
Conforming implementation
    |
    +-- runtime provider
    +-- capability/credential provider
    +-- network/ingress provider
    |
    v
Working development session
```

**Standardize the intent, not the implementation.**

`spec.capabilities` names authority a session may need. It must not encode Vault paths, Keeper IDs, `op://` refs, broker URLs, or which runtime hosts the workspace. Those belong to local/org bindings, brokers, and existing runtimes (DevPod, Coder, cloud agents, Kubernetes as one possible host, etc.).

## Exploratory `pade.local` API group

`apiVersion: pade.local/v1alpha1` and schema `$id` values under `https://pade.local/schema/...` are **exploratory identifiers**, not published URLs. The `.local` suffix is intentional: it avoids claiming a public DNS domain (`.local` is special-use / link-local). Revisit the group name before any public standards publish if a real owned domain is preferred.

## Non-normative adjacent influence

This convention was influenced methodologically by adjacent CNCF discussion around project-agnostic declarative specifications for application integration dependencies ([cncf/toc#1797](https://github.com/cncf/toc/issues/1797)):

- declare requirements independently of implementations;
- prefer existing specifications when possible;
- keep the format programmatically consumable;
- let conforming systems choose how requirements are satisfied.

PADE is **not** affiliated with CNCF, endorsed by CNCF, compatible with that unfinished specification, or part of that initiative.

## Minimal example

```yaml
apiVersion: pade.local/v1alpha1
kind: DevelopmentSession
metadata:
  name: demo
spec:
  capabilities:
    github.user.read:
      access: read
```

Machine-readable contract: [spec/pade.schema.json](../spec/pade.schema.json). Prose: [spec/intent.md](../spec/intent.md).
