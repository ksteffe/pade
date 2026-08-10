# Prefer a repo-local Go toolchain when present (.tools/go from go.dev).
# System Go 1.13 and similar cannot build this module (no embed stdlib, old modules).
GO ?= $(shell if [ -x "$(CURDIR)/.tools/go/bin/go" ]; then echo "$(CURDIR)/.tools/go/bin/go"; else command -v go; fi)

.PHONY: check-go test build validate plan capabilities vet fmt-check ci

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

validate: check-go
	$(GO) run ./cmd/pade validate -f spec/examples/web-app.yaml

plan: check-go
	$(GO) run ./cmd/pade plan -f spec/examples/web-app.yaml

capabilities: check-go
	$(GO) run ./cmd/pade capabilities -f spec/examples/web-app.yaml --bindings spec/examples/bindings.example.yaml

# Local mirror of .github/workflows/ci.yml
ci: check-go fmt-check vet test build
	./bin/pade validate -f spec/examples/web-app.yaml
	./bin/pade plan -f spec/examples/web-app.yaml --json > /tmp/pade-plan.json
	./bin/pade capabilities -f spec/examples/web-app.yaml --bindings spec/examples/bindings.example.yaml --json > /tmp/pade-capabilities.json
	mkdir -p /tmp/pade-orch/.devcontainer
	printf '{}\n' > /tmp/pade-orch/.devcontainer/devcontainer.json
	cp spec/examples/web-app-orchestrated.yaml /tmp/pade-orch/pade.yaml
	./bin/pade validate -f /tmp/pade-orch/pade.yaml
	./bin/pade plan -f /tmp/pade-orch/pade.yaml --json > /tmp/pade-orch-plan.json
