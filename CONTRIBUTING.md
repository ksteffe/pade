# Contributing

Thanks for interest in PADE. The project is early (Milestone 5: identity separation; see [docs/identity-separation.md](docs/identity-separation.md)).

## Before you change code or schema

1. Read [README.md](README.md), [RFC.md](RFC.md), and [DESIGN.md](DESIGN.md) (including the DevPod-first revisions).
2. Prefer composing Dev Containers / DevPod / existing secret managers over new PADE subsystems.
3. Keep secrets out of manifests, logs, plan output, and git.

## Development

Requires **Go 1.22+**. Older toolchains (for example Homebrew Go 1.13) fail on `embed` and modern module requirements.

```bash
# Prefer a current SDK on PATH, or the repo-local toolchain:
export PATH="$(pwd)/.tools/go/bin:$PATH"

go test ./...
go run ./cmd/pade validate -f spec/examples/web-app.yaml
go run ./cmd/pade plan -f spec/examples/web-app.yaml

# Or:
make test
make ci   # same checks as GitHub Actions
```

Pull requests and pushes to `main` run [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

## Pull requests

- Keep changes focused and explained against the design hypotheses.
- Update docs and [spec/](spec/) when behavior or schema changes.
- Do not commit `.env`, credential files, or `.pade/` state.

## License

By contributing, you agree that your contributions are licensed under the Apache License 2.0.
