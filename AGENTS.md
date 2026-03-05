# AGENTS.md - Coding Agent Guidelines

## Project Overview

Kubernetes operator (controller-runtime) that manages vSphere capacity through three CRDs:
**Lease**, **Pool**, and **Network** under API group `vspherecapacitymanager.splat.io/v1`.

- **Language**: Go 1.22 (`go.mod` specifies `go 1.22.0`, toolchain `go1.22.2`)
- **Module**: `github.com/openshift-splat-team/vsphere-capacity-manager`
- **Framework**: `sigs.k8s.io/controller-runtime` v0.17.3
- **Dependencies are vendored** in `vendor/` and committed to the repo. Use `go mod vendor` after changing deps.

## Build Commands

```sh
make build               # Build binary to bin/vsphere-capacity-manager
make test                # Run all tests (sets up envtest, runs ginkgo -r)
make all                 # check + build + test
make generate            # Regenerate CRDs, RBAC, deepcopy, go generate
make image               # Build container image via podman
make deploy              # Deploy CRDs + configs + deployment to OpenShift cluster
```

Build details: `CGO_ENABLED=0`, static binary, ldflags inject version/commit.

## Testing

Two test layers, both run by `make test` (which invokes `ginkgo -r` recursively):

### Unit tests (`pkg/` directories)
Standard Go table-driven tests using `testing.T`.

```sh
# Run all tests
make test

# Run a single unit test by name
go test ./pkg/controller/ -run TestDoesLeaseContainPortGroup -v
go test ./pkg/utils/ -run TestGetFittingPools -v

# Run all unit tests in a package
go test ./pkg/utils/ -v
go test ./pkg/controller/ -v
```

### Integration tests (`test/` directory)
Ginkgo BDD tests using `envtest` (real API server). Requires envtest binaries.

```sh
# Run integration tests via ginkgo (preferred)
KUBEBUILDER_ASSETS="$(go run vendor/sigs.k8s.io/controller-runtime/tools/setup-envtest use 1.29 -p path --bin-dir bin)" \
  go run vendor/github.com/onsi/ginkgo/v2/ginkgo -r ./test/

# Run a single integration test by description
KUBEBUILDER_ASSETS="$(go run vendor/sigs.k8s.io/controller-runtime/tools/setup-envtest use 1.29 -p path --bin-dir bin)" \
  go run vendor/github.com/onsi/ginkgo/v2/ginkgo --focus "should fulfill a small lease" ./test/

# Or use go test with ginkgo focus
KUBEBUILDER_ASSETS="..." go test ./test/ -v -ginkgo.focus "should fulfill a small lease"
```

### Linting

```sh
# Run golangci-lint (no project-level config; uses defaults, 30m timeout in CI)
go run vendor/github.com/golangci/golangci-lint/cmd/golangci-lint run --timeout=30m
```

## Code Style

### Import Ordering (enforced by `gci` via `hack/go-fmt.sh`)

Four groups separated by blank lines, in this order:
1. Standard library (`context`, `fmt`, `log`, `time`, etc.)
2. Third-party (`k8s.io/...`, `sigs.k8s.io/...`, `github.com/prometheus/...`)
3. Internal project packages (`github.com/openshift-splat-team/...`)
4. Blank imports (if any)

```go
import (
    "context"
    "fmt"
    "log"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"

    v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
    "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/utils"
)
```

### Formatting

- Use `gofmt -s` (standard Go formatting).
- Run `hack/go-fmt.sh` for full formatting pass (gofmt + gci import ordering).

### Naming Conventions

| Category | Convention | Examples |
|---|---|---|
| Phase/strategy constants | `SCREAMING_SNAKE_CASE` | `PHASE_FULFILLED`, `RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED` |
| Condition type constants | `PascalCase` | `LeaseConditionTypeFulfilled`, `ConditionSeverityError` |
| Labels/finalizers | kebab-case strings | `"vsphere-capacity-manager.splat-team.io/lease-finalizer"` |
| CRD types | `PascalCase` | `Lease`, `Pool`, `Network`, `LeaseSpec`, `PoolStatus` |
| Reconciler structs | `<Resource>Reconciler` | `LeaseReconciler`, `PoolReconciler` |
| Package aliases | Short aliases | `v1` for API types, `ctrl` for controller-runtime, `configv1` for OpenShift API |

### Error Handling

- Wrap errors with context: `fmt.Errorf("context description: %w", err)`
- Use `client.IgnoreNotFound(err)` for Kubernetes resource lookups that may 404
- Return `ctrl.Result{}` or `ctrl.Result{RequeueAfter: duration}` from reconcilers
- Use `log.Printf()` for logging (standard library `log`, not structured logging)
- Only use `os.Exit(1)` in `cmd/main.go`; everywhere else return errors up the stack

### Controller Pattern

- Each reconciler embeds `client.Client` and implements `Reconcile(ctx, req) (ctrl.Result, error)`
- Register controllers via `SetupWithManager(mgr)` method
- Global mutex (`reconcileLock` in `pkg/controller/context.go`) serializes reconciliation
- In-memory maps (`pools`, `leases`, `networks`) hold shared state, protected by mutex
- Conditions follow the Cluster API pattern (custom `Condition` type with Type/Status/Severity/Reason/Message)
- Finalizers used for cleanup on deletion
- Status subresource: separate `Update()` and `Status().Update()` calls

### Testing Patterns

**Unit tests** - table-driven with `t.Run()`:
```go
func TestFoo(t *testing.T) {
    tests := []struct {
        name     string
        input    SomeType
        expected bool
    }{
        {name: "descriptive case name", input: ..., expected: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := functionUnderTest(tt.input)
            if got != tt.expected {
                t.Errorf("functionUnderTest() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

**Integration tests** - Ginkgo BDD with envtest:
- Use `Describe/It/By/Eventually` pattern
- `BeforeEach` starts controller-runtime manager with all reconcilers
- `AfterEach` stops manager and cleans up resources
- Builder pattern for test fixtures: `GetLease().WithShape(SHAPE_SMALL).WithPool("...").Build()`
- Use `Eventually()` for polling async reconciliation results
- Dot-import Ginkgo and Gomega: `. "github.com/onsi/ginkgo/v2"` and `. "github.com/onsi/gomega"`

## Project Structure

```
cmd/main.go                  Entry point (manager setup, controller registration)
pkg/apis/.../v1/             CRD type definitions, deepcopy, constants
pkg/controller/              Reconciler implementations (leases, pools, networks, namespaces)
pkg/controller/context.go    Shared state (mutex, global maps)
pkg/controller/metrics.go    Prometheus metrics
pkg/utils/                   Utility functions (pool fitting, conditions)
test/                        Integration tests (Ginkgo + envtest)
test/manifests/              YAML fixtures for integration tests
hack/                        Build/dev scripts
config/crd/bases/            Generated CRD YAML manifests
vendor/                      Vendored Go dependencies (committed)
```

## License Header

All Go source files must include the Apache 2.0 license header from `hack/boilerplate.go.txt`:
```go
/*
Copyright 2022 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
...
*/
```

## CI Checks (GitHub Actions)

All run on pull requests:
1. **lint** - `golangci-lint` with `--timeout=30m`
2. **test** - Ginkgo tests with envtest (Go 1.21, envtest K8s 1.29)
3. **vendor** - `go mod verify`
4. **build** - Container image build with Buildah
