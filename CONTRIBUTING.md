# Contributing

Thanks for interest in PADE. The project is early (Milestone 0: specification and scaffolding).

## Before you change code or schema

1. Read [README.md](README.md), [RFC.md](RFC.md), and [DESIGN.md](DESIGN.md) (including the DevPod-first revisions).
2. Prefer composing Dev Containers / DevPod / existing secret managers over new PADE subsystems.
3. Keep secrets out of manifests, logs, plan output, and git.

## Pull requests

- Keep changes focused and explained against the design hypotheses.
- Update docs and [spec/](spec/) when behavior or schema changes.
- Do not commit `.env`, credential files, or `.pade/` state.

## License

By contributing, you agree that your contributions are licensed under the Apache License 2.0.
