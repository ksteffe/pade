# PADE Intent Specification

**Status:** Draft / Exploratory  
**Version:** v1alpha1 (`apiVersion: pade.local/v1alpha1`)  
**Machine-readable shape:** [pade.schema.json](pade.schema.json)  
**Conventions:** [../docs/manifest-conventions.md](../docs/manifest-conventions.md)

## Purpose

The Intent Specification defines **portable project metadata**: what external capabilities a development workload for this repository *may need*.

**The portable thing is the intent.**

It answers:

> What external capabilities does this project declare that a development workload may need?

It does **not** answer:

- where execution occurs;
- which agent or IDE is used;
- where secrets are stored;
- which broker is used;
- which vendor provides the capability;
- which human or workload is authorized;
- what credential value will be issued.

Intent is repository-scoped declaration. Authorization and materialization happen elsewhere (Consumer, Broker, providers, and downstream resources). See [consumer.md](consumer.md), [broker.md](broker.md), and [../SECURITY.md](../SECURITY.md).

## Current representation

The current Intent document is a Kubernetes-style declarative API object, conventionally named `pade.yaml` (or passed via consumer flags such as `pade validate -f …`).

PADE uses this grammar for familiarity (`apiVersion`, `kind`, `metadata`, `spec`) but **does not require Kubernetes**. `DevelopmentSession` is **not** currently a Kubernetes CRD.

The JSON Schema at [pade.schema.json](pade.schema.json) is the **machine-readable definition** of the v1alpha1 manifest shape. Implementations that claim Intent v1alpha1 support MUST validate against that schema (or an equivalent encoding of the same constraints).

Do not invent fields that the schema does not define.

The `apiVersion` group `pade.local` and the schema `$id` are **exploratory identifiers** (not hosted URLs). The `.local` suffix intentionally avoids claiming a public DNS domain.

### Root fields (v1alpha1)

| Field | Required | Role |
|-------|----------|------|
| `apiVersion` | yes | MUST be `pade.local/v1alpha1`. |
| `kind` | yes | MUST be `DevelopmentSession`. |
| `metadata` | yes | Portable identity. `metadata.name` is required (DNS-1123 subdomain-ish). Optional `labels` / `annotations` maps of strings. |
| `spec` | yes | Desired portable intent. |

`additionalProperties` is false at the root: unknown top-level keys (including a checked-in `status`) are rejected.

Do **not** pull Kubernetes ObjectMeta into Intent (`uid`, `resourceVersion`, `generation`, `managedFields`, timestamps, `ownerReferences`, `finalizers`, …).

### `spec` fields

| Field | Required | Role |
|-------|----------|------|
| `capabilities` | no | Map of capability id → request object. Primary Intent surface. |

Environment construction, services, and lifecycle remain owned by Dev Containers / DevPod (or peers). They are **not** part of normative v1alpha1 Intent input.

### Capability request object

Each entry under `spec.capabilities` is a **request** object. Allowed properties:

| Field | Role |
|-------|------|
| `access` | Free-form access intent string (for example `read` or `write`). Not an authorization decision. |
| `provider` | Optional **prototype binding hint** (for example `env`, `vault`). Prefer user/org-level bindings long term; this field is not a portable vendor integration. |
| `env` | Optional list of **environment variable names** (not values), mainly for `provider: env` demos. |
| `required` | Optional boolean (schema default `true`). |

Capability **names** (map keys) are opaque strings. Vocabulary is exploratory (see below).

### Simplest useful example

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

A document with empty `spec: {}` (or empty `capabilities`) is schema-valid but declares no capabilities. Examples live under [examples/](examples/) (for example [examples/web-app.yaml](examples/web-app.yaml)).

### Legacy `version: "0.1"` manifests

The historical flat shape:

```yaml
version: "0.1"
capabilities:
  github.user.read:
    access: read
```

is **not** accepted. Conforming Consumers SHOULD reject it with an explicit migration error pointing at `apiVersion: pade.local/v1alpha1` / `kind: DevelopmentSession`. Do not silently reinterpret legacy documents.

## Intent is not authorization

This distinction is load-bearing:

| `pade.yaml` means | `pade.yaml` does **not** mean |
|-------------------|-------------------------------|
| “This project may need capability X.” | “This workload is authorized to receive capability X.” |

Authorization occurs later at the consumer / broker / provider / resource boundary.

A malicious repository MUST NOT gain authority merely by adding capability names to `pade.yaml`. Consumers and brokers MUST treat Intent as **untrusted input** that describes requests, not grants. See [../SECURITY.md](../SECURITY.md).

## Vendor neutrality

Portable Intent MUST NOT contain:

- Keeper record IDs or Keeper Notation handles;
- Vault paths;
- 1Password item IDs or `op://` references;
- Cursor configuration;
- broker endpoints or audiences;
- `KSM_CONFIG` or other bootstrap secrets;
- API tokens or credential values;
- vendor-specific credential configuration.

Those details belong to **reference implementation bindings**, broker policy, or other non-portable configuration—not the Intent document.

Transitional note: optional `provider` / `env` fields on a capability request are prototype hints already present in the schema. Prefer keeping resolution configuration in user/org bindings. Do not use Intent fields to smuggle vendor secrets or broker URLs.

## `status` (future, non-input)

Kubernetes-style `spec` / `status` semantics are useful conceptually:

- `spec` = desired portable intent
- `status` = how a particular implementation satisfied it

PADE does **not** yet define a persisted session/status model. Checked-in Intent documents MUST NOT include `status`. Reference Consumers continue to expose satisfaction via `plan` and `capabilities` output.

A non-normative future illustration (runtime-produced, not authoring input):

```yaml
# NON-NORMATIVE — future direction only; not v1alpha1 input
status:
  conditions:
    - type: CapabilitiesResolved
      status: "True"
```

## Relationship to Dev Containers

| Concern | Owner |
|---------|--------|
| What software / toolchain / image is available? | Dev Containers (`devcontainer.json`) |
| What external authority / capability might be required? | PADE Intent (`pade.yaml`) |

PADE does **not** replace Dev Containers. Workspace lifecycle (create, SSH, ports, prebuilds) should remain with DevPod or an equivalent runtime.

## Capability vocabulary

Capability naming is currently **exploratory**.

Examples in this repository (`github.user.read`, `github.repo.read`, `google-analytics.read`, `datadog.logs.read`) illustrate shapes only.

- **`github.user.read`** — common in stage-1 direct-materialization dogfood (PAT / whoami baseline).
- **`github.repo.read`** — preferred pre-release derived-token dogfood (installation token; repo-scoped validation—not `/user` whoami).
- **`google-analytics.read`** — second structural provider test; directory name is dogfood convenience only (not GA product support in PADE).

None of these names are registered or normative. External repos and dogfood may use other opaque strings.

This pass does **not** define:

- a normative global capability registry;
- canonical namespaces;
- registration or extension rules for third parties;
- a fixed taxonomy of access verbs.

Those remain future specification work. Implementations SHOULD treat capability ids as opaque strings for matching and display until vocabulary rules exist.

## Future capability types (open design)

Current reference work has focused primarily on **credentials / material** (resolved into environment variables).

Future capabilities might represent resources or leases such as:

- `preview.http`
- ephemeral database
- temporary environment
- service lease
- temporary queue / storage / cloud role

These ideas are **not** part of the normative v1alpha1 schema and have no standardized result model yet. See [Open specification questions](README.md#open-specification-questions) in the spec overview. Do not add them to Intent documents expecting interoperability.

## Normative vs reference

| Layer | Location |
|-------|----------|
| Intent Spec (this document + schema) | `spec/` |
| Reference parser / validator | `internal/manifest` (and planner output in `internal/planner`) |

Third-party Intent validators need not use this repository’s Go packages; they MUST respect the same schema constraints if they claim v1alpha1 compatibility.

## Related documents

- [README.md](README.md) — specification overview
- [consumer.md](consumer.md) — consuming Intent
- [broker.md](broker.md) — authorizing and materializing capabilities
- [../docs/manifest-conventions.md](../docs/manifest-conventions.md) — apiVersion/kind/metadata/spec conventions
- [../SECURITY.md](../SECURITY.md) — trust boundaries
