# Prefer a repo-local Go toolchain when present (.tools/go from go.dev).
# System Go 1.13 and similar cannot build this module (no embed stdlib, old modules).
GO ?= $(shell if [ -x "$(CURDIR)/.tools/go/bin/go" ]; then echo "$(CURDIR)/.tools/go/bin/go"; else command -v go; fi)
DEVPOD_DOGFOOD := $(CURDIR)/scripts/devpod-dogfood.sh

IDENTITY_DOGFOOD := $(CURDIR)/scripts/identity-dogfood.sh
VAULT_DOGFOOD := $(CURDIR)/scripts/vault-dogfood.sh

.PHONY: check-go test build build-linux validate plan capabilities exec-demo dogfood \
	dogfood-identity dogfood-vault \
	dogfood-devpod dogfood-devpod-check dogfood-devpod-provider dogfood-devpod-up \
	dogfood-devpod-install dogfood-devpod-smoke dogfood-devpod-down dogfood-devpod-delete \
	dogfood-devpod-ci \
	vet fmt-check ci-unit ci-smoke ci

check-go:
	@v="$$( $(GO) env GOVERSION 2>/dev/null || true )"; \
	echo "Using: $(GO) ($$v)"; \
	case "$$v" in \
	  go1.2[2-9]*|go1.[3-9][0-9]*|go2*) ;; \
	  *) echo "error: need Go 1.22+; got $$v" >&2; \
	     echo "Install: https://go.dev/dl/  or unpack into .tools/go" >&2; \
	     echo "Then: export PATH=\"$$(pwd)/.tools/go/bin:\$$PATH\"" >&2; \
	     exit 1 ;; \
	esac

fmt-check: check-go
	@unformatted="$$(gofmt -l cmd internal spec)"; \
	if [ -n "$$unformatted" ]; then \
	  echo "The following files need gofmt:"; \
	  echo "$$unformatted"; \
	  exit 1; \
	fi

vet: check-go
	$(GO) vet ./...

test: check-go
	$(GO) test ./...

build: check-go
	$(GO) build -o bin/pade ./cmd/pade

# Cross-compile for the local Docker VM (linux/amd64 or linux/arm64).
build-linux:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) build

validate: check-go
	$(GO) run ./cmd/pade validate -f spec/examples/web-app.yaml

plan: check-go
	$(GO) run ./cmd/pade plan -f spec/examples/web-app.yaml

capabilities: check-go
	$(GO) run ./cmd/pade capabilities -f spec/examples/web-app.yaml --bindings spec/examples/bindings.example.yaml

# Demonstrates process-scoped injection without printing secret values.
exec-demo: check-go build
	GA_PROPERTY_ID=demo-property \
	GOOGLE_APPLICATION_CREDENTIALS=/tmp/fake-ga.json \
	./bin/pade exec \
	  -f spec/examples/web-app.yaml \
	  --bindings spec/examples/bindings.example.yaml \
	  --capability google-analytics.read \
	  -- /bin/sh -c 'test -n "$$GA_PROPERTY_ID" && test -n "$$GOOGLE_APPLICATION_CREDENTIALS" && echo exec-ok'

# Milestone 4: PADE smoke against examples/demo-project (DevPod not required).
dogfood: check-go build
	chmod +x examples/demo-project/scripts/ga-summary
	./bin/pade validate -f examples/demo-project/pade.yaml
	./bin/pade plan -f examples/demo-project/pade.yaml --bindings examples/demo-project/bindings.example.yaml
	GA_PROPERTY_ID=demo-property \
	GOOGLE_APPLICATION_CREDENTIALS=/tmp/fake-ga.json \
	./bin/pade exec \
	  -f examples/demo-project/pade.yaml \
	  --bindings examples/demo-project/bindings.example.yaml \
	  --capability google-analytics.read \
	  -- ./scripts/ga-summary

# Milestone 5: same pade.yaml, two simulated identities, distinct resolved material.
dogfood-identity: check-go build
	@chmod +x "$(IDENTITY_DOGFOOD)"
	@PADE="$(CURDIR)/bin/pade" "$(IDENTITY_DOGFOOD)"

# Vault -dev dogfood: resolve capabilities from Vault (prototype only; downloads Vault if needed).
dogfood-vault: check-go build
	@chmod +x "$(VAULT_DOGFOOD)"
	@PADE="$(CURDIR)/bin/pade" "$(VAULT_DOGFOOD)"

# --- DevPod dogfood (requires docker + devpod; separate DevPod GHA workflow) ---
dogfood-devpod-check:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) check

dogfood-devpod-provider:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) provider

dogfood-devpod-up:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) up

dogfood-devpod-install:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) install

dogfood-devpod-smoke:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) smoke

dogfood-devpod-down:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) down

dogfood-devpod-delete:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) delete

# Full local proof: provider + up + linux pade install + in-workspace smoke.
dogfood-devpod:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) all

# CI entrypoint used by .github/workflows/devpod-dogfood.yml
dogfood-devpod-ci:
	@chmod +x "$(DEVPOD_DOGFOOD)"
	@$(DEVPOD_DOGFOOD) ci

# Fast path: mirrors the GitHub Actions "Unit tests" job.
ci-unit: check-go fmt-check vet test build

# Smoke path: mirrors the GitHub Actions "Smoke" job (env + Vault dogfood).
ci-smoke: check-go build dogfood dogfood-identity dogfood-vault
	./bin/pade validate -f spec/examples/web-app.yaml
	./bin/pade plan -f spec/examples/web-app.yaml --json > /tmp/pade-plan.json
	./bin/pade capabilities -f spec/examples/web-app.yaml --bindings spec/examples/bindings.example.yaml --json > /tmp/pade-capabilities.json
	GA_PROPERTY_ID=demo-property \
	GOOGLE_APPLICATION_CREDENTIALS=/tmp/fake-ga.json \
	./bin/pade exec \
	  -f spec/examples/web-app.yaml \
	  --bindings spec/examples/bindings.example.yaml \
	  --capability google-analytics.read \
	  -- /bin/sh -c 'test -n "$$GA_PROPERTY_ID" && test -n "$$GOOGLE_APPLICATION_CREDENTIALS" && printf exec-ok'
	mkdir -p /tmp/pade-orch/.devcontainer
	printf '{}\n' > /tmp/pade-orch/.devcontainer/devcontainer.json
	cp spec/examples/web-app-orchestrated.yaml /tmp/pade-orch/pade.yaml
	./bin/pade validate -f /tmp/pade-orch/pade.yaml
	./bin/pade plan -f /tmp/pade-orch/pade.yaml --json > /tmp/pade-orch-plan.json

# Full local mirror of .github/workflows/ci.yml (unit then smoke).
ci: ci-unit ci-smoke
