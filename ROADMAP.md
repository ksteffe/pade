# PADE Roadmap

**Authoritative planned work** for this repository. Specs under [`spec/`](spec/README.md) remain the interoperability contract; [README.md](README.md) is the project entrypoint; [RFC.md](RFC.md) / [DESIGN.md](DESIGN.md) retain architectural and implementation history.

This document is planning only until milestones are implemented. It does not change Intent schema, Consumer/Broker wire behavior, or reference code by itself.

## Purpose and principles

PADE should prove **generic authority brokering** once. Downstream systems (Google Analytics, Cloudflare Tunnel, GitHub, …) should work because the mechanism is generic—not because PADE knows about them.

Governing principles:

1. Keep PADE deliberately small and vendor-neutral.
2. Compose mature tools; do not reimplement workspace lifecycle, IAM, or secret stores.
3. Treat `apiVersion: pade.local/v1alpha1` / `kind: DevelopmentSession` and the Intent / Consumer / Broker specs as authoritative.
4. Prefer dogfood evidence before extending Intent or inventing Grant/Lease protocols.
5. Do not reclaim responsibility for starting and orchestrating development environments unless real dogfood proves that necessary.
6. PADE standardizes the **seam**, not every provider implementation. Vendor-specific business logic (Google Analytics, GitHub Apps, Keeper, 1Password, cloud IAM exchanges, …) belongs in external providers or deployment configs—not in PADE core.
7. Brokers **SHOULD** prefer session-scoped, short-lived, or otherwise **derived** credentials over delivering durable source credentials when the configured provider supports such derivation. Direct durable-secret **materialization** remains a valid interoperability mechanism where necessary; PADE must not pretend every external system supports ephemeral credentials.

Document roles:

| Document | Role |
|----------|------|
| [README.md](README.md) | Project entrypoint |
| [`spec/`](spec/README.md) | Interoperability contract |
| [RFC.md](RFC.md) | Architectural evolution (history + current framing) |
| [DESIGN.md](DESIGN.md) | Reference implementation design history |
| **ROADMAP.md** (this file) | Planned work |

## Current implementation assessment

Derived from the repository as of this roadmap pass (reference Consumer/Broker code, schema, dogfood, CI). Status meanings:

- **DONE** — present and usable; do not re-roadmap as new work
- **PARTIAL** — present with known limits; no PADE change until dogfood shows a real gap (unless noted)
- **MISSING** — not implemented; near-term PADE work or explicit follow-up
- **UNKNOWN** — needs real dogfood before deciding

| Area | Status | Notes |
|------|--------|-------|
| DevelopmentSession parsing/validation (`pade.local/v1alpha1`) | DONE | [`internal/manifest`](internal/manifest), [`spec/pade.schema.json`](spec/pade.schema.json) |
| Capability declarations (`spec.capabilities`) | DONE | Free-form opaque names; no registry |
| Consumer (`validate` / `plan` / `capabilities` / `exec`) | DONE | [`cmd/pade`](cmd/pade) |
| Local provider bindings | DONE | env, Vault, 1Password, Keeper, Keeper Secrets Manager |
| Remote broker mode | DONE | `provider: broker` client + [`cmd/pade-broker`](cmd/pade-broker) spike |
| Cursor OIDC workload identity | DONE | Reference adapter; not a normative PADE identity standard |
| Broker authentication | DONE (spike) | JWKS / RS256; iss/aud; JTI replay store still deferred |
| Broker authorization | DONE (spike) | Server policy: subject + capability allowlist + optional `repo_urls` |
| Provider materialization | DONE | Server-side providers via broker bindings |
| Arbitrary Material / env injection | DONE (env maps) | `Material.Env map[string]string` only; sufficient for token/API credentials |
| Structured / multiline credential Material | PARTIAL | String values may contain newlines; not dogfooded; **no anticipated PADE work** until a proven deficiency |
| `pade exec` | DONE | Process-scoped resolve → inject → wait → discard |
| Child exit-code propagation | DONE | Maps to process exit |
| Output redaction | DONE | Exact-match secret redaction on child stdout/stderr (defense in depth) |
| Signal forwarding (SIGINT / SIGTERM → child) | MISSING | No explicit forward; relies on `CommandContext` cancel / OS defaults |
| Long-running child-process behavior | PARTIAL / UNKNOWN | Wait + stream works; process-group / signal behavior needs dogfood |
| Remote broker endpoint / audience configuration | DONE | Local bindings only (not in Intent) |
| Containerized `pade-broker` | DONE | Root [`Dockerfile`](Dockerfile); `make smoke-broker-container` |
| Cloud Run–compatible transport mode | DONE | `PORT` + `-tls-termination=proxy` |
| Fake broker dogfood | DONE | `make dogfood-broker` (CI smoke); Stage B real Cursor OIDC (not CI) |
| Version reporting (`pade` / `pade-broker --version`) | MISSING | No SemVer / commit ldflags today |
| GitHub Release artifacts | MISSING | No release workflow |
| Broker container publishing (GHCR) | MISSING | Image buildable and smoked; not published |

**Genuinely missing PADE-repository work** for the near term:

1. Release / version foundation (Milestone A)
2. Generic `pade exec` signal-forwarding improvement **if** long-running dogfood shows a gap (Milestone E follow-up—not “Cloudflare support”)
3. External dogfood validation of released artifacts (Milestones B–G; mostly external repos)
4. Endpoint-declaration architecture decision after preview dogfood (Milestone F)—implementation only if warranted

Do not roadmap work that is already DONE.

## Generic Material dogfood verdict

The GitHub-token broker path already proves the generic capability-resolution chain:

```text
DevelopmentSession
    ↓
PADE Consumer (pade)
    ↓
Cursor workload identity (OIDC)
    ↓
PADE Broker (pade-broker)
    ↓
authorization (server policy)
    ↓
Provider (e.g. Keeper Secrets Manager)
    ↓
Material (env map, e.g. GITHUB_TOKEN)
    ↓
pade exec child process
```

Evidence: `make dogfood-broker`, Stage B (`make dogfood-broker-stage-b`), demo Intent `github.user.read`, and related provider dogfoods that resolve the same capability through different bindings.

**PADE-level credential brokering is sufficiently demonstrated.** Additional downstream systems such as Google Analytics should be treated as **external integration/dogfood**, not new PADE examples or provider adapters.

Keep examples minimal. Do **not** add GA examples, Cloudflare examples, or vendor-specific PADE integrations merely to demonstrate the same Material behavior again.

## Material vs Endpoint vs Grant

Three emerging concepts—only **Material** is implemented today:

| Concept | Meaning | Status |
|---------|---------|--------|
| **Material** | Authority returned to a child process (today: env map). Examples: API credential, tunnel token. | Current protocol / reference implementation |
| **Endpoint** | Potential portable description of a **local** service a capability may act upon. Example: HTTP on port 3000. | Unevaluated; see Milestone F |
| **Grant / Lease** | Potential future broker result for dynamically provisioned resources (preview URL, ephemeral DB, expiration/revocation). | Deferred; no schema or type |

Immediate Cloudflare / preview dogfood should first test:

```text
existing Material
+
repo-local service startup
+
possibly future Endpoint declaration (only if dogfood requires it)
```

Do **not** implement Grant/Lease in the near term.

### Fulfillment maturity (Material path)

Near-term dogfood (Milestones D–G) intentionally uses **direct materialization**. A later hardening step (Milestone I) improves how brokers fulfill the same portable capability request without changing DevelopmentSession Intent.

| Stage | What happens | Status |
|-------|----------------|--------|
| **1. Direct materialization** | Broker retrieves a secret from a source store and safely delivers it as `Material` to the DevelopmentSession. | **Today** — required for current dogfood and legacy systems |
| **2. Derived / session-scoped credentials** | Durable authority remains broker-side. A configurable provider **derives** or **issues** a temporary credential; the Consumer receives only that derived `Material`, with expiry/lifecycle associated with the session where possible. | **Milestone I** — planned hardening |
| **3. Mediated capabilities** | Consumer exercises a capability through the broker/provider **without** receiving the underlying credential at all. | **Future direction** — not near-term |

#### DevelopmentSession relationship

Preserve the current DevelopmentSession direction:

- **Durable authority** belongs to an external identity / provider / secret system.
- A **capability** is authorized for a particular **DevelopmentSession**.
- The **Consumer** exercises that capability.
- The **Broker** / **Provider** determines the safest available **materialization** (or mediation) mechanism.

The DevelopmentSession Intent **must not** need to know whether fulfillment ultimately came from a static secret, a derived access token, an OIDC exchange, an external credential issuer, or a broker-mediated service. That remains a fulfillment concern.

#### Conceptual fulfillment pipeline (illustrative names)

Stage labels such as SOURCE / DERIVATION / DELIVERY are **illustrative only**—not normative API terms. Prefer existing PADE vocabulary (`Provider`, **materialization**, `Material`) until naming is decided (see [Open design questions](#open-design-questions)).

```text
Source
  Where durable authority originates
  (Keeper, 1Password, Vault, cloud secret manager, …)
        ↓
Provider materialization / optional derivation-issuance
  Optional broker-side transformation
  durable credential → short-lived credential
  OIDC identity → temporary cloud credential
  private key → installation/session token
  static secret → unchanged secret (direct materialization)
        ↓
Delivery
  How the resulting capability is made usable
  Material.env (today)
  file / credential helper / mediated operation (possible later)
```

Non-normative examples (do **not** imply PADE core knows these vendors):

```text
Keeper-held Google service-account credential
    ↓
external PADE provider (derivation)
    ↓
short-lived OAuth access token
    ↓
DevelopmentSession (Material)

GitHub App private key
    ↓
external PADE provider (derivation)
    ↓
short-lived installation token
    ↓
DevelopmentSession (Material)
```

PADE standardizes the seam so independent provider implementations can plug in. It does **not** contain Google Analytics–, GitHub–, Keeper–, or other vendor-specific business logic for these flows.

## Ownership boundaries

### PADE repository (this repo)

Near-term:

- Release / version foundation (CLI + broker container artifacts)
- Any genuinely missing **generic** Consumer/Broker execution behavior (e.g. signal forwarding if dogfood fails)
- External dogfood validation criteria (documented here)
- Endpoint-declaration **decision** after dogfood (schema only if warranted)

Later (after dogfood priorities):

- Derived / session-scoped fulfillment and a provider-neutral extension seam ([Milestone I](#milestone-i--derived--session-scoped-fulfillment-hardening))

PADE should **not** implement:

- Google Analytics client code, GA providers, GA capability-registry entries, or GA examples beyond generic Material behavior already covered
- Cloudflare-specific APIs or tunnel provisioning
- Application start/stop orchestration (DevPod / repo tooling owns lifecycle)
- Vendor-specific credential derivation logic in PADE core (that belongs in external providers)

### `pade-broker-deployment` (external)

- Google Cloud Run (or equivalent) hosting of the broker
- Private broker policy and secret-store bootstrap
- Capability → credential bindings (including GA and Cloudflare credentials)
- Consumption of **released** broker images (`ghcr.io/ksteffe/pade-broker:vX.Y.Z`)

### `after-certainty` (external)

- `DevelopmentSession` Intent for the site
- Cursor Cloud / iOS agent environment setup
- GA4 tooling, analytics commands/scripts, use of returned credentials
- Application start command, local port, Cloudflare preview command, stable preview hostname
- Consumption of **released** PADE CLI (`PADE_VERSION=vX.Y.Z`)

Do not move these external responsibilities into PADE merely because PADE enables them.

### Google Analytics ownership

| Owner | Responsibility |
|-------|----------------|
| `pade-broker-deployment` | Private capability → GA credential binding; secret-store configuration |
| `after-certainty` | GA4 tooling; analytics commands; use of returned Material |
| **PADE** | Nothing GA-specific. If dogfood reveals a **generic** Material deficiency, open a generic PADE follow-up—do not anticipate one. |

### Cloudflare ownership

| Owner | Responsibility |
|-------|----------------|
| `pade-broker-deployment` | Private capability → Cloudflare credential binding |
| `after-certainty` | App dev command; `cloudflared` install; preview workflow; hostname configuration |
| **PADE** | No Cloudflare-specific APIs or tunnel provisioning near-term. Track the **endpoint declaration** question (Milestone F) as a portable protocol concern if needed. |

## Forward milestones

Completed learning dogfood (0–9 / 9b) is summarized under [Historical dogfood milestones](#historical-dogfood-milestones). It is not reopened as the live plan.

| Milestone | Focus | Where work happens |
|-----------|--------|--------------------|
| **A — Release foundation** | Versioned CLI + broker artifacts | PADE repo |
| **B — Released broker deployment** | Private deploy consumes released broker image | `pade-broker-deployment` |
| **C — Released Consumer dogfood** | Cursor Cloud uses released `pade` against real broker | External + PADE artifacts |
| **D — External credential dogfood** | after-certainty uses GA via generic Material (**direct materialization** OK) | External (`after-certainty` + broker deployment) |
| **E — External preview dogfood** | App + Cloudflare Tunnel via generic Material; evaluate long-running `pade exec` | External + possible generic exec fix in PADE |
| **F — Endpoint decision** | Decide whether portable endpoint declaration is warranted | Architecture decision (PADE Intent only if yes) |
| **G — Full Cursor iOS workflow** | End-to-end acceptance story | External acceptance |
| **H — Post-dogfood protocol evaluation** | Only generic deficiencies return to PADE | Conditional |
| **I — Derived / session-scoped fulfillment** | Provider-side derivation seam; prefer short-lived Material | PADE repo (hardening) + external providers |

### Milestone A — Release foundation

**Goal:** External consumers pin `PADE_VERSION=vX.Y.Z` instead of arbitrary `main` commits.

**SemVer strategy:** Pre-1.0 SemVer. Breaking changes allowed with minor bumps while major remains `0`. Initial release: **`v0.1.0`** (aligns with existing “v0.1 capability path” language; broker remains experimental).

**PADE-side requirements (to implement later—not in this docs pass):**

- `pade --version` and `pade-broker --version` reporting SemVer + source commit (+ optional build time)
- Annotated git tags `vX.Y.Z`
- GitHub Releases with release notes (generated from commits/PRs at release time; formal `CHANGELOG.md` optional later)
- Checksums (e.g. SHA-256) for all CLI artifacts
- CLI binaries: **Linux amd64**, **Linux arm64**, **macOS arm64** (`pade`; include `pade-broker` binary if useful for local spikes). Skip **macOS amd64** unless a consumer asks.
- Versioned broker container on GHCR: `ghcr.io/ksteffe/pade-broker:vX.Y.Z`
- Immutable image digest published alongside the tag; prefer digest pins in production deploy
- OCI labels: `org.opencontainers.image.version`, `org.opencontainers.image.revision` (git commit), `org.opencontainers.image.source`

#### Release workflow recommendation

Prefer **avoiding accidental releases**.

**Recommendation: manually triggered GitHub Actions workflow (`workflow_dispatch`) with a required explicit version input** (e.g. `v0.1.0`).

Conceptual flow:

```text
normal merge to main
    ↓
CI (unit + smoke + container smoke)
explicit release (workflow_dispatch + version)
    ↓
full tests
    ↓
CLI binaries + checksums
    ↓
broker container → GHCR
    ↓
GitHub Release (+ tag)
```

- **No release on merge to `main`.**
- Tag-only triggers are rejected for the first release system because accidental `git push --tags` is too easy; an explicit version input matches “deliberate release.”
- Automatic version bumping remains deferred unless a compelling simple approach appears later.

#### Artifact contract (PADE publishes; externals consume)

**`pade-broker-deployment` should eventually consume:**

```text
ghcr.io/ksteffe/pade-broker:vX.Y.Z
```

without cloning or building PADE source. Operators must be able to identify:

- PADE version (tag)
- source commit (OCI revision label / release metadata)
- image digest

**Cloud development environments should install:**

```text
pade vX.Y.Z
```

from a GitHub Release asset (not `go install` of an arbitrary commit).

This roadmap covers **PADE-side** publishing only. External repos are not modified here.

### Milestone B — Released broker deployment

External: private broker deployment pulls the released GHCR image, mounts policy/bindings/secrets, and runs with Cloud Run–compatible transport (`PORT`, trusted TLS termination).

PADE acceptance: image runs as already smoked by `make smoke-broker-container`; no PADE protocol change required.

### Milestone C — Released Consumer dogfood

External Cursor Cloud (or equivalent) installs released `pade`, points agent bindings at the real broker, and resolves a capability end-to-end with Cursor OIDC.

PADE acceptance: released CLI talks to released broker using the existing wire protocol.

### Milestone D — External credential dogfood

```text
after-certainty
    ↓
DevelopmentSession
    ↓
released PADE Consumer
    ↓
Cursor OIDC
    ↓
released pade-broker
    ↓
private broker bindings → Google Analytics credential
    ↓
repo-owned GA tooling
```

**PADE acceptance criterion:** Neither Google Analytics nor any other downstream API requires PADE to know what that API is. Capability names such as `analytics.google.read` remain free-form exploratory strings—not a registry entry in this repo.

Near-term fulfillment may use **direct materialization** of a durable credential into the DevelopmentSession. Preferring derived/session-scoped credentials is **Milestone I** hardening—not a blocker for this dogfood.

### Milestone E — External preview dogfood

after-certainty runs its normal application and establishes a Cloudflare Tunnel using **generic** PADE-brokered Material (tunnel token or equivalent), plus **repo-local** start command / port / `cloudflared` invocation.

Evaluate:

- Long-running `pade exec` children
- Ctrl-C / SIGTERM behavior
- Child exit-code propagation
- Material lifetime (process-scoped discard after exit)
- Output redaction under sustained streaming

If signal forwarding or related generic execution gaps appear, fix them in PADE as **generic `pade exec` improvements**—not as Cloudflare support.

### Milestone F — Endpoint decision

A credential capability alone may not fully describe a preview workflow. A preview needs external authority **and** a local service to expose.

**Architectural question (do not change schema until decided):**

> Should a `DevelopmentSession` be able to declare local endpoints/services that external capabilities may act upon?

Illustrative **future** shape only (not accepted by today’s schema):

```yaml
apiVersion: pade.local/v1alpha1
kind: DevelopmentSession
metadata:
  name: example
spec:
  capabilities:
    preview.http:
      access: use
  endpoints:
    web:
      protocol: http
      port: 3000
```

Potential pairings (conceptual):

| Capability (exploratory name) | Endpoint |
|-------------------------------|----------|
| `preview.http` | `web` |
| `debug.remote` | `debug` |
| `webhook.receive` | `webhooks` |
| `database.proxy` | `postgres` |

Keep **application startup** separate from endpoint intent:

| Layer | Question | Owner today |
|-------|----------|-------------|
| App / runtime config | How do I start the service? | `package.json` / Makefile / Dev Container |
| DevelopmentSession | What local endpoint exists / may need external capability? | **Unevaluated** (possibly Intent later) |
| Capability | What external authority is required? | Intent `spec.capabilities` |

Do **not** reintroduce old lifecycle/orchestration fields. PADE deliberately moved away from owning workspace lifecycle.

#### Decision gate (dogfood-first)

```text
First: dogfood after-certainty with a repo-local preview command
that knows start command, port, and cloudflared invocation
outside PADE.
        ↓
If this feels clean
        → keep endpoint metadata out of PADE.
        ↓
If every Consumer needs repo-specific knowledge of which
local service/port a capability targets
        → promote endpoint declaration into the Intent roadmap
          and implement only then.
        ↓
If PADE must also start/stop services itself
        → revisit lifecycle carefully; do not assume this outcome.
```

This is an explicit **post-preview-dogfood** architecture decision (Milestone F). Do not implement endpoints immediately.

### Milestone G — Full Cursor iOS workflow

Acceptance story:

From Cursor iOS, launch a fresh after-certainty Cloud Agent and ask it to inspect Google Analytics, make an appropriate change, run the application, and provide a public preview—**without manually copying credentials into the agent**.

Expected path:

```text
released PADE CLI
    ↓
DevelopmentSession
    ↓
Cursor workload identity
    ↓
released PADE broker
    ↓
private authority mapping
    ↓
GA credential / tunnel credential (Material)
    ↓
repo-owned tooling (GA scripts, app start, cloudflared)
```

PADE itself remains unaware of the downstream vendors.

### Milestone H — Post-dogfood protocol evaluation

Only **generic** deficiencies discovered during real use return to the PADE repository (Intent, Consumer, Broker, or reference execution behavior). Vendor-specific work stays in external repos.

### Milestone I — Derived / session-scoped fulfillment (hardening)

**When:** After basic broker/provider interoperability works (Milestones A–C) and preferably after the full Cursor iOS acceptance story (Milestone G). Do **not** block near-term dogfood (D–G) on this milestone—those may continue to use direct materialization.

**Goal:** A broker can satisfy a requested capability by **deriving** a short-lived or session-scoped credential from durable authority, without exposing the durable credential to the DevelopmentSession or Consumer—when a configured provider supports that path.

```text
durable authority
    ↓
source / secret store
    ↓
broker-side provider (materialization + optional derivation/issuance)
    ↓
short-lived / session-scoped Material
    ↓
Consumer / DevelopmentSession
```

Eventually (mediated direction, not required for Milestone I completion):

```text
durable authority
    ↓
broker-side provider
    ↓
broker-mediated capability
    ↓
Consumer never receives credential Material
```

**PADE-side focus (generic seam only):**

- Define a **provider extension seam** so independent implementations can fulfill/derive capabilities without vendor-specific code in PADE core.
- Distinguish the **semantic abstraction** (fulfill / derive a capability) from **implementation bindings**. Candidate bindings (not a final design choice in this roadmap):
  - subprocess / exec
  - local plugin
  - HTTP
  - gRPC
  - separately packaged provider implementation
- An exec-style binding may be an attractive **first experiment** later (providers in any language), but the abstraction must not collapse into “a generic shell hook.”
- Prefer derived Material when the configured provider supports it (principle 7); keep direct materialization valid.

**Dependencies / design questions** (not committed requirements yet):

- DevelopmentSession lifecycle semantics (what “session-scoped” means in practice)
- Capability / provider matching
- Broker policy evaluation relative to derived fulfillment
- Provider extension contract
- Credential expiry metadata (overlap with Grant/Lease open question)
- Revocation / lifecycle behavior
- Secure provider execution / isolation
- Delivery bindings beyond today’s env `Material`

**Out of scope for PADE core:** Google Analytics–, GitHub–, Keeper–, 1Password–, or other vendor-specific derivation logic. Those live in external provider packages or deployment-side configuration (`pade-broker-deployment` and peers).

## Capability naming (exploratory)

External dogfood may use names resembling `analytics.google.read` or `preview.http`.

- Do **not** create a capability registry in PADE.
- Do **not** standardize vendor capability names in this roadmap pass.

Post-dogfood questions (track only):

- Do capability names need namespaces?
- Does `access` add useful semantics?
- Should common capabilities eventually have standardized meanings?

## Open design questions

Track these without freezing Intent or Broker wire formats prematurely. Sequencing ownership: [Milestone I](#milestone-i--derived--session-scoped-fulfillment-hardening) for derivation; Milestone F for endpoints; Grant/Lease remains deferred.

1. **Fulfillment pipeline naming** — Are illustrative labels (source / derivation / delivery) useful long-term, or should the specs stick to existing **Provider** / **materialization** / `Material` vocabulary?
2. **Provider extension contract** — What is the minimal portable seam for independent fulfill/derive implementations? Which implementation binding (exec, plugin, HTTP, gRPC, separate package) should the reference broker explore first?
3. **Expiry / revocation metadata** — Does derived Material need explicit expiry fields, or does that belong to a future Grant/Lease result model?
4. **Mediated capabilities** — Does credential-less, broker-mediated exercise of a capability require Consumer protocol changes beyond today’s resolve → Material path?
5. **Runtime Conditions (CNCF)** — Could Runtime Conditions eventually describe some of the portable *demand* while PADE focuses on creating and fulfilling an identity-bound DevelopmentSession? This remains under discussion with Runtime Conditions maintainers. **Do not** claim that PADE will adopt, extend, replace, or integrate Runtime Conditions yet.
6. **Endpoint declaration** — See Milestone F.
7. **Capability vocabulary** — See [Capability naming](#capability-naming-exploratory).

## Deferred until after the full workflow works

Explicitly deferred:

- Endpoint schema implementation **unless** Milestone F decides it is required
- Grant / Lease model
- **Derived / session-scoped fulfillment and the provider extension seam** — planned as **Milestone I** (after A–G dogfood priorities); not a near-term blocker
- **Mediated capabilities** (Consumer never receives credential Material) — future direction beyond Milestone I’s derived-Material focus
- Provider packaging / plugin distribution choices (which binding wins; how operators install external providers)
- Dynamic preview provisioning by the broker
- Multiple simultaneous previews
- Branch-specific preview URLs
- Capability registry / standardized capability naming
- frp or other tunnel implementations inside PADE
- Additional workload identity adapters (beyond Cursor OIDC reference)
- Codex / Claude (or other agent runtime) support as PADE features
- Managed broker products
- PADE website
- Package-manager distribution (Homebrew, etc.)
- Automatic release versioning
- Supply-chain signing / provenance beyond checksums (revisit before sensitive enterprise recommendations)
- Standards / governance work
- Runtime Conditions adoption or integration (open question only)
- JTI replay store, multi-tenant broker hosting, DB-backed policy (still deferred spike hardening)

## Historical dogfood milestones

The following learning milestones are **complete or spiked** and are not the live forward plan. Detail lives in [README.md](README.md) “Current status”, [DESIGN.md](DESIGN.md), and dogfood guides under [`docs/`](docs/README.md).

| Milestone | Focus |
|-----------|--------|
| **0** | Repo, docs, license, schema stub, examples |
| **1** | `validate` / `plan` against schema |
| **2** | Local bindings (`env`, `vault`) + `capabilities` |
| **3** | Scoped `pade exec --capability` |
| **4** | DevPod dogfood (`examples/demo-project`) |
| **5** | Identity separation (same repo, distinct credentials) |
| **5b** | Vault `-dev` validation of bindings / identity paths |
| **6** | Second credential provider (1Password CLI adapter) |
| **7** | Keeper Commander CLI binding provider (fake-keeper CI) |
| **8** | Local Teleport authenticated ingress (`examples/ingress-demo`) — spike |
| **9** | Keeper Secrets Manager + Cursor Cloud dogfood |
| **9b** | Cursor OIDC + minimal `pade-broker` — **experimental reference Broker** |

Earlier README “9+ / Later” rows (optional release artifacts, external validation) are **superseded** by Milestones A–I in this document.
