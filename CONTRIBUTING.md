# Contributing to quay-builder

## Setup

```bash
# Requires podman or docker
make build

# Run worker locally
./bin/quay-builder --config worker.yaml
```

## Development

Build worker that:
- Pulls Dockerfiles from Quay build triggers
- Executes builds via Podman/Docker
- Pushes results back to Quay

## Testing

```bash
# Unit tests
make test

# Integration test (requires Quay instance)
./test/integration.sh

# Local build test
./bin/quay-builder build --dockerfile testdata/Dockerfile
```

## Build Context

Supports multiple context sources:
- Git (GitHub, GitLab, Bitbucket)
- Tarball upload
- Inline Dockerfile

## Pull Requests

- Test with both Podman and Docker
- Update buildpack logic tests
- Security: validate Dockerfile sources, no arbitrary command injection
- Performance: benchmark large builds (>1GB context)

## Code Structure

- `cmd/quay-builder/` - worker entrypoint
- `buildctx/` - context fetchers (git, tar, http)
- `buildpack/` - Docker/Podman build execution
- `containerclient/` - container runtime abstraction
- `rpc/` - gRPC API to Quay
