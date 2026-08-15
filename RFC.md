# RFC: Portable Agent Development Environments (PADE)

> Converted from the project Google Doc for repository use.
> Preserve revision history in-document; README states the current v0.1 direction.
>
> **Reading guide:** Early sections describe the broad portable workspace vision
> (including historical “workspace broker” language). Sections 24+ revise toward
> DevPod-first capability interoperability. Section 36 records the latest model:
> PADE as an Interoperability Contract (Intent Spec + Consumer Spec + Broker Spec).
> Section 37 records the Kubernetes-style DevelopmentSession Intent convention.
> For current draft specifications see [spec/README.md](spec/README.md). The README
> is the short source of truth for current direction.
>
> **Pre-release validation (current):** Before `v0.1.0`, PADE dogfoods generic
> broker-side credential derivation through two **non-normative** in-tree reference
> providers on the same seam (GitHub App; Google service-account OAuth as a second
> structural test). See [ROADMAP.md](ROADMAP.md#why-two-derived-token-providers-before-v010).
> Early RFC mentions of Google Analytics as example capabilities are **historical
> context**, not a claim that PADE ships GA product support.

**Status: Exploratory**  
**Date: August 2026**  

## Abstract

AI development tools increasingly execute work outside the developer’s local workstation. Current products often couple the agent, execution environment, credentials, networking, and user interface to a particular vendor. This RFC proposes a portable abstraction for an agent development workspace that can execute locally or across multiple cloud-agent providers while preserving environment definition, organizational policy, runtime identity, tool access, and interactive development capabilities.

The proposal does not define a new authentication or authorization system. Existing identity, IAM, secrets-management, workload-identity, and resource authorization systems remain authoritative. The goal is to make development environments and their capability requirements portable across agent runtimes.

## 1. Motivation and Problem Statement

A developer may use Cursor locally, Cursor Cloud, Claude Code, GitHub Codespaces, Coder, CI runners, or other agent environments. Each execution location independently solves some combination of environment provisioning, repository checkout, developer tools, agent instructions and skills, secrets, identity, network access, port forwarding, persistent context, compute, and lifecycle management.

This unnecessarily couples four concerns that should be separable:

- Human interface — phone, laptop, browser, IDE, voice, or future interfaces.
- Agent — Cursor, Claude Code, Codex, Gemini, or another model/agent system.
- Execution environment — local machine, ephemeral cloud workspace, CI runner, or enterprise infrastructure.
- Authority — the identity and permissions granted to a particular running workload.

Changing an agent or execution provider should not require reconstructing the developer’s entire environment and its trust relationships.

## 2. Design Goals

A portable agent workspace should allow a developer to define an environment once and run it through multiple compatible providers. The developer should be able to choose the most appropriate interaction surface and execution location for each task.

The design should:

- support both local and cloud execution;
- avoid coupling the workspace to a particular AI vendor or model;
- reuse existing development-environment standards where possible;
- separate environment definition from runtime authority;
- support short-lived and dynamically issued credentials;
- preserve downstream authorization and domain invariants;
- provide authenticated interactive ingress for development servers;
- support reproducible and rapidly provisioned environments;
- allow organizations to define security and cost guardrails without centralizing every task-level decision;
- support auditing and lifecycle management of ephemeral workloads.

## 3. Non-Goals

PADE is not intended to replace OAuth, OIDC, IAM, RBAC, ABAC, ReBAC, SPIFFE, secrets managers, database authorization, API authorization, or domain-specific security logic.

It is not a new cryptographic credential format. Zero-knowledge proofs may be useful for particular identity or selective-disclosure scenarios, but they are not required by this architecture.

PADE is also not an MCP replacement. MCP may provide an excellent semantic interface between agents and services. PADE concerns the environment in which agents execute and the authority available to those environments.

## 4. Conceptual Architecture

The desired architecture separates interaction from execution:

```text
Developer
 ↓
Phone / Laptop / Browser / IDE / Future Interface
 ↓
Agent Client (Cursor / Claude / Codex / other)
 ↓
Workspace Broker
 ↓
Local Runtime OR Cloud Runtime A OR Cloud Runtime B
 ↓
Ephemeral Workspace Identity
 ↓
Credential / Capability Broker
 ↓
GitHub / Google Analytics / Datadog / Cloud / Internal APIs / Databases
```

The human interface does not determine where execution occurs. The agent provider does not need to own the development environment. The workspace definition does not contain authority. Authority is negotiated at runtime.

## 5. Workspace Definition

A repository SHOULD be capable of declaring execution requirements independently of the AI vendor. The Development Container specification (devcontainer.json) is a strong existing foundation because it already describes reproducible development environments and is supported by multiple tools, including GitHub Codespaces and Coder.

A PADE manifest could add agent-specific concerns around that existing standard rather than replacing it. For example:

```yaml
workspace:
 environment:
 devcontainer: .devcontainer/devcontainer.json


 capabilities:
 - github
 - google-analytics.read
 - search-console.read


 services:
 web:
 command: npm run dev
 port: 3000
 ingress: authenticated


 agent:
 skills: .agents/skills
 instructions: AGENTS.md


 resources:
 cpu: 4
 memory: 8GB


 lifecycle:
 idle_timeout: 30m
 maximum_lifetime: 8h
```

This manifest declares required capabilities, not credentials. Credentials MUST NOT be embedded in portable workspace definitions or images.

## 6. Environment, Behavior, and Authority

The architecture explicitly separates three concepts.

Environment answers: What software and tools are available?

Behavior answers: How has the agent been instructed to behave?

Authority answers: What can this particular running instance actually do?

An image may contain psql, kubectl, a Google Analytics query utility, Git tooling, and agent skills without possessing authority to use any external resource.

Agent behavior belongs close to the agent. Security enforcement belongs outside the agent.

Agent instructions may encourage good behavior such as checking token TTL, requesting minimal scopes, avoiding destructive operations, or explaining permission requirements. These instructions improve ergonomics and reduce invalid requests, but they MUST NOT be treated as the authoritative security boundary.

## 7. Runtime Identity

Every ephemeral workspace SHOULD receive a unique workload identity. Existing workload identity standards should be preferred over proprietary identity schemes.

SPIFFE is one candidate because it defines portable workload identities across heterogeneous infrastructure. A workspace might receive an identity conceptually similar to:

spiffe://company.example/developer/user/workspace/7f821

This identifies the running workload rather than merely the human developer.

The runtime identity can then participate in credential exchange, policy decisions, auditing, and revocation.

## 8. Human-to-Workload Delegation

The developer authenticates using the organization’s existing identity provider. The workspace broker determines what authority may be delegated to the requested workspace.

The resulting authority should be no greater than the intersection of:

```text
Human authority
∩ Organizational policy
∩ Workspace-requested capabilities
```

For example, if a developer can read Google Analytics and write to a GitHub repository, but the workspace only requests Google Analytics read access, the workspace should receive only that capability.

If organizational policy prohibits production writes from cloud agents, a developer’s own production-write entitlement does not override that policy.

This is authority attenuation, not authority creation.

## 9. Credential and Capability Brokering

Where supported, the runtime SHOULD exchange its workload identity for short-lived resource credentials instead of injecting persistent secrets.

Existing systems already demonstrate important pieces of this model. GitHub Actions OIDC can exchange job identity for short-lived cloud credentials. HashiCorp Vault can issue dynamic database credentials with leases and revocation. SPIFFE and products such as Teleport provide workload identity primitives.

The standardized opportunity is therefore not a new secret-storage system. It is a common way for ephemeral development workloads to request capabilities from whatever identity and secrets infrastructure an organization already uses.

Conceptually:

```text
Workspace
 ↓
```

Credential Broker API
```text
 ↓
```

Vault / 1Password / AWS IAM / GCP IAM / Azure Entra / GitHub / OAuth / Enterprise IdP
```text
 ↓
```

Short-lived scoped credential
```text
 ↓
```

Target resource

Static secrets may be supported as a compatibility fallback when a provider cannot issue dynamic credentials, but SHOULD NOT be the preferred model.

## 10. Interactive Ingress

Interactive ingress SHOULD be a first-class workspace capability.

A workspace may declare that a development server listens on port 3000 and should be visible only to the authenticated developer. The runtime can expose a temporary HTTPS endpoint associated with the workspace.

For example:

Cloud workspace :3000
```text
 ↓
```

Authenticated temporary ingress
```text
 ↓
https://7f821.workspace.example
 ↓
```

Safari / desktop browser / API client

The URL SHOULD expire with the workspace and SHOULD be private by default.

GitHub Codespaces already demonstrates an important version of this model with authenticated port forwarding. Cursor cloud agents currently demonstrate related desktop port-forwarding capabilities. A portable standard would make this behavior available independently of a particular agent client.

This is especially valuable for mobile development workflows. A developer could request a UI change from a phone, allow a cloud agent to implement it, open the running application interactively in a mobile browser, provide feedback, and continue iterating without triggering a deployment build.

## 11. Execution Provider Portability

A workspace SHOULD be executable by multiple providers. Conceptually:

pade run --local
pade run --provider=cursor
pade run --provider=coder
pade run --provider=codespaces
pade run --provider=enterprise-cloud

The commands are illustrative. The important principle is that the repository describes the environment and capability requirements while the execution provider determines how to instantiate them.

This resembles the relationship between OCI images and container runtimes: a common description enables multiple implementations.

## 12. Agent Portability

The workspace SHOULD NOT require a particular model or agent implementation.

A compatible environment might host Cursor, Claude Code, Codex, Gemini, an enterprise model, or future agent systems. Agent-specific extensions are acceptable, but the basic environment and capability requirements should remain portable.

This gives organizations an important property: changing models or agent vendors does not require rebuilding developer infrastructure.

## 13. Security Model and Defense in Depth

The workspace layer is not the authoritative authorization layer for downstream resources.

If an agent invokes psql, Postgres remains responsible for database roles, grants, row-level security, views, and other database authorization.

If an agent invokes an MCP tool that calls a company API, the company API remains responsible for domain authorization and invariants.

If an MCP server prevents an invalid or unauthorized request before it reaches a downstream service, that is useful defense in depth and improved agent UX. The downstream system MUST still enforce the real boundary.

MCP can therefore be understood in many architectures as analogous to a backend-for-frontend for AI: it provides semantic information, curated capabilities, request shaping, aggregation, and useful errors. It may participate in authentication and authorization, but passing through MCP does not remove the responsibility of downstream systems to enforce their own security rules.

A useful principle is:

Help the agent form good requests. Never depend on it forming good requests.

## 14. Local and Cloud Execution

Local execution remains valuable. A developer’s workstation often already contains repositories, tools, caches, credentials, network access, and context. Local compute can also be inexpensive when hardware has already been provisioned.

Cloud execution offers different advantages: isolation, elastic compute, parallelism, reproducibility, rapid teardown, centralized policy, and easier control over the blast radius of an individual workload.

PADE therefore does not assume cloud agents will replace developer laptops. It assumes execution placement will increasingly become a task-level decision.

A scheduler or user may choose an execution environment based on variables including compute cost, CI/build cost, latency, data locality, security sensitivity, credential availability, network access, context-hydration cost, required hardware, parallelism, persistence, and interaction needs.

## 15. Cost-Aware Execution

Agent workflows introduce a distinction between development compute and deployment infrastructure.

For example, a developer using Vercel may currently trigger preview deployments simply to interactively test changes produced by a cloud agent. This consumes deployment build CPU for exploratory development.

With interactive cloud-workspace ingress, the workflow can instead become:

Define task → ephemeral development workspace → edit/lint/test → interactive live preview → iterate → deployment preview only when needed → production.

The execution broker could eventually support policies such as “do not invoke paid CI for exploratory branches” or “run cheap validation before requesting a deployment build.”

The goal is not merely cloud execution, but choosing the cheapest, safest, and most useful execution location for each stage of a task.

## 16. Existing Products and Standards

This RFC intentionally composes existing ideas rather than claiming the underlying technologies are novel.

Development Containers (containers.dev) provide a portable development-environment specification and should be considered a foundation rather than reinvented.

GitHub Codespaces demonstrates repository-integrated cloud development environments, Dev Container support, development secrets, and authenticated port forwarding.

Coder provides centrally managed development workspaces, templates, Dev Container integration, enterprise identity integration, and increasing support for AI-agent execution.

Cursor Cloud Agents demonstrate isolated cloud agent environments, environment configuration, snapshots, secrets, GitHub integration, and remote development workflows.

SPIFFE provides standardized workload identities intended to work across heterogeneous infrastructure.

HashiCorp Vault demonstrates dynamic secrets, leases, and revocation.

Teleport demonstrates short-lived machine/workload identity and SPIFFE interoperability.

GitHub Actions OIDC demonstrates how an ephemeral workload can exchange an authenticated workload identity for temporary credentials instead of relying on stored long-lived cloud keys.

The opportunity is not to rebuild these systems. It is to define an interoperable layer connecting portable development workspaces, agent clients, execution providers, and existing identity infrastructure.

## 17. Relationship to MCP

MCP is complementary to this proposal.

An MCP server may expose a curated semantic interface to a company API. It can help an agent understand available operations and form correct requests. This is similar to a backend-for-frontend tailored to an AI client.

PADE answers a different question: where does the agent execute, what environment does it receive, and how does that ephemeral workload obtain legitimate authority?

An agent in a PADE workspace may use MCP, CLI tools, direct HTTP APIs, database clients, or any combination of these.

Security MUST NOT depend solely on the MCP layer. Downstream services remain responsible for authoritative authorization and domain invariants.

## 18. Example: Personal Website Development

A useful initial validation case is a developer maintaining a personal website from both a phone and laptop.

Desired workflow:

1. The repository defines its environment with a Dev Container.
2. The workspace includes Node, Git, testing tools, and approved analytics tooling.
3. The developer starts a cloud workspace from Cursor, Claude Code, or another compatible agent client.
4. Runtime credentials provide read-only Google Analytics and Search Console access without storing persistent credentials in the workspace image.
5. The agent can answer questions about site analytics from the phone.
## 6. The agent starts the development server.
7. The developer receives an authenticated temporary URL and interactively tests the site from Safari.
8. The developer provides additional instructions and iterates against the same running workspace.
9. Vercel preview/deployment infrastructure is invoked only when deployment-specific validation is required.
10. When work ends, the workspace is destroyed and its temporary credentials are revoked or expire.

This scenario exercises portability, remote execution, interactive ingress, credential brokering, mobile interaction, cost management, and lifecycle controls without requiring a hypothetical enterprise use case.

## 19. Prototype Acceptance Criteria

An initial prototype should avoid inventing unnecessary infrastructure. It should reuse Dev Containers, an existing cloud workspace provider, existing identity/secret mechanisms, and existing agent tools.

The prototype succeeds if:

- one repository can be provisioned reproducibly in an ephemeral environment;
- at least two different AI development tools can operate against substantially the same workspace definition;
- long-lived credentials are not baked into the workspace image;
- the workspace can obtain approved access to at least one external API such as Google Analytics;
- a development server can be accessed through authenticated temporary ingress;
- the same workspace can be controlled from more than one user interface or device;
- the workspace can be destroyed without leaving reusable runtime credentials behind;
- downstream resources continue enforcing their existing authorization policies.

## 20. Open Questions

Several questions require experimentation rather than premature standardization.

How much agent state should be portable across vendors? Conversation history, plans, skills, task state, and model-specific context may have different portability characteristics.

Should capability manifests be standardized, or should PADE initially provide adapters to existing provider-specific declarations?

What is the appropriate protocol between a workspace and a credential broker?

How should user identity be propagated when a cloud agent acts on behalf of a human while still maintaining a distinct workload identity?

How should policy distinguish interactive human-approved actions from autonomous actions?

How should organizations reason about cost limits for elastic agent compute and parallel execution?

How should workspace snapshots avoid accidentally persisting secrets or sensitive runtime state?

What should the trust model be when the agent provider and workspace provider are different organizations?

How much context should be hydrated into a cloud workspace versus fetched on demand?

## 21. Product Hypothesis

The product opportunity, if one exists, is not “secure MCP” or a new credential primitive.

It is a portable execution and capability-brokering layer for agentic development.

The core user promise is:

Define my development environment once. Let me use it from whichever AI development tool and device I choose. Run it locally or in an ephemeral cloud environment. Give the running workload only the authority it legitimately needs. Let me interact with what it builds. Tear everything down cleanly when the task is finished.

The enterprise promise is similarly pragmatic: organizations retain their existing IAM, authorization, secrets, APIs, databases, and security policies while gaining a consistent way to provision and govern ephemeral AI development workloads.

## 22. Guiding Principles

Prefer existing standards over new primitives.

Separate environment, behavior, and authority.

Treat agent instructions as ergonomics and defense in depth, not as security boundaries.

Keep authoritative authorization close to the resources and domain logic it protects.

Make the secure path easier than the insecure path.

Allow local and cloud execution to coexist.

Treat execution location as a task-placement decision rather than a developer identity.

Issue authority at runtime rather than baking it into images.

Make ephemeral environments genuinely ephemeral.

Help the agent form good requests. Never depend on it forming good requests.

## 23. Next Step

Before proposing a new standard or company, build the smallest useful prototype around an actual development workflow.

A strong experiment would use one real repository, Dev Containers, an existing cloud workspace provider, authenticated live ingress, a read-only external API capability such as Google Analytics, and two different agent clients.

The central question is not whether the architecture is patentable. It is whether the resulting workflow is meaningfully better than the tools developers already have.

If it is, the prototype will reveal which interoperability gaps are genuinely painful. Those gaps—not a predetermined technology—should determine what needs to be standardized or built next.

## 24. DevPod-First Revision

Further research shows that DevPod already provides much of the portable workspace/runtime layer originally envisioned for PADE. DevPod consumes Dev Container definitions, supports local and remote execution through pluggable providers, manages workspace lifecycle, exposes SSH connectivity and port forwarding, and supports prebuilds stored in OCI registries.

PADE therefore SHOULD NOT implement its own general workspace runtime, provider lifecycle, port forwarding, or prebuild registry unless a concrete gap requires it.

The revised architecture is:

```text
Repository
 ↓
Dev Container specification
 ↓
DevPod
 ↓
Local / SSH / Kubernetes / cloud provider
 ↓
Development workspace
```

Alongside that runtime path, PADE focuses on a narrower portable capability layer:

```text
Repository
 ↓
Capability declaration
 ↓
PADE capability resolver / binding
 ↓
Vault / 1Password / OAuth / cloud IAM / enterprise broker
 ↓
```

Developer-specific runtime authority

The key remaining question is not “How do we create a portable cloud workspace?” DevPod already answers much of that. The remaining hypothesis is:

Can a repository declaratively express the external capabilities its development workload requires in a way that is portable across DevPod, Cursor, Coder, Codespaces, Claude Code, and other agent environments, while allowing each runtime or organization to bind those capabilities to its existing identity and credential infrastructure?

## 25. Revised Scope

PADE SHOULD focus on the seams not already owned by mature tools.

Dev Containers own the portable development environment definition.

DevPod owns portable runtime placement, provider abstraction, workspace lifecycle, remote connectivity, port forwarding, and prebuild orchestration.

Vault, 1Password, OAuth, workload identity, and cloud IAM own credential storage, issuance, exchange, and revocation.

Cursor, Claude Code, Codex, and other agent systems own agent behavior and model execution.

Downstream APIs, databases, and services continue to own authoritative authorization and domain invariants.

PADE's candidate responsibility is limited to:

- a portable vocabulary for declaring required external capabilities;
- a provider-neutral binding model that maps those capabilities to existing credential systems;
- optional process-scoped capability injection or execution wrappers;
- inspection commands that show which capabilities a workspace requests and how they will be resolved;
- interoperability conventions that agent/runtime vendors could eventually implement directly.

## 26. Revised Manifest Direction

Rather than duplicate Dev Container environment configuration, the portable declaration SHOULD complement it.

One possible shape is:

```yaml
capabilities:
 github.repo:
 access: read


 google-analytics:
 access: read


 datadog.logs:
 access: read
```

The capability declaration MUST NOT contain secret values or require a particular credential provider.

Provider-specific bindings belong outside the portable repository definition. For example, one developer or organization may bind google-analytics.read to Vault, another to 1Password, and another to a native OAuth flow.

This separation allows the same repository and Dev Container to be used under different identities and organizations without modifying the portable capability declaration.

## 27. DevPod Integration Strategy

The first prototype SHOULD treat DevPod as the execution substrate rather than implementing PADE runtime-provider interfaces.

A minimal flow becomes:

pade capabilities
```text
 ↓
```

validate requested capabilities
```text
 ↓
```

resolve bindings for current developer
```text
 ↓
devpod up
 ↓
```

workspace created from devcontainer.json
```text
 ↓
```

run selected process with resolved capability

PADE MAY invoke DevPod as an external CLI during the prototype. Native integration is not required.

The prototype should also investigate whether PADE can be expressed as a Dev Container customization or a DevPod extension rather than requiring a permanent standalone CLI.

For example:

```json
{
 "customizations": {
 "pade": {
 "capabilities": {
 "google-analytics": {
 "access": "read"
 }
 }
 }
 }
}
```

This is exploratory. A separate manifest should only be retained if it is materially clearer or more portable than using existing extension points.

## 28. Revised Prototype

The revised prototype should prove the narrowest remaining hypothesis.

Phase A — Environment portability

Use DevPod and a Dev Container directly. Do not write PADE runtime code. Demonstrate the same workspace locally and on one remote provider.

Phase B — Capability declaration

Define one portable read-only capability such as google-analytics.read.

Phase C — Local binding

Resolve that capability using local Vault in dev mode. The repository contains only the capability declaration; developer-specific secret material lives in Vault.

Phase D — Identity separation

Run the same repository under two different Vault identities or paths and demonstrate that each workspace receives different credential material without changing the repository definition.

Phase E — Scoped execution

Prefer resolving a capability only for the process that needs it rather than globally injecting it into the entire workspace.

Phase F — Second credential provider

Add one alternative binding mechanism, such as environment variables, 1Password, or OAuth. The same capability declaration should remain unchanged.

The success criterion becomes:

Same repository + same devcontainer.json + same capability declaration; different runtime and credential providers.

## 29. Revised Product Hypothesis

PADE is no longer hypothesized as a portable workspace platform.

Its possible value is a small interoperability specification for developer/agent capability requirements that sits beside Dev Containers and DevPod.

The strongest outcome may be that PADE disappears as a standalone product and its useful concepts become an extension to Dev Containers, DevPod, or another existing standard.

That outcome should be considered success rather than failure.

The project should continuously ask:

Does this capability belong in PADE, or can an existing tool express it cleanly?

If an existing tool owns the concern, PADE should compose with it or remove the abstraction.

## 30. Ideal Developer Experience

The ideal developer experience starts when a project declares the external capabilities it needs. A developer should not manually copy secrets into local files, cloud-agent configuration, or deployment settings simply because an agent needs to use a tool.

A project might declare:

```yaml
capabilities:
 google-analytics.read:
 access: read
 datadog.logs.read:
 access: read
```

When the developer obtains an API key or grants access in the source service, the approved credential manager should offer to save the credential directly under the corresponding capability name. The credential manager might be 1Password, Keeper, Vault, a cloud secrets manager, or an internally hosted enterprise service.

The important separation is:

```
Project declares what is needed.
Developer or organization binds how it is satisfied.
Runtime resolves the capability only when needed.
```

Resource remains responsible for final authorization.

This means the same repository can be used from a laptop, phone-driven cloud agent, remote DevPod workspace, or future interface without copying secrets between machines.

## 31. Capabilities and Skills

Capabilities and skills should remain separate.

A capability says: this work requires authority to use a resource.

A skill says: here is how to use that authority with existing tooling.

The skill may use a CLI, SDK, HTTP API, MCP server, database client, browser automation, or other mechanism. PADE should not care which implementation path is chosen. MCP remains a useful semantic interface, but the capability model is independent of MCP.

## 32. Authenticated Review Environments

The long-term developer workflow should let a cloud workspace expose an authenticated ephemeral review environment. A developer can run the executable in the cloud, open it from any authenticated device, and share access with teammates inside the organization.

The review flow becomes:

```
Developer starts workspace → executable runs in cloud → authenticated ingress URL is created → reviewer receives notification → reviewer opens the live artifact → humans review behavior together.
```

This changes review from only inspecting representations of the system to also inspecting the running system itself. Code review remains important, but executable review moves attention closer to the thing users actually experience.

## 33. Expanded Product Hypothesis

The possible product is not merely portable workspace setup or secret injection. Existing tools already handle much of that. The more compelling hypothesis is a developer-centered collaboration loop:

## 1. Declare capabilities in the project.
## 2. Bind credentials through approved credential managers.
3. Run the workspace through DevPod or another existing runtime.
## 4. Resolve capabilities at execution time.
5. Expose the running artifact through authenticated ephemeral ingress.
## 6. Share that artifact with collaborators.
7. Review the executable together before relying only on code or deployment previews.

Running code is the first target because the pain is concrete. The same pattern may later apply to rendered design documents, notebooks, architecture artifacts, or other reviewable outputs.

The guiding phrase is:

Humans should be able to review the executable together, not only the code that produced it.

## 34. Milestone 9 learning: cloud agents and secret-manager composition

A phone-driven Cursor Cloud Agent can use the same portable capability declaration as a laptop if:

1. The cloud environment supplies a narrowly scoped secret-manager bootstrap (for Milestone 9, Keeper Secrets Manager via ambient `KSM_CONFIG`).
2. Environment-specific bindings map capability names to secret-manager handles (Keeper Notation), not to secret values.
3. PADE resolves at `pade exec` time into the child process only.

This does not require Cursor-specific fields in `pade.yaml`. Cursor Runtime Secrets, environment install, and network allowlists remain Cursor’s concerns. PADE must not claim to sandbox a VM that already possesses the bootstrap credential.

A later evolution can replace the long-lived KSM bootstrap in the VM with Cursor workload OIDC verified by an external capability broker that holds Keeper credentials. Portable capability intent stays the same; only the binding/provider changes.

## 35. Phase 2 spike note

A prototype `pade-broker` now demonstrates that server-side policy (subject + complete `repo_urls` + capability allowlist) can gate resolution while Keeper bootstrap remains off the agent VM. This is a learning spike: short-lived token + audience binding without JTI replay storage; localhost HTTP for tests; broker-managed TLS or explicit `-tls-termination=proxy` for non-loopback. See `docs/cursor-oidc-broker-dogfood.md` and `SECURITY.md`.

## 36. PADE as an Interoperability Contract

This section records the latest architectural conclusion. Earlier RFC sections remain historical context; draft normative specs live under [`spec/README.md`](spec/README.md). The RFC itself is not the normative specification.

| Stage | Question / learning |
|-------|---------------------|
| Original question | How do we make agent development environments portable? |
| Learning | Existing runtimes (Dev Containers + DevPod and peers) already own much of workspace lifecycle. |
| Next question | What is actually portable? |
| Learning | The valuable portable artifact is **capability intent** (`DevelopmentSession` in `pade.yaml`), not a new workspace runtime. |
| OIDC / broker learning | The development runtime and the **authority boundary** can be independent. |
| Current model | **Intent Specification** + **Consumer Specification** + **Broker Specification** |
| Reference implementation | `pade` (Consumer) + `pade-broker` (experimental Broker) |

```text
                         PADE
              interoperability contract
       ┌────────────┬──────────────┐
       │            │              │
       ▼            ▼              ▼
  Intent Spec   Consumer Spec   Broker Spec
       │            │              │
       └────────────┴──────────────┘
                    │
                    ▼
             Go reference implementation
```

The Go implementation is used to **discover and validate protocol boundaries**, not to define the only valid PADE. Other runtimes, agent vendors, brokers, and service providers could implement the contract without using this repository’s code. **Provider adapters** (env, Vault, 1Password, Keeper, Keeper Secrets Manager) and the **Cursor OIDC workload identity adapter** remain implementation-specific unless later promoted into a specification with evidence.

Historical “workspace broker” / orchestration language in earlier sections remains useful context; prefer sections 24+ and this section when they conflict with early product-platform framing. The specifications are exploratory drafts (`pade.local/v1alpha1` Intent), not standards-body deliverables.

## 37. Kubernetes-Style Declarative Resource Convention

PADE adopts the familiar Kubernetes **API-object grammar** for portable Intent documents—not Kubernetes itself.

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

| Field | Why it is useful |
|-------|------------------|
| `apiVersion` | Explicit evolution path for the experimental wire format (`pade.local/v1alpha1`; `.local` avoids claiming a public DNS domain). |
| `kind` | Leaves room for future resource types without requiring them now. |
| `metadata` | Identifies the portable object (`name`, optional labels/annotations) without adopting API-server ObjectMeta. |
| `spec` | Portable desired intent (capability requests). |

Runtime/provider-specific observed state may eventually be represented separately as `status`, but that is **not** part of v1alpha1 Intent input. Reference Consumers continue to show satisfaction via `plan` / `capabilities`.

A Kubernetes platform **could** choose to expose PADE resources as CRDs/controllers. That would be one conforming implementation environment—not a PADE dependency or requirement. PADE MUST remain usable on a laptop, with DevPod, Coder, cloud agents, or other hosts.

This convention supports the design principle:

> PADE SHOULD STANDARDIZE ONLY WHAT MUST BE SHARED BETWEEN PROVIDERS.

Standardize portable intent. Leave runtimes, credential managers, brokers, and ingress systems as pluggable conforming implementations. See [docs/manifest-conventions.md](docs/manifest-conventions.md) and [spec/intent.md](spec/intent.md).

## 38. Implementation experiments may narrow after proving a seam

PADE uses implementation experiments to discover interoperability seams. Mechanisms that prove unnecessary or create disproportionate security burden may intentionally be narrowed or removed after they have served that purpose. Having functionality replaced by stronger standards or mature implementations is considered success.

Concrete example — external provider subprocess (`provider: exec`):

| Stage | Outcome |
|-------|---------|
| Experiment | Subprocess adapter to prove independently packaged providers |
| Lesson | A generic semantic provider contract works (GitHub App and Google SA OAuth on one seam) |
| Security finding | Development-side arbitrary execution is an unnecessary trust boundary next to repository state |
| Result | Exec retained only as a **broker-side** reference adapter; Consumer/development bindings cannot select it |

Portable `DevelopmentSession` declarations never needed to name executables. The semantic claim (“authorized capability → Material”) remains; the process adapter is optional scaffolding.

### Non-normative note: adjacent CNCF discussion

This direction was influenced methodologically by adjacent CNCF discussion of project-agnostic declarative specifications for application integration dependencies ([cncf/toc#1797](https://github.com/cncf/toc/issues/1797)): declare requirements independently of implementations, prefer existing specifications when possible, keep formats programmatically consumable, and let conforming systems choose how requirements are satisfied.

PADE is **not** affiliated with CNCF, endorsed by CNCF, compatible with that unfinished specification, or part of that initiative.
