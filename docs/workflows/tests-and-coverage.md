# CI/CD Workflows

## Overview

The project uses GitHub Actions for continuous integration. Workflows are defined in `.github/workflows/`.

## Tests and Coverage (`test.yaml`)

**Trigger**: Every push to any branch.

**Runner**: `ubuntu-latest`

### Pipeline

```
setup-go (1.23)
    |
install system deps (libjansson-dev, libhiredis-dev)
    |
checkout code
    |
make getdeps (install golangci-lint, zbusc, go mod tidy)
    |
make testrace
    |
    +-- lint (golangci-lint)
    +-- build (CGO_ENABLED=0 go build ./...)
    +-- test (go test, excludes stubs and network packages)
```

### Steps

| Step | What it does |
|------|-------------|
| Set up Go 1.23 | Installs Go toolchain via `actions/setup-go` |
| Prepare dependencies | Installs C libraries needed by CGo bindings (`libjansson-dev`, `libhiredis-dev`) |
| Checkout code | Checks out the repository |
| Get dependencies | Runs `make getdeps` in `pkg/` — installs `golangci-lint`, `zbusc` (zbus stub generator), and runs `go mod tidy` |
| Run tests | Runs `make testrace` in `pkg/` — lint, build, then test |

### Lint Configuration

The linter is configured via `.golangci.yml` at the repository root:

| Linter | What it checks |
|--------|---------------|
| `errcheck` | Unchecked error return values |
| `gofmt` | Code formatting (no trailing whitespace, proper indentation) |
| `govet` | Suspicious constructs (printf format mismatches, struct tag issues, etc.) |
| `ineffassign` | Assignments to variables that are never used |
| `unconvert` | Unnecessary type conversions |
| `staticcheck` | Static analysis (deprecated APIs, unreachable code, etc.) |
| `gocyclo` | Cyclomatic complexity (threshold: 100) |

Lint timeout is set to 20 minutes.

### Build

```bash
CGO_ENABLED=0 go build -v ./...
```

The build disables CGo to ensure all packages compile without C dependencies at build time (C libraries are only needed for testing).

### Tests

```bash
sudo -E go test -vet=off -v $(go list ./... | grep -Ev "stubs|network")
```

- Runs with `sudo` because some tests require elevated privileges (network namespaces, device management)
- `-E` preserves environment variables (Go path, module settings)
- `-vet=off` skips `go vet` during test (already run by the linter)
- Excludes `stubs` packages (auto-generated zbus client code) and `network` packages (require real network namespaces/interfaces)

## Makefile Targets

All targets run from the `pkg/` directory.

| Target | Description |
|--------|-------------|
| `make getdeps` | Install tooling (golangci-lint, zbusc) and tidy modules |
| `make lint` | Run golangci-lint with the repo config |
| `make build` | Build all packages with CGo disabled |
| `make test` | Lint + build + run tests |
| `make testrace` | Same as `test` (race detector not currently enabled) |
| `make generate` | Regenerate zbus client stubs via `go generate` |
