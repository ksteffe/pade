# PADE Intent Specification

**Status:** Draft / Exploratory  
**Version:** 0.1  
**Machine-readable shape:** [pade.schema.json](pade.schema.json)

## Purpose

The Intent Specification defines **portable project metadata**: what external capabilities a development workload for this repository *may need*.

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

The current Intent document is conventionally named `pade.yaml` (or passed via consumer flags such as `pade validate -f …`).

The JSON Schema at [pade.schema.json](pade.schema.json) is the **machine-readable definition** of the v0.1 manifest shape. Implementations that claim Intent v0.1 support MUST validate against that schema (or an equivalent encoding of the same constraints).

Do not invent fields that the schema does not define.

### Root fields (v0.1)

| Field | Required | Role |
|-------|----------|------|
| `version` | yes | MUST be the string `"0.1"`. |
| `capabilities` | no | Map of capability id → request object. Primary v0.1 surface. |
| `environment` | no | Optional pointer to a Dev Container config (`devcontainer` path). Lifecycle remains Dev Containers / DevPod. |
| `services` | no | Optional legacy/compatibility service map (`command`, `port`, optional `ingress`). |
| `lifecycle` | no | Optional idle/max duration strings from earlier drafts. |

`additionalProperties` is false at the root: unknown top-level keys are rejected.

### Capability request object

Each entry under `capabilities` is a **request** object. Allowed properties:

| Field | Role |
|-------|------|
| `access` | Free-form access intent string (for example `read` or `write`). Not an authorization decision. |
| `provider` | Optional **prototype binding hint** (for example `env`, `vault`). Prefer user/org-level bindings long term; this field is not a portable vendor integration. |
| `env` | Optional list of **environment variable names** (not values), mainly for `provider: env` demos. |
| `required` | Optional boolean (schema default `true`). |

Capability **names** (map keys) are opaque strings in v0.1. Vocabulary is exploratory (see below).

### Simplest useful example

```yaml
version: "0.1"
capabilities:
  github.user.read:
    access: read
```

A document with only `version: "0.1"` is schema-valid but declares no capabilities. Capability-first examples live under [examples/](examples/) (for example [examples/web-app.yaml](examples/web-app.yaml)).

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

Transitional note: optional `provider` / `env` fields on a capability request are prototype hints already present in the v0.1 schema. Prefer keeping resolution configuration in user/org bindings. Do not use Intent fields to smuggle vendor secrets or broker URLs. A future manifest version *may* further separate pure portable intent from such runtime hints; that separation is not designed yet.

## Relationship to Dev Containers

| Concern | Owner |
|---------|--------|
| What software / toolchain / image is available? | Dev Containers (`devcontainer.json`) |
| What external authority / capability might be required? | PADE Intent (`pade.yaml`) |

PADE does **not** replace Dev Containers. Workspace lifecycle (create, SSH, ports, prebuilds) should remain with DevPod or an equivalent runtime. Optional `environment.devcontainer` may point at a Dev Container path; it does not make PADE an orchestrator of that lifecycle.

## Capability vocabulary

Capability naming is currently **exploratory**.

Examples in this repository (`github.user.read`, `google-analytics.read`, `datadog.logs.read`) illustrate shapes only.

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

These ideas are **not** part of the normative v0.1 schema and have no standardized result model yet. See [Open specification questions](README.md#open-specification-questions) in the spec overview. Do not add them to Intent documents expecting interoperability.

## Normative vs reference

| Layer | Location |
|-------|----------|
| Intent Spec (this document + schema) | `spec/` |
| Reference parser / validator | `internal/manifest` (and planner output in `internal/planner`) |

Third-party Intent validators need not use this repository’s Go packages; they MUST respect the same schema constraints if they claim v0.1 compatibility.

## Related documents

- [README.md](README.md) — specification overview
- [consumer.md](consumer.md) — consuming Intent
- [broker.md](broker.md) — authorizing and materializing capabilities
- [../SECURITY.md](../SECURITY.md) — trust boundaries
