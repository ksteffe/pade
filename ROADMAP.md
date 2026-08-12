# PADE Roadmap

**Authoritative planned work** for this repository. Specs under [`spec/`](spec/README.md) remain the interoperability contract; [README.md](README.md) is the project entrypoint; [RFC.md](RFC.md) / [DESIGN.md](DESIGN.md) retain architectural and implementation history.

This document is planning only until milestones are implemented. It does not change Intent schema, Consumer/Broker wire behavior, or reference code by itself.

## Purpose and principles

PADE should prove **generic authority brokering** once. Downstream systems (Google Analytics, Cloudflare Tunnel, GitHub, …) should work because the mechanism is generic—not because PADE knows about them.

Before the first meaningful release (`v0.1.0`), PADE must prove more than direct secret materialization. It must also demonstrate that a broker can fulfill a DevelopmentSession capability through an **independently implemented provider** that can derive a session-scoped credential—without vendor-specific behavior in PADE core.

Governing principles:

1. Keep PADE deliberately small and vendor-neutral.
2. Compose mature tools; do not reimplement workspace lifecycle, IAM, or secret stores.
3. Treat `apiVersion: pade.local/v1alpha1` / `kind: DevelopmentSession` and the Intent / Consumer / Broker specs as authoritative.
4. Prefer dogfood evidence before extending Intent or inventing Grant/Lease protocols.
5. Do not reclaim responsibility for starting and orchestrating development environments unless real dogfood proves that necessary.
6. PADE standardizes the **provider seam**, not every provider implementation. A provider should be architecturally removable from PADE core. Vendor-specific business logic (Google Analytics auth, Google service accounts/OAuth scopes, GitHub App tokens, AWS STS, Keeper/1Password/Vault derivation details, provider-specific field names or credential formats, …) belongs in independently implemented providers—not in PADE core (`cmd/`, portable `internal/` packages).
7. Brokers **SHOULD** prefer session-scoped, short-lived, or otherwise **derived** credentials over delivering durable source credentials when the configured provider supports such derivation. Direct durable-secret **materialization** remains a valid interoperability mechanism where necessary; PADE must work with systems that only expose static credentials and must not pretend every capability can be fulfilled without exposing credential material.
8. Prove the provider seam **before** cutting the initial versioned release. Accumulating many vendor integrations is less important than proving that third parties can implement providers without modifying PADE itself.

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
| Local provider bindings | DONE | env, Vault, 1Password, Keeper, Keeper Secrets Manager (direct materialization adapters) |
| Remote broker mode | DONE | `provider: broker` client + [`cmd/pade-broker`](cmd/pade-broker) spike |
| Cursor OIDC workload identity | DONE | Reference adapter; not a normative PADE identity standard |
| Broker authentication | DONE (spike) | JWKS / RS256; iss/aud; JTI replay store still deferred |
| Broker authorization | DONE (spike) | Server policy: subject + capability allowlist + optional `repo_urls` |
| Provider materialization (in-tree adapters) | DONE | Server-side env/Vault/op/keeper/ksm via broker bindings |
| External/independently packaged provider seam | MISSING | Needed before `v0.1.0` (Milestones B–C) |
| Derived / session-scoped fulfillment dogfood | MISSING | Needed before `v0.1.0` (Milestones D–E) |
| Arbitrary Material / env injection | DONE (env maps) | `Material.Env map[string]string` only; sufficient for token/API credentials |
| Structured / multiline credential Material | PARTIAL | String values may contain newlines; not dogfooded; **no anticipated PADE work** until a proven deficiency |
| `pade exec` | DONE | Process-scoped resolve → inject → wait → discard |
| Child exit-code propagation | DONE | Maps to process exit |
| Output redaction | DONE | Exact-match secret redaction on child stdout/stderr (defense in depth) |
| Signal forwarding (SIGINT / SIGTERM → child) | MISSING | No explicit forward; evaluate in Milestone J if needed |
| Long-running child-process behavior | PARTIAL / UNKNOWN | Wait + stream works; process-group / signal behavior needs dogfood |
| Remote broker endpoint / audience configuration | DONE | Local bindings only (not in Intent) |
| Containerized `pade-broker` | DONE | Root [`Dockerfile`](Dockerfile); `make smoke-broker-container` |
| Cloud Run–compatible transport mode | DONE | `PORT` + `-tls-termination=proxy` |
| Fake broker dogfood (GitHub direct Material) | DONE | `make dogfood-broker` (CI smoke); Stage B real Cursor OIDC (not CI) |
| Version reporting (`pade` / `pade-broker --version`) | MISSING | Part of Milestone G |
| GitHub Release artifacts | MISSING | Part of Milestone G |
| Broker container publishing (GHCR) | MISSING | Part of Milestone G |

**Genuinely missing PADE-repository work** before the first release:

1. Minimal generic provider contract + dogfood binding (Milestones B–C)
2. Google Analytics **reference** provider under `examples/providers/google-analytics/` (Milestone D) — non-normative
3. Cloud-agent derived-credential proof (Milestone E)
4. Spec/docs tightening from that dogfood (Milestone F)
5. Versioned release foundation gated on the above (Milestone G)

Do not roadmap work that is already DONE.

## Direct materialization dogfood verdict

The GitHub-token broker path already proves **direct materialization** (fulfillment maturity stage 1):

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

That path remains **valid and minimal** and is a **required** baseline for `v0.1.0`. It does **not** yet prove the larger model: an independently implemented provider that derives session-scoped credentials while durable authority stays broker-side.

Keep the GitHub dogfood. Do **not** add Cloudflare or additional vendor integrations merely to repeat direct Material delivery. One **non-normative** in-tree Google Analytics reference provider is planned specifically to dogfood the **provider contract and derivation** (Milestones D–E)—not to make GA part of the PADE standard.

## Material vs Endpoint vs Grant

Three emerging concepts—only **Material** is implemented today:

| Concept | Meaning | Status |
|---------|---------|--------|
| **Material** | Authority returned to a child process (today: env map). Examples: API credential, tunnel token. | Current protocol / reference implementation |
| **Endpoint** | Potential portable description of a **local** service a capability may act upon. Example: HTTP on port 3000. | Unevaluated; see Milestone K |
| **Grant / Lease** | Potential future broker result for dynamically provisioned resources (preview URL, ephemeral DB, expiration/revocation). | Deferred; no schema or type |

Post-release Cloudflare / preview dogfood (Milestone J) should first test:

```text
existing Material
+
repo-local service startup
+
possibly future Endpoint declaration (only if dogfood requires it)
```

Do **not** implement Grant/Lease before the initial release.

### Fulfillment maturity (Material path)

| Stage | What happens | Status |
|-------|----------------|--------|
| **1. Direct materialization** | Broker retrieves a secret from a source store and safely delivers it as `Material` to the DevelopmentSession. | **Today** — GitHub dogfood; **required** for `v0.1.0` |
| **2. Derived / session-scoped credentials** | Durable authority remains broker-side. A configurable external provider **derives** or **issues** a temporary credential; the Consumer receives only that derived `Material`, with expiry/lifecycle associated with the session where possible. | **Milestones B–E** — **required** for `v0.1.0` |
| **3. Mediated capabilities** | Consumer exercises a capability through the broker/provider **without** receiving the underlying credential at all. | **Future direction** — not required for `v0.1.0` |

The initial release must demonstrate stages **1 and 2**. Stage 3 remains future work.

#### DevelopmentSession relationship

Preserve the current DevelopmentSession direction:

- **Durable authority** belongs to an external identity / provider / secret system.
- A **capability** is authorized for a particular **DevelopmentSession**.
- The **Consumer** exercises that capability.
- The **Broker** / **Provider** determines the safest available **materialization** (or mediation) mechanism.

The DevelopmentSession Intent **must not** need to know whether fulfillment ultimately came from a static secret, a derived access token, an OIDC exchange, an external credential issuer, or a broker-mediated service. That remains a fulfillment concern. Do not redesign Intent around CNCF Runtime Conditions while that discussion is open.

#### Conceptual fulfillment pipeline (illustrative names)

Stage labels such as SOURCE / PROVIDER / DERIVATION / DELIVERY are **illustrative only**—not normative API terms. Prefer existing PADE vocabulary (`Provider`, **materialization**, `Material`) until naming is decided (see [Open design questions](#open-design-questions)).

```text
Source
  Where durable authority originates
  (Keeper, 1Password, Vault, cloud secret manager, …)
        ↓
Provider / optional derivation
  Given a capability request and authorized DevelopmentSession context,
  fulfill via a configured external implementation. May:
  retrieve durable authority
  derive a temporary credential
  perform an identity exchange
  mint a scoped token
  return an existing secret unchanged
  eventually mediate without returning credential material
        ↓
Delivery
  How the resulting capability is made usable
  Material.env (today)
  file / credential helper / consumer protocol response
  eventually broker-mediated operation
```

The provider mechanism is a **semantic fulfill/derive contract**, not an arbitrary “hook.” An exec/subprocess integration may be a useful first **binding**; that is an implementation choice, not the definition of the abstraction.

Non-normative examples (do **not** imply PADE core knows these vendors):

```text
Keeper-held Google service-account credential
    ↓
Google Analytics reference provider (examples/providers/…)
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

## Ownership boundaries

### PADE repository (this repo)

Near-term (before `v0.1.0`):

- Keep GitHub direct-materialization broker dogfood healthy (Milestone A)
- Define minimal generic provider contract + dogfood binding (Milestones B–C)
- In-tree **non-normative** Google Analytics reference provider at planned `examples/providers/google-analytics/` (Milestone D)
- Cloud-agent derived-credential proof + docs tighten (Milestones E–F)
- Versioned CLI + broker release artifacts (Milestone G), gated on A–F
- Any genuinely missing **generic** Consumer/Broker behavior discovered while proving the seam

PADE should **not** put into core:

- Google Analytics authentication, service-account, OAuth-scope, or API client logic
- Cloudflare-specific APIs or tunnel provisioning
- Application start/stop orchestration (DevPod / repo tooling owns lifecycle)
- A catalog of vendor integrations

In-tree reference providers under `examples/providers/` exist for **dogfooding and illustrating the provider contract**. Their presence does **not** make the vendor, API, or capability part of the PADE standard. Prefer `examples/providers/` over an `extensions/` directory name (avoids collision with CNCF Runtime Conditions “extension” terminology).

### `pade-broker-deployment` (external)

- Google Cloud Run (or equivalent) hosting of the broker
- Private broker policy and secret-store bootstrap (durable authority)
- Capability → source/provider configuration (including durable Google authority and Cloudflare credentials)
- Consumption of **released** broker images (`ghcr.io/ksteffe/pade-broker:vX.Y.Z`) after Milestone G

### `after-certainty` (external)

- `DevelopmentSession` Intent for the site
- Cursor Cloud / iOS agent environment setup
- Product GA4 tooling, analytics commands/scripts, use of returned credentials
- Application start command, local port, Cloudflare preview command, stable preview hostname
- Consumption of **released** PADE CLI (`PADE_VERSION=vX.Y.Z`) after Milestone G

Do not move these external product responsibilities into PADE merely because PADE enables them.

### Google Analytics ownership

| Owner | Responsibility |
|-------|----------------|
| **PADE** (`examples/providers/google-analytics/`) | Non-normative **reference provider** that exercises the generic contract and performs broker-side derivation for dogfood. Not part of the PADE standard. |
| `pade-broker-deployment` | Private durable Google authority / secret-store bootstrap and broker policy wiring |
| `after-certainty` | Product GA4 tooling; analytics commands; use of returned Material |
| **PADE core** | Remains vendor-neutral; sees only the generic provider request/result contract |

### Cloudflare ownership

| Owner | Responsibility |
|-------|----------------|
| `pade-broker-deployment` | Private capability → Cloudflare credential binding |
| `after-certainty` | App dev command; `cloudflared` install; preview workflow; hostname configuration |
| **PADE** | No Cloudflare-specific APIs or tunnel provisioning. Track the **endpoint declaration** question (Milestone K) as a portable protocol concern if needed. Post-release (Milestone J). |

## Forward milestones

Completed learning dogfood (0–9 / 9b) is summarized under [Historical dogfood milestones](#historical-dogfood-milestones). It is not reopened as the live plan.

### Mapping from previous roadmap letters

Previous letters (release-first A–I) are **superseded** by the table below:

| Previous | Now |
|----------|-----|
| Old A (release foundation) | **G** (gated on provider dogfood) |
| Old B (released broker deploy) | **H** |
| Old C (released Consumer dogfood) | **I** |
| Old D (external GA via direct Material) | Absorbed into **D–E** (derived reference provider) + product use in **L** |
| Old E (Cloudflare preview) | **J** |
| Old F (endpoint decision) | **K** |
| Old G (Cursor iOS workflow) | **L** |
| Old H (post-dogfood evaluation) | **M** |
| Old I (derived fulfillment hardening) | **B–E** (moved **before** initial release) |

### Pre-release (required for `v0.1.0`)

| Milestone | Focus | Where work happens |
|-----------|--------|--------------------|
| **A — Broker dogfood baseline** | Keep DevelopmentSession + broker + GitHub **direct materialization** valid and minimal | PADE repo (mostly DONE) |
| **B — Generic provider contract** | Minimal portable fulfill/derive seam | PADE repo |
| **C — Dogfood provider binding** | One binding sufficient for dogfood (exec is a candidate, not frozen) | PADE repo |
| **D — GA reference provider** | `examples/providers/google-analytics/` exercises the contract | PADE repo (non-normative example) |
| **E — Derived-credential cloud-agent proof** | Durable Google authority stays broker-side; session uses derived Material against GA API | PADE + broker deploy config + cloud agent |
| **F — Spec/docs tighten from dogfood** | Capture B–E learnings; no premature Intent redesign | PADE repo |
| **G — Initial versioned release (`v0.1.0`)** | CLI + broker artifacts; **gated on A–F**; must show stages 1 and 2 | PADE repo |

### Post-release (preserved; reordered)

| Milestone | Focus | Where work happens |
|-----------|--------|--------------------|
| **H — Released broker deployment** | Private deploy consumes `ghcr.io/ksteffe/pade-broker:vX.Y.Z` | `pade-broker-deployment` |
| **I — Released Consumer dogfood** | Cursor Cloud uses released `pade` against real deployed broker | External + PADE artifacts |
| **J — External preview / Cloudflare dogfood** | App + tunnel via generic Material; long-running `pade exec` | External + possible generic exec fix |
| **K — Endpoint decision** | Portable endpoint declaration decision gate | Architecture decision |
| **L — Full Cursor iOS workflow** | Analytics → edit → run → preview acceptance | External acceptance |
| **M — Post-dogfood protocol evaluation** | Only generic deficiencies return to PADE | Conditional |

```text
A (GitHub direct Material)
  → B (provider contract)
  → C (dogfood binding)
  → D (GA reference provider)
  → E (cloud-agent derived proof)
  → F (docs tighten)
  → G (v0.1.0)
  → H (released broker deploy)
  → I (released Consumer)
  → J (Cloudflare preview)
  → K (endpoint decision)
  → L (Cursor iOS acceptance)
  → M (post-dogfood evaluation)
```

### Milestone A — Broker dogfood baseline

**Goal:** Preserve and keep healthy the existing DevelopmentSession / broker / GitHub direct-materialization dogfood.

- Intent requests a capability (`github.user.read`)
- Cloud agent (or fake OIDC in CI) communicates with the broker
- Broker obtains an existing credential from a configured source
- Capability is delivered as `Material` to the development environment
- `make dogfood-broker` / Stage B remain the minimal proof

Mostly **DONE**. Do not remove or replace this path.

### Milestone B — Generic provider contract

**Goal:** Define and dogfood a minimal generic PADE provider contract.

Semantic responsibility (approximate):

> Given a capability request and authorized DevelopmentSession context, fulfill that capability using a configured external implementation.

An implementation may retrieve durable authority, derive a temporary credential, perform an identity exchange, mint a scoped token, return an existing secret unchanged, or (later) mediate without returning credential material.

Reuse existing terms (`Provider`, **materialization**, `Material`) where they fit. If no existing term clearly names the external fulfill/derive seam, describe it generically and treat **naming** as an open design question. Do **not** freeze the exact protocol in this roadmap document alone—dogfood drives the shape.

This is **not** “an arbitrary hook.”

### Milestone C — Dogfood provider binding

**Goal:** Implement one **implementation binding** sufficient for dogfood of the contract from Milestone B.

Candidates include subprocess/exec, local plugin, HTTP, gRPC, or a separately packaged provider. **Do not commit the roadmap to all of them.** An exec/subprocess binding may be an attractive first experiment (any language), but distinguish:

- **Provider contract** — semantic abstraction
- **Exec/subprocess** — one possible binding

### Milestone D — Google Analytics reference provider

**Goal:** Add an in-tree reference provider that exercises the contract with a real external system involving broker-side credential derivation.

Planned location: **`examples/providers/google-analytics/`** (clearly non-core; matches existing `examples/` layout). Avoid an `extensions/` directory name.

Explicit statement:

> In-tree reference providers exist for dogfooding and illustrating the provider contract. Their presence does **not** make the vendor, API, or capability part of the PADE standard.

The provider contains all Google-specific behavior needed to derive a temporary usable credential. PADE core sees only the generic provider request/result contract. Do not select the exact Google authentication mechanism in this planning document unless later implementation dogfood establishes one.

### Milestone E — Derived-credential cloud-agent proof

**Goal:** Prove the end-to-end derived path from a cloud-agent DevelopmentSession:

```text
durable Google authority
        ↓
secret source / broker configuration
        ↓
PADE broker
        ↓
Google Analytics reference provider
        ↓
short-lived / session-scoped credential (Material)
        ↓
cloud-agent DevelopmentSession
        ↓
Google Analytics API
```

Acceptance:

- Durable credential remains broker-side
- Provider is vendor-specific; PADE core remains generic
- DevelopmentSession does not need to know whether the credential came from a service account, OAuth exchange, identity federation, or another Google mechanism
- Provider lifecycle and error behavior are documented enough for a first release

### Milestone F — Spec/docs tighten from dogfood

**Goal:** Update draft specs, SECURITY notes, and examples based on what B–E learned.

Do **not** redesign DevelopmentSession / Intent around Runtime Conditions or invent Grant/Lease without evidence. Keep changes proportional to dogfood.

### Milestone G — Initial versioned release (`v0.1.0`)

**Goal:** External consumers pin `PADE_VERSION=vX.Y.Z` instead of arbitrary `main` commits—**only after** A–F prove both direct and derived fulfillment.

**SemVer strategy:** Pre-1.0 SemVer. Breaking changes allowed with minor bumps while major remains `0`. Initial release: **`v0.1.0`**.

#### Release gate (required)

**Stage 1 — existing/basic dogfood**

- DevelopmentSession/Intent can request a capability
- Cloud agent can communicate with the broker
- Broker can obtain an existing credential from a configured source
- Capability can be delivered to the development environment
- Existing GitHub-token dogfood remains valid and minimal

**Stage 2 — provider dogfood**

- Broker can invoke an external/provider implementation through a generic contract
- Provider implementation can remain vendor-specific while PADE core remains generic
- Google Analytics reference provider exercises that contract
- Durable Google authority stays broker-side
- Provider returns a derived/session-scoped usable credential
- Cloud development session can successfully use it against the Google Analytics API
- Provider lifecycle/error behavior is sufficiently documented for a first release

**Not required for `v0.1.0`:** mediated capabilities; full provider ecosystem; every future binding; Intent redesign; Runtime Conditions integration; Cloudflare/preview completion.

#### Release artifacts (implement at G)

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
- Automatic version bumping remains deferred.

#### Artifact contract (PADE publishes; externals consume)

**`pade-broker-deployment` should eventually consume:**

```text
ghcr.io/ksteffe/pade-broker:vX.Y.Z
```

without cloning or building PADE source. Operators must be able to identify PADE version, source commit, and image digest.

**Cloud development environments should install** `pade vX.Y.Z` from a GitHub Release asset.

### Milestone H — Released broker deployment

External: private broker deployment pulls the released GHCR image, mounts policy/bindings/secrets, and runs with Cloud Run–compatible transport (`PORT`, trusted TLS termination).

PADE acceptance: image runs as already smoked by `make smoke-broker-container`; no PADE protocol change required.

### Milestone I — Released Consumer dogfood

External Cursor Cloud (or equivalent) installs released `pade`, points agent bindings at the real broker, and resolves a capability end-to-end with Cursor OIDC.

PADE acceptance: released CLI talks to released broker using the existing wire protocol.

### Milestone J — External preview / Cloudflare dogfood

after-certainty runs its normal application and establishes a Cloudflare Tunnel using **generic** PADE-brokered Material (tunnel token or equivalent), plus **repo-local** start command / port / `cloudflared` invocation.

Evaluate long-running `pade exec` (signals, exit codes, Material lifetime, redaction). Fix gaps as **generic execution** improvements—not Cloudflare support.

Preserved from the prior roadmap; intentionally **after** the initial release.

### Milestone K — Endpoint decision

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

Keep **application startup** separate from endpoint intent (`package.json` / Makefile / Dev Container vs possible future Intent endpoints vs capability authority).

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

### Milestone L — Full Cursor iOS workflow

Acceptance story:

From Cursor iOS, launch a fresh after-certainty Cloud Agent and ask it to inspect Google Analytics, make an appropriate change, run the application, and provide a public preview—**without manually copying credentials into the agent**.

Expected path uses **released** CLI/broker (post-G) and derived Material where the configured provider supports it:

```text
released PADE CLI
    ↓
DevelopmentSession
    ↓
Cursor workload identity
    ↓
released PADE broker
    ↓
provider fulfillment (derived GA credential / tunnel Material)
    ↓
repo-owned tooling (GA scripts, app start, cloudflared)
```

PADE core remains unaware of the downstream vendors.

### Milestone M — Post-dogfood protocol evaluation

Only **generic** deficiencies discovered during real use return to the PADE repository (Intent, Consumer, Broker, or reference execution behavior). Vendor-specific work stays in external repos or `examples/providers/`.

## Capability naming (exploratory)

External dogfood may use names resembling `analytics.google.read` or `preview.http`.

- Do **not** create a capability registry in PADE.
- Do **not** standardize vendor capability names in this roadmap pass.
- Reference-provider capability names remain exploratory and non-normative.

Post-dogfood questions (track only):

- Do capability names need namespaces?
- Does `access` add useful semantics?
- Should common capabilities eventually have standardized meanings?

## Open design questions

Track these without freezing Intent or Broker wire formats prematurely.

1. **Provider naming** — What should the external fulfill/derive seam be called in specs vs reference code (`Provider`, materialization adapter, issuer, …)?
2. **Provider contract shape** — Minimal request/result fields for capability + DevelopmentSession context → Material (or later mediation)?
3. **First provider binding** — Exec/subprocess vs HTTP vs gRPC vs plugin for the first dogfood—choose during Milestone C, not in this planning pass.
4. **Credential lifetime / expiry semantics** — Does derived Material need explicit expiry/revocation metadata, or does that belong to a future Grant/Lease model?
5. **Fulfillment pipeline naming** — Keep illustrative SOURCE / PROVIDER / DERIVATION / DELIVERY, or stick strictly to Provider / materialization / Material?
6. **Mediated capabilities** — Does credential-less exercise require Consumer protocol changes beyond today’s resolve → Material path?
7. **Runtime Conditions (CNCF)** — Active discussion with Runtime Conditions maintainers. Emerging possibility: Runtime Conditions describe requirements of a runtime *thing* (application runtime, development environment, Dev Container, code session, agent sandbox), while PADE creates/fulfills an identity-bound DevelopmentSession and handles downstream capability fulfillment. **Do not** claim adopt/extend/replace/integrate yet. **Do not** redesign Intent while this is open.
8. **Hierarchical profiles** — Emerging idea that a development runtime could add requirements atop those already required by the application/codebase. Record only; no PADE schema work yet.
9. **Endpoint declaration** — See Milestone K.
10. **Capability vocabulary** — See [Capability naming](#capability-naming-exploratory).

## Deferred until after the initial release / full workflow

Explicitly deferred (not blockers for `v0.1.0` unless noted otherwise):

- **Mediated capabilities** (Consumer never receives credential Material) — after stage 2 is proven
- Endpoint schema implementation **unless** Milestone K decides it is required
- Grant / Lease model
- Additional vendor reference providers / integration catalog
- Extra provider bindings beyond the first dogfood binding
- Provider packaging / distribution choices beyond what dogfood needs
- Dynamic preview provisioning by the broker
- Multiple simultaneous previews / branch-specific preview URLs
- Capability registry / standardized capability naming
- frp or other tunnel implementations inside PADE
- Additional workload identity adapters (beyond Cursor OIDC reference)
- Codex / Claude (or other agent runtime) support as PADE features
- Managed broker products
- PADE website
- Package-manager distribution (Homebrew, etc.)
- Automatic release versioning
- Supply-chain signing / provenance beyond checksums
- Standards / governance work
- Runtime Conditions adoption or hierarchical-profile Intent changes
- JTI replay store, multi-tenant broker hosting, DB-backed policy (still deferred spike hardening)
- Cloudflare / preview completion (Milestone J — post-release, preserved)

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

Earlier README “9+ / Later” rows and the previous release-first A–I ordering (derived fulfillment as post-release Milestone I) are **superseded** by Milestones **A–M** in this document.
