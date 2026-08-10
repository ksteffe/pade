# Prefer a repo-local Go toolchain when present (.tools/go from go.dev).
# System Go 1.13 and similar cannot build this module (no embed stdlib, old modules).
GO ?= $(shell if [ -x "$(CURDIR)/.tools/go/bin/go" ]; then echo "$(CURDIR)/.tools/go/bin/go"; else command -v go; fi)

.PHONY: check-go test build validate plan

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

test: check-go
	$(GO) test ./...

build: check-go
	$(GO) build -o bin/pade ./cmd/pade

validate: check-go
	$(GO) run ./cmd/pade validate -f spec/examples/web-app.yaml

plan: check-go
	$(GO) run ./cmd/pade plan -f spec/examples/web-app.yaml
