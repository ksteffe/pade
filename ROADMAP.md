# PADE Roadmap

**Authoritative planned work** for this repository. Specs under [`spec/`](spec/README.md) remain the interoperability contract; [README.md](README.md) is the project entrypoint; [RFC.md](RFC.md) / [DESIGN.md](DESIGN.md) retain architectural and implementation history.

This document is planning only until milestones are implemented. It does not change Intent schema, Consumer/Broker wire behavior, or reference code by itself.

## Purpose and principles

PADE should prove **generic authority brokering** once. Downstream systems (Google Analytics, Cloudflare Tunnel, GitHub, …) should work because the mechanism is generic—not because PADE knows about them.

Before the first meaningful release (`v0.1.0`), PADE must prove that a broker can fulfill a DevelopmentSession capability through an **independently implemented provider** that derives a session-scoped credential—**twice**, with two different vendors—without vendor-specific behavior in PADE core.

PADE core should understand only something equivalent to:

> An authorized DevelopmentSession requests a capability, and a configured provider fulfills it.

Governing principles:

1. Keep PADE deliberately small and vendor-neutral.
2. Compose mature tools; do not reimplement workspace lifecycle, IAM, or secret stores.
3. Treat `apiVersion: pade.local/v1alpha1` / `kind: DevelopmentSession` and the Intent / Consumer / Broker specs as authoritative.
4. Prefer dogfood evidence before extending Intent or inventing Grant/Lease protocols.
5. Do not reclaim responsibility for starting and orchestrating development environments unless real dogfood proves that necessary.
6. PADE standardizes the **provider seam**, not every provider implementation. A provider should be architecturally removable from PADE core. Vendor-specific business logic (GitHub App auth/installation tokens, Google Analytics auth, Google service accounts/OAuth scopes, AWS STS, Keeper/1Password/Vault derivation details, provider-specific field names or credential formats, …) belongs in independently implemented providers—not in PADE core (`cmd/`, portable `internal/` packages).
7. Brokers **SHOULD** prefer session-scoped, short-lived, or otherwise **derived** credentials over delivering durable source credentials when the configured provider supports such derivation. Direct durable-secret **materialization** remains a valid interoperability mechanism where necessary; PADE must work with systems that only expose static credentials and must not pretend every capability can be fulfilled without exposing credential material.
8. Prove the provider seam with **two** non-normative reference providers (**GitHub App**, then **Google service-account OAuth** as a second structurally different test) **before** cutting the initial versioned release. Accumulating many integrations is less important than proving third parties can implement providers without modifying PADE itself.
9. **Hard stopping rule after the second provider:** once two structurally different derived-token providers prove the seam, do **not** add in-tree providers merely to demonstrate vendor compatibility. A third provider belongs only if it exposes a **new generic protocol, lifecycle, or security property**—not another OAuth/API variant for its own sake.

Document roles:

| Document | Role |
|----------|------|
| [README.md](README.md) | Project entrypoint |
| [`spec/`](spec/README.md) | Interoperability contract |
| [RFC.md](RFC.md) | Architectural evolution (history + current framing) |
| [DESIGN.md](DESIGN.md) | Reference implementation design history |
| **ROADMAP.md** (this file) | Planned work |

## Why two derived-token providers before v0.1.0

Google is **not** in PADE because PADE needs Google Analytics support. The in-tree Google example exists as a **second architectural and security test** of the broker-side credential-derivation seam—not to expand PADE’s integration catalog.

The first reference provider (GitHub App) could still leave a **GitHub-App-shaped** abstraction with a generic name. The second provider must answer:

> Did PADE create a genuinely generic broker-side credential-derivation seam, or did we accidentally design around one vendor’s token exchange?

Both in-tree providers are **reference / architectural tests**. They are **non-normative**, **architecturally removable**, and do **not**:

- make GitHub or Google part of the PADE protocol;
- create a capability registry;
- imply first-class GitHub or Google Analytics product support in PADE.

Portable Intent stays vendor-neutral. Durable authority stays broker-side. When downstream systems support temporary issuance, the Consumer should receive **narrower, shorter-lived** credentials—not the durable private key or long-lived secret.

**Release-gate rationale:** The second derived-token provider is not intended to expand PADE’s integration catalog. It is an architectural test that the broker-side provider seam can convert **different forms** of durable authority into short-lived session credentials without encoding vendor-specific behavior in PADE core.

**Security motivation (non-absolute):** Keeping durable authority broker-side and issuing shorter-lived credentials **reduces the blast radius** of credential exposure in ephemeral and agent-driven development environments when temporary issuance is available. Agent and cloud environments may have unexpected logging, trace capture, or isolation failures. Short-lived credentials still require narrow scoping, protection in the Consumer, and downstream resource authorization; they **do not eliminate** compromise risk.

See [Structurally different derivation flows](#structurally-different-derivation-flows) and [Fulfillment maturity (Material path)](#fulfillment-maturity-material-path).

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
| External/independently packaged provider seam | PARTIAL | Draft `provider: exec` contract + adapter (`docs/provider-contract.md`, `internal/binding/exec`); stub dogfood in CI |
| GitHub App derived-credential dogfood | PARTIAL | Real App JWT → installation token in `examples/providers/github/` (httptest + fake CI); preferred repo-scoped dogfood path landed; live App install still optional/local (Milestone E) |
| Google OAuth derivation dogfood (second provider test) | PARTIAL | SA JWT → access token in `examples/providers/google-analytics/` (httptest + fake CI); directory name is dogfood-only—not GA product support; live property proof optional/local (Milestone F) |
| Two-provider same-seam validation | DONE (CI) | `make dogfood-exec-provider-two` — GitHub + GA on `provider: exec` without core vendor fields (Milestone G) |
| Arbitrary Material / env injection | DONE (env maps) | `Material.Env map[string]string` only; sufficient for token/API credentials |
| Structured / multiline credential Material | PARTIAL | String values may contain newlines; not dogfooded; **no anticipated PADE work** until a proven deficiency |
| `pade exec` | DONE | Process-scoped resolve → inject → wait → discard |
| Child exit-code propagation | DONE | Maps to process exit |
| Output redaction | DONE | Exact-match secret redaction on child stdout/stderr (defense in depth) |
| Signal forwarding (SIGINT / SIGTERM → child) | MISSING | No explicit forward; evaluate in Milestone L if needed |
| Long-running child-process behavior | PARTIAL / UNKNOWN | Wait + stream works; process-group / signal behavior needs dogfood |
| Remote broker endpoint / audience configuration | DONE | Local bindings only (not in Intent) |
| Containerized `pade-broker` | DONE | Root [`Dockerfile`](Dockerfile); `make smoke-broker-container` |
| Cloud Run–compatible transport mode | DONE | `PORT` + `-tls-termination=proxy` |
| Fake broker dogfood (GitHub PAT direct Material) | DONE | `make dogfood-broker` (CI smoke); Stage B real Cursor OIDC (not CI) — **stage 1 baseline**, not preferred final pre-release GitHub dogfood |
| Version reporting (`pade` / `pade-broker --version`) | MISSING | Part of Milestone I |
| GitHub Release artifacts | MISSING | Part of Milestone I |
| Broker container publishing (GHCR) | MISSING | Part of Milestone I |

**Genuinely missing PADE-repository work** before the first release:

1. Finish preferred GitHub / live App or live GA proofs where useful (E/F remainders) — offline preferred paths already landed
2. Spec/docs tightening from D–G dogfood (Milestone H)
3. Versioned release foundation gated on A–H (Milestone I)

**Landed for B–G:** draft exec contract, `provider: exec` binding, stub + GitHub App + Google service-account reference providers (fake CI + httptest-tested real derivation), two-provider same-seam dogfood. Targets: `make dogfood-exec-provider{,-github,-ga,-two}`.

Do not roadmap work that is already DONE.

## Direct materialization vs preferred GitHub dogfood

### Stage 1 baseline (PAT / direct materialization) — keep

The GitHub-token broker path already proves **direct materialization** (fulfillment maturity stage 1):

```text
Current (baseline / simple path):
PAT
  ↓
secret store
  ↓
broker
  ↓
DevelopmentSession
```

Evidence: `make dogfood-broker`, Stage B (`make dogfood-broker-stage-b`), demo Intent `github.user.read`, and related provider dogfoods. This path remains **documented and useful** for showing the simple interoperability mechanism and is a **required** stage-1 baseline for `v0.1.0`.

It is **not** the preferred final pre-release GitHub dogfood. Do **not** remove the PAT example if it remains useful for the simple path.

### Preferred pre-release GitHub dogfood (derived)

```text
Preferred:
GitHub App private key
  ↓
trusted broker-side source
  ↓
PADE broker
  ↓
GitHub reference provider
  ↓
short-lived GitHub App installation token
  ↓
cloud DevelopmentSession
  ↓
repository-scoped GitHub operation
```

The durable App private key must remain broker-side. The DevelopmentSession receives only the derived installation token.

Do **not** add Cloudflare or additional vendor integrations merely to repeat direct Material delivery. Two **non-normative** in-tree reference providers (`examples/providers/github/`, `examples/providers/google-analytics/`) dogfood the **provider contract and derivation**—not to make GitHub or Google Analytics part of the PADE standard. The Google directory name reflects a convenient dogfood validation hook only; PADE does not ship Google Analytics as an application feature.

## Material vs Endpoint vs Grant

Three emerging concepts—only **Material** is implemented today:

| Concept | Meaning | Status |
|---------|---------|--------|
| **Material** | Authority returned to a child process (today: env map). Examples: API credential, tunnel token. | Current protocol / reference implementation |
| **Endpoint** | Potential portable description of a **local** service a capability may act upon. Example: HTTP on port 3000. | Unevaluated; see Milestone M |
| **Grant / Lease** | Potential future broker result for dynamically provisioned resources (preview URL, ephemeral DB, expiration/revocation). | Deferred; no schema or type |

Post-release Cloudflare / preview dogfood (Milestone L) should first test:

```text
existing Material
+
repo-local service startup
+
possibly future Endpoint declaration (only if dogfood requires it)
```

Do **not** implement Grant/Lease before the initial release.

### Fulfillment maturity (Material path)

**Architectural framing only** — these three stages are **not** represented in the Intent schema and are **not** normative PADE protocol terms. They describe how fulfillment may evolve; only stages 1 and 2 are required before `v0.1.0`.

#### Stage 1 — Pass-through material (exists today)

```text
long-lived credential
        ↓
      broker
        ↓
same credential
        ↓
      Consumer
```

Direct materialization: the broker retrieves a secret from a source store and delivers it as `Material` unchanged. **Today** — GitHub PAT dogfood (Milestone A); **required** for `v0.1.0`.

#### Stage 2 — Derived / session-scoped credential (pre-release validation target)

```text
durable authority
        ↓
      broker
        ↓
provider-specific derivation/exchange
        ↓
short-lived credential
        ↓
      Consumer
```

Durable authority remains broker-side. A configurable external provider derives or issues a temporary credential; the Consumer receives only that derived `Material`. **Milestones B–G** — **required** for `v0.1.0`, demonstrated by **both** reference providers (GitHub App first, Google service-account OAuth second).

#### Stage 3 — Mediated capability (future / exploratory — NOT current behavior)

```text
durable authority
        ↓
      broker
        ↓
broker performs operation
        ↓
result only
        ↓
      Consumer
```

The Consumer exercises a capability through the broker/provider **without** receiving the underlying credential. **Future direction only** — do **not** implement, document as current behavior, or promote as part of `v0.1.0`.

| Stage | Summary | Status |
|-------|---------|--------|
| **1. Direct materialization** | Pass-through: same credential reaches the Consumer | **Today**; required for `v0.1.0` |
| **2. Derived / session-scoped** | Broker-side derivation → short-lived credential → Consumer | **Pre-release proof** (Milestones B–G); required for `v0.1.0` |
| **3. Mediated capabilities** | Broker performs operation; Consumer gets result only | **Future / exploratory**; not `v0.1.0` |

The initial release must demonstrate stages **1 and 2**. Stage 3 remains future work.

#### Structurally different derivation flows

The two pre-release reference providers exercise **different** derivation mechanics on the **same** generic seam (`provider: exec`). Neither is an integration goal.

**GitHub App (first architectural test):**

```text
GitHub App private key
        ↓
signed JWT
        ↓
GitHub installation-token exchange
        ↓
short-lived installation token
        ↓
Material
```

**Google service account (second architectural test):**

```text
Google service-account private key
        ↓
signed assertion
        ↓
OAuth token exchange
        ↓
short-lived Google access token
        ↓
Material
```

The Google path uses Google’s generic OAuth2 JWT-bearer grant. The in-tree example under `examples/providers/google-analytics/` uses an Analytics-related scope and a minimal Admin API call **only as a dogfood validation hook**—not because PADE owns GA4 report logic, dimensions/metrics, property semantics, GA client libraries, or after-certainty analytics tooling. Those belong in downstream applications such as `after-certainty`.

#### DevelopmentSession relationship

Preserve the current DevelopmentSession direction:

- **Durable authority** belongs to an external identity / provider / secret system.
- A **capability** is authorized for a particular **DevelopmentSession**.
- The **Consumer** exercises that capability.
- The **Broker** / **Provider** determines the safest available **materialization** (or mediation) mechanism.

The DevelopmentSession Intent **must not** need to know whether fulfillment ultimately came from a PAT, a GitHub App installation token, an OAuth access token, OIDC federation, a static secret, or a broker-mediated operation. That remains a fulfillment concern. Do not redesign Intent around CNCF Runtime Conditions while that discussion is open.

#### Conceptual fulfillment pipeline (illustrative names)

Stage labels such as SOURCE / PROVIDER / DERIVATION / DELIVERY are **illustrative only**—not normative API terms. Prefer existing PADE vocabulary (`Provider`, **materialization**, `Material`) until naming is decided (see [Open design questions](#open-design-questions)).

```text
durable authority
      ↓
source / secret store
      ↓
PADE broker
      ↓
configured provider
      ↓
derived/session-scoped result (or unchanged static secret when derivation is unavailable)
      ↓
DevelopmentSession
```

The provider contract should eventually allow implementations to derive short-lived credentials, mint scoped tokens, exchange identities, pass static credentials through when derivation is unavailable, and eventually mediate capabilities without exposing credential material. Do **not** freeze the exact API shape in this roadmap document.

The provider mechanism is a **semantic fulfill/derive contract**, not an arbitrary “hook.” An exec/subprocess integration may be a useful first **binding**; that is an implementation choice, not the definition of the abstraction.

## Ownership boundaries

### PADE repository (this repo)

Near-term (before `v0.1.0`):

- Keep PAT direct-materialization broker dogfood healthy as stage-1 baseline (Milestone A)
- Define minimal generic provider contract + dogfood binding (Milestones B–C)
- In-tree **non-normative** GitHub App reference provider at `examples/providers/github/` and migrate preferred GitHub dogfood (Milestones D–E)
- In-tree **non-normative** Google service-account reference provider at `examples/providers/google-analytics/` — second structurally different derivation test (Milestone F)
- Two-provider same-seam architectural test (Milestone G)
- Spec/docs tighten + versioned CLI/broker release (Milestones H–I), gated on A–G

PADE should **not** put into core:

- GitHub App authentication, installation identification, token issuance, or repository/permission policy fields as protocol semantics
- Google service-account / OAuth derivation logic, GA4 report logic, analytics dimensions/metrics, property semantics, or GA client behavior in PADE core
- Cloudflare-specific APIs or tunnel provisioning
- Application start/stop orchestration (DevPod / repo tooling owns lifecycle)
- A catalog of vendor integrations

In-tree reference providers under `examples/providers/` exist for **dogfooding and illustrating the provider contract**. They are **non-normative** and **architecturally removable** from PADE core. Their presence does **not** make the vendor, API, or capability part of the PADE standard. Prefer `examples/providers/` over an `extensions/` directory name (avoids collision with CNCF Runtime Conditions “extension” terminology).

### `pade-broker-deployment` (external)

- Google Cloud Run (or equivalent) hosting of the broker
- Private broker policy and secret-store bootstrap (durable authority, including GitHub App private key and Google durable credentials)
- Capability → source/provider configuration
- Consumption of **released** broker images (`ghcr.io/ksteffe/pade-broker:vX.Y.Z`) after Milestone I

### `after-certainty` (external)

- `DevelopmentSession` Intent for the site
- Cursor Cloud / iOS agent environment setup
- Product GA4 tooling, analytics commands/scripts, use of returned credentials
- Application start command, local port, Cloudflare preview command, stable preview hostname
- Consumption of **released** PADE CLI (`PADE_VERSION=vX.Y.Z`) after Milestone I

Do not move these external product responsibilities into PADE merely because PADE enables them.

### GitHub ownership

| Owner | Responsibility |
|-------|----------------|
| **PADE** (`examples/providers/github/`) | Non-normative **reference provider / architectural test**: GitHub App auth, installation identification, token issuance, repo/permission restriction, expiry handling. Not part of the PADE standard. Not a normative PADE identity or GitHub integration requirement. |
| `pade-broker-deployment` | Private durable GitHub App private key / secret-store bootstrap and broker policy wiring |
| Demo / dogfood scripts | Repository-scoped validation matching granted permissions (not `/user` whoami) |
| **PADE core** | Remains vendor-neutral; sees only the generic provider request/result contract |

### Google service-account reference provider (second test)

The directory is named `google-analytics/` for dogfood convenience only. PADE does **not** ship Google Analytics as an application feature.

| Owner | Responsibility |
|-------|----------------|
| **PADE** (`examples/providers/google-analytics/`) | Non-normative **reference provider / architectural test**: broker-side service-account → OAuth access-token derivation on the same generic seam as GitHub. Not part of the PADE standard. |
| `pade-broker-deployment` | Private durable Google service-account / secret-store bootstrap and broker policy wiring |
| `after-certainty` | Product GA4 tooling; analytics commands; dimensions/metrics; use of returned Material |
| **PADE core** | Remains vendor-neutral; sees only the generic provider request/result contract |

PADE must **not** own or contain: GA4 report logic, GA dimensions/metrics, Analytics property semantics, GA client libraries, or after-certainty analytics tooling. The in-tree provider demonstrates **credential derivation only**; a minimal downstream API call in dogfood validates the derived token—not GA product behavior inside PADE.

### Cloudflare ownership

| Owner | Responsibility |
|-------|----------------|
| `pade-broker-deployment` | Private capability → Cloudflare credential binding |
| `after-certainty` | App dev command; `cloudflared` install; preview workflow; hostname configuration |
| **PADE** | No Cloudflare-specific APIs or tunnel provisioning. Track the **endpoint declaration** question (Milestone M) as a portable protocol concern if needed. Post-release (Milestone L). |

## Forward milestones

Completed learning dogfood (0–9 / 9b) is summarized under [Historical dogfood milestones](#historical-dogfood-milestones). It is not reopened as the live plan.

### Mapping from previous roadmap letters

Previous letters after PR #31 (GA-first A–M) are **superseded**:

| Previous (PR #31) | Now |
|-------------------|-----|
| A (PAT baseline) | **A** (unchanged role) |
| B (provider contract) | **B** |
| C (dogfood binding) | **C** |
| D–E (GA provider + proof) | Split/replaced by **D–E** (GitHub App first) then **F–G** (GA + same-seam test) |
| F (docs tighten) | **H** |
| G (v0.1.0) | **I** |
| H (released broker deploy) | **J** |
| I (released Consumer) | **K** |
| J (Cloudflare preview) | **L** |
| K (endpoint decision) | **M** |
| L (Cursor iOS) | **N** |
| M (post-dogfood evaluation) | **O** |

### Pre-release (required for `v0.1.0`)

| Milestone | Focus | Where work happens |
|-----------|--------|--------------------|
| **A — Broker dogfood baseline** | Keep PAT **direct materialization** healthy as stage-1 baseline (not preferred final GitHub dogfood) | PADE repo (mostly DONE) |
| **B — Generic provider contract** | Minimal portable fulfill/derive seam | PADE repo — **draft landed** (`docs/provider-contract.md`) |
| **C — Dogfood provider binding** | One binding sufficient for dogfood (`provider: exec`) | PADE repo — **landed** (other bindings still open) |
| **D — GitHub App reference provider** | `examples/providers/github/` — App private key broker-side → installation token | PADE repo — **landed** (fake CI + httptest-tested real path) |
| **E — GitHub dogfood migration** | Prefer derived installation token; repo-scoped validation (not `/user`) | PADE — **partial** (repo-meta script + fake preferred path); live App optional |
| **F — Google service-account reference provider (second test)** | `examples/providers/google-analytics/` — structurally different OAuth derivation on same seam | PADE repo — **landed** (fake CI + httptest-tested SA→token); live optional |
| **G — Two-provider architectural test** | Same `provider: exec` seam for both; no vendor leakage into core | PADE repo — **landed** (`make dogfood-exec-provider-two`) |
| **H — Spec/docs tighten from dogfood** | Capture D–G learnings; no premature Intent redesign | PADE repo |
| **I — Initial versioned release (`v0.1.0`)** | CLI + broker artifacts; **gated on A–H**; stages 1 and 2 via both providers | PADE repo |

### Post-release (preserved; renumbered)

| Milestone | Focus | Where work happens |
|-----------|--------|--------------------|
| **J — Released broker deployment** | Private deploy consumes `ghcr.io/ksteffe/pade-broker:vX.Y.Z` | `pade-broker-deployment` |
| **K — Released Consumer dogfood** | Cursor Cloud uses released `pade` against real deployed broker | External + PADE artifacts |
| **L — External preview / Cloudflare dogfood** | App + tunnel via generic Material; long-running `pade exec` | External + possible generic exec fix |
| **M — Endpoint decision** | Portable endpoint declaration decision gate | Architecture decision |
| **N — Full Cursor iOS workflow** | Analytics → edit → run → preview acceptance | External acceptance |
| **O — Post-dogfood protocol evaluation** | Only generic deficiencies return to PADE | Conditional |

```text
A (PAT direct Material baseline)
  → B (provider contract)
  → C (dogfood binding)
  → D (GitHub App reference provider)
  → E (migrate GitHub dogfood to installation token)
  → F (GA reference provider)
  → G (same-seam architectural test)
  → H (docs tighten)
  → I (v0.1.0)
  → J (released broker deploy)
  → K (released Consumer)
  → L (Cloudflare preview)
  → M (endpoint decision)
  → N (Cursor iOS acceptance)
  → O (post-dogfood evaluation)
```

**After `v0.1.0`, move outward** — do not keep adding in-tree vendor examples instead of external dogfood:

```text
released PADE (v0.1.0)
    ↓
pade-broker-deployment
    ↓
real Cloud Run broker
    ↓
after-certainty
    ├── Google Analytics usage (product tooling — not PADE core)
    └── Cloudflare preview
```

The Google reference provider must **not** become an excuse to keep building vendor examples inside PADE. Real product dogfood (GA usage, preview tunnels) belongs in external repos after released artifacts exist (Milestones J–N).

### Milestone A — Broker dogfood baseline

**Goal:** Preserve and keep healthy the existing DevelopmentSession / broker / GitHub **PAT direct-materialization** dogfood as the stage-1 baseline.

- Intent requests a capability (`github.user.read`)
- Cloud agent (or fake OIDC in CI) communicates with the broker
- Broker obtains an existing credential from a configured source
- Capability is delivered as `Material` to the development environment
- `make dogfood-broker` / Stage B remain the minimal stage-1 proof

Mostly **DONE**. Do not remove this path. It is useful for the simple path and for systems that only expose static credentials. It is **not** the preferred final pre-release GitHub derived dogfood (see Milestones D–E).

### Milestone B — Generic provider contract

**Status:** Draft contract documented in [`docs/provider-contract.md`](docs/provider-contract.md). Naming remains an open question; dogfood continues to drive the shape.

**Goal:** Define and dogfood a minimal generic PADE provider contract.

Semantic responsibility (approximate):

> Given a capability request and authorized DevelopmentSession context, fulfill that capability using a configured external implementation.

An implementation may retrieve durable authority, derive a temporary credential, perform an identity exchange, mint a scoped token, return an existing secret unchanged, or (later) mediate without returning credential material.

Reuse existing terms (`Provider`, **materialization**, `Material`) where they fit. If no existing term clearly names the external fulfill/derive seam, describe it generically and treat **naming** as an open design question. Do **not** freeze the exact protocol in this roadmap document alone—dogfood drives the shape.

This is **not** “an arbitrary hook.”

### Milestone C — Dogfood provider binding

**Status:** First binding landed as `provider: exec` ([`internal/binding/exec`](internal/binding/exec)). Dogfood: `make dogfood-exec-provider`. Other bindings (HTTP, gRPC, plugin) remain open options—not committed.

**Goal:** Implement one **implementation binding** sufficient for dogfood of the contract from Milestone B.

Candidates include subprocess/exec, local plugin, HTTP, gRPC, or a separately packaged provider. **Do not commit the roadmap to all of them.** Distinguish:

- **Provider contract** — semantic abstraction
- **Exec/subprocess** — the first dogfood binding (not the definition of the abstraction)

### Milestone D — GitHub App reference provider (first)

**Status:** Landed under [`examples/providers/github`](examples/providers/github). Real mode mints an App JWT and exchanges it for a short-lived installation token; `PADE_PROVIDER_FAKE=1` keeps CI offline. Unit coverage uses `httptest`. Dogfood: `make dogfood-exec-provider-github`.

**Goal:** Add the **first** in-tree reference provider and primary derived-credential dogfood path.

Location: **`examples/providers/github/`** (clearly non-core). Avoid an `extensions/` directory name.

Explicit statement:

> In-tree reference providers exist for dogfooding and illustrating the provider contract. They are non-normative and architecturally removable from PADE core. Their presence does **not** make GitHub (or Google Analytics) part of the PADE standard.

Preferred flow:

```text
GitHub App private key
        ↓
trusted broker-side source
        ↓
PADE broker
        ↓
GitHub reference provider
        ↓
short-lived GitHub App installation token
        ↓
cloud DevelopmentSession
        ↓
GitHub repository operation
```

The provider is responsible for all GitHub-specific behavior, such as:

- GitHub App authentication
- installation identification
- token issuance
- repository restriction
- requested permission restriction
- token expiry handling

PADE core must **not** gain fields or logic specific to GitHub Apps (for example `githubInstallationId` as normative protocol semantics). Opaque provider configuration local to the broker/admin side is fine.

### Milestone E — GitHub dogfood migration

**Status:** Partial. Preferred offline path uses capability `github.repo.read` + [`examples/demo-project/scripts/github-repo-meta`](examples/demo-project/scripts/github-repo-meta) (GET `/repos/{owner}/{repo}`; fake tokens skip network). Wired into `make dogfood-exec-provider-github`. PAT + `github-whoami` remain the stage-1 baseline. Live App install + cloud agent proof still optional/local.

**Goal:** Migrate the **preferred** pre-release GitHub dogfood from PAT delivery to derived installation tokens, while keeping the PAT path documented as stage 1.

**Success condition:** Do **not** assume `/user` or `whoami` remains appropriate. GitHub App installation tokens represent the **app installation**, not the developer personally.

Minimal **repository-scoped** validation matching granted permissions:

- reading repository metadata (`github-repo-meta`) — preferred
- listing branches / fetching contents — optional later

Do **not** select overly broad permissions merely to make the demo easy.

The dogfood should demonstrate that the resulting token is:

- short-lived
- repository-scoped where practical
- permission-scoped
- derived without exposing the durable private key to the DevelopmentSession

### Milestone F — Google service-account reference provider (second structural test)

**Status:** Landed under [`examples/providers/google-analytics`](examples/providers/google-analytics). Real mode uses a broker-side service account JSON/key to mint a JWT assertion and exchange it for a short-lived OAuth2 access token (`analytics.readonly` by default). `PADE_PROVIDER_FAKE=1` keeps CI offline. Dogfood: `make dogfood-exec-provider-ga` (+ `ga-property-meta`). Live Google property proof remains optional/local.

**Goal:** Add the **second** in-tree reference provider to prove the generic contract was **not** accidentally designed around GitHub’s installation-token exchange. This is **not** vendor breadth or Google Analytics product support.

Location: **`examples/providers/google-analytics/`** (directory name is dogfood convenience only).

```text
Google service-account private key (broker-side)
        ↓
signed assertion
        ↓
OAuth token exchange
        ↓
short-lived access token (+ optional GA_PROPERTY_ID in Material)
        ↓
DevelopmentSession
        ↓
minimal downstream API call (validation only — not GA product logic in PADE)
```

All Google-specific authentication behavior belongs inside the provider. Do **not** add Google service-account, OAuth-scope, or Analytics-specific fields to PADE core. Dogfood chose service-account JWT bearer grant as the first strategy; alternatives remain provider-local.

### Milestone G — Two-provider architectural test

**Status:** Landed. `make dogfood-exec-provider-two` binds `github.repo.read` and `google-analytics.read` through the same `provider: exec` seam in one Intent/bindings pair (fake mode). No GitHub/Google fields were added to PADE core or Intent schema.

**Goal:** Explicit pre-release validation that GitHub and Google providers use the **same** generic PADE provider seam without requiring vendor-specific changes to PADE core.

The architectural question this milestone answers:

> Can two structurally different derivation flows (GitHub installation-token exchange vs Google OAuth JWT-bearer grant) bind through the same `provider: exec` contract without new normative fields in PADE core or Intent?

Treat this as an **architectural test**, not an integration catalog milestone.

Examples of undesirable leakage into normative PADE protocol semantics:

- `githubInstallationId`
- `googleServiceAccount`
- `analyticsPropertyId`
- `oauthScope`

Opaque provider configuration (admin/local policy) is acceptable; vendor fields as portable Intent or core wire semantics are not.

If implementing the second provider had required adding vendor-specific concepts to the provider protocol, the abstraction would need revisiting rather than extending core with provider-specific fields. Milestone G dogfood shows that did **not** occur for these two flows.

### Milestone H — Spec/docs tighten from dogfood

**Goal:** Update draft specs, SECURITY notes, and examples based on what D–G learned.

Do **not** redesign DevelopmentSession / Intent around Runtime Conditions or invent Grant/Lease without evidence. Keep changes proportional to dogfood.

### Milestone I — Initial versioned release (`v0.1.0`)

**Goal:** External consumers pin `PADE_VERSION=vX.Y.Z` instead of arbitrary `main` commits—**only after** A–H succeed.

**SemVer strategy:** Pre-1.0 SemVer. Breaking changes allowed with minor bumps while major remains `0`. Initial release: **`v0.1.0`**.

#### Release gate (required)

**Direct materialization path (stage 1)**

- Existing credential can be retrieved
- Broker can deliver it to a DevelopmentSession
- Simple current GitHub/PAT dogfood remains documented if useful

**Generic provider path**

- Broker can invoke a provider through a vendor-neutral contract
- Provider implementation remains outside PADE core
- Expiry metadata can be represented sufficiently for dogfood

**GitHub provider**

- GitHub App durable private key remains broker-side
- Provider derives a short-lived installation token
- Token is scoped appropriately
- Cloud DevelopmentSession successfully performs a repository-scoped operation

**Google service-account provider (second test)**

- Durable service-account private key remains broker-side
- Provider derives a short-lived OAuth access token via JWT-bearer exchange
- Token is scoped appropriately for dogfood
- DevelopmentSession successfully performs a minimal downstream API call using the derived token (validation only—not GA product logic in PADE)
- Offline/fake + httptest paths satisfy the seam proof; live Google property proof remains optional/local

**Abstraction validation**

- Both providers use the same generic provider seam (`provider: exec`)
- Neither requires vendor-specific PADE core behavior
- The second provider exists to prove the seam is not GitHub-App-shaped—not to expand PADE’s integration catalog
- See [Why two derived-token providers before v0.1.0](#why-two-derived-token-providers-before-v010) for release-gate and security rationale

**Not required for `v0.1.0`:** mediated capabilities (stage 3); full provider ecosystem; every future binding; Intent redesign; Runtime Conditions integration; Cloudflare/preview completion; live App/GA proofs (optional enhancements only).

#### Release artifacts (implement at I)

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
- Cut the first release only after the dogfood gate above succeeds; broker deployment should then consume released/versioned artifacts rather than unreleased repository state.

#### Artifact contract (PADE publishes; externals consume)

**`pade-broker-deployment` should eventually consume:**

```text
ghcr.io/ksteffe/pade-broker:vX.Y.Z
```

without cloning or building PADE source. Operators must be able to identify PADE version, source commit, and image digest.

**Cloud development environments should install** `pade vX.Y.Z` from a GitHub Release asset.

### Milestone J — Released broker deployment

External: private broker deployment pulls the released GHCR image, mounts policy/bindings/secrets, and runs with Cloud Run–compatible transport (`PORT`, trusted TLS termination).

PADE acceptance: image runs as already smoked by `make smoke-broker-container`; no PADE protocol change required.

### Milestone K — Released Consumer dogfood

External Cursor Cloud (or equivalent) installs released `pade`, points agent bindings at the real broker, and resolves a capability end-to-end with Cursor OIDC.

PADE acceptance: released CLI talks to released broker using the existing wire protocol.

### Milestone L — External preview / Cloudflare dogfood

after-certainty runs its normal application and establishes a Cloudflare Tunnel using **generic** PADE-brokered Material (tunnel token or equivalent), plus **repo-local** start command / port / `cloudflared` invocation.

Evaluate long-running `pade exec` (signals, exit codes, Material lifetime, redaction). Fix gaps as **generic execution** improvements—not Cloudflare support.

Preserved from the prior roadmap; intentionally **after** the initial release. Keeper/1Password and other historical dogfood remain documented under historical milestones and existing docs—not deleted.

### Milestone M — Endpoint decision

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

### Milestone N — Full Cursor iOS workflow

Acceptance story:

From Cursor iOS, launch a fresh after-certainty Cloud Agent and ask it to inspect Google Analytics, make an appropriate change, run the application, and provide a public preview—**without manually copying credentials into the agent**.

Expected path uses **released** CLI/broker (post-I) and derived Material where the configured provider supports it:

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

### Milestone O — Post-dogfood protocol evaluation

Only **generic** deficiencies discovered during real use return to the PADE repository (Intent, Consumer, Broker, or reference execution behavior). Vendor-specific work stays in external repos or `examples/providers/`.

## Capability naming (exploratory)

External dogfood may use names resembling `github.repo.read`, `analytics.google.read`, or `preview.http`.

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
3. **Capability-specific parameters** — How are they passed without leaking vendor semantics into core (opaque provider config vs normative protocol fields)?
4. **Durable authority handoff** — How does the broker pass durable authority to the provider safely?
5. **Generic provider result** — What does a portable result look like beyond today’s env `Material`?
6. **Expiry / refresh** — How are expiry and refresh represented for dogfood and beyond?
7. **Renewal during a live DevelopmentSession** — Can a provider request renewal while a session is still running?
8. **Revocation on session end** — How is revocation handled when a session ends?
9. **First provider binding** — Exec/subprocess vs HTTP vs gRPC vs plugin—choose during Milestone C, not in this planning pass.
10. **Provider failure surfacing** — How are provider failures surfaced to the Consumer?
11. **Provider isolation** — How are provider implementations isolated from the broker?
12. **Config vs Intent** — Which provider configuration is local/admin policy versus portable project Intent?
13. **Fulfillment pipeline naming** — Keep illustrative SOURCE / PROVIDER / DERIVATION / DELIVERY, or stick strictly to Provider / materialization / Material?
14. **Mediated capabilities** — Does credential-less exercise require Consumer protocol changes beyond today’s resolve → Material path?
15. **Runtime Conditions (CNCF)** — Active discussion with Runtime Conditions maintainers. Emerging possibility: Runtime Conditions describe requirements of a runtime *thing* (application, Dev Container, code session, agent sandbox, development environment), while PADE creates/fulfills an identity-bound DevelopmentSession and handles downstream capability fulfillment. **Do not** claim adopt/extend/replace/integrate yet. **Do not** redesign Intent while this is open.
16. **Hierarchical / composed profiles** — Emerging idea that development-session requirements could layer on top of application runtime requirements. Record only; no PADE schema work yet.
17. **Endpoint declaration** — See Milestone M.
18. **Capability vocabulary** — See [Capability naming](#capability-naming-exploratory).

## Deferred until after the initial release / full workflow

Explicitly deferred (not blockers for `v0.1.0` unless noted otherwise):

- **Mediated capabilities** (Consumer never receives credential Material) — after stage 2 is proven with both reference providers
- Endpoint schema implementation **unless** Milestone M decides it is required
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
- Cloudflare / preview completion (Milestone L — post-release, preserved)
- Keeper / 1Password / Vault historical dogfood remains available; not deleted

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

Earlier README “9+ / Later” rows, the release-first A–I ordering, and the GA-first A–M ordering (PR #31) are **superseded** by Milestones **A–O** in this document (GitHub App first, Google Analytics second, both before `v0.1.0`).
