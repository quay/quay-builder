# Agent Guide: quay-builder

## Purpose

Single-job Quay build worker. It connects to Quay build manager over gRPC, fetches build context, builds with Podman or Docker, pushes the image, reports logs/status, and exits.

## Start Here

- Entrypoint: `cmd/quay-builder/main.go`
- Context fetch: `buildctx/`
- Build execution: `buildpack/`
- Runtime abstraction: `containerclient/`
- Quay gRPC client: `rpc/`
- Protocol bindings: `buildman_pb/`

## Common Tasks

- Add context source: update `buildctx/`, request parsing, and tests.
- Debug build failure: check context fetch, Dockerfile parse, runtime logs, gRPC phase/log updates.
- Change runtime behavior: use `containerclient/`; do not call Podman/Docker directly from unrelated packages.

## Commands

```bash
make test
make build
make build-ubi8
```

## Guardrails

- Keep build directories isolated and cleaned up.
- Do not leak registry credentials or build args into logs.
- Resource limits are enforced outside the worker; avoid bypassing them in code.
- `quay/quay` owns the build manager server under `buildman/`.
