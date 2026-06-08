# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`clusteradm` is the CLI for [Open Cluster Management (OCM)](https://open-cluster-management.io/). It drives the OCM hub/managed-cluster lifecycle (`init`, `join`, `accept`, `unjoin`, `clean`), add-ons, cluster sets, placements, and the cluster proxy. The binary is built from `cmd/clusteradm/clusteradm.go` and behaves as a `kubectl`-style tool layered on top of `k8s.io/kubectl`'s factory and cli-runtime.

## Common commands

Build & install:
- `make build` — installs `clusteradm` into `$GOPATH/bin` with version ldflags. Requires Go 1.25+ (see `go.mod`; the README's "Go 1.24+" is out of date).
- `make build-bin` — cross-compiles release tarballs into `bin/`.
- `make plugin` — also installs `oc-clusteradm` and `kubectl-clusteradm` for kubectl/oc plugin use.

Lint & verify:
- `make verify` — runs `go vet ./...` and `golangci-lint` (pinned to v1.64.6, vendor mode, gofmt enabled).
- `make check` / `build/check-copyright.sh` — enforces the copyright header on all source files.

Tests:
- `make test` — unit tests via `build/run-unit-tests.sh`. Iterates over `$GOPACKAGES` (everything except `vendor`, `build`, `test`), runs `go test -cover` per package, and merges coverage. `cmd.go` and `client.go` files are excluded from the merged coverage. Requires envtest assets, fetched by the `envtest-setup` target.
- `go test ./pkg/cmd/<pkg>/...` — run a single package's unit tests directly without coverage merging. Use this for fast iteration; `make test` is what CI runs.
- `make test-integration` — Ginkgo-driven integration tests for `pkg/cmd/addon/{enable,disable}` and `pkg/cmd/install/hubaddon`. Uses envtest (no kind cluster needed). Targets in `test/integration-test.mk`.
- `make test-e2e` — spins up two kind clusters (`clusteradm-e2e-test-hub`, `clusteradm-e2e-test-c1`), installs the local build, and runs the Ginkgo suite in `test/e2e/clusteradm`. Override the OCM version with `BUNDLE_VERSION=latest|default|vX.Y.Z`; filter cases with `GINKGO_LABEL_FILTER=...`. Requires Docker. After modifying e2e tests, `make test-only` re-runs the suite against existing clusters.
- `make clean-e2e` — deletes the kind clusters.

Vendor & CRDs:
- `make vendor` — `go mod tidy && go mod vendor`.
- `make copy-crd` — runs `build/copy-crds.sh` to pull CRDs out of vendored OCM modules.

## Command architecture

Every subcommand follows a strict three-file pattern (see `CONTRIBUTING.md` and `pkg/cmd/version/` for the canonical example):

- `cmd.go` — constructs the cobra command, wires flags, and calls `complete → validate → run` in `RunE`. Exposes `NewCmd(clusteradmFlags, streams)`.
- `options.go` — defines the `Options` struct and `newOptions(...)` constructor.
- `exec.go` — implements `(o *Options) complete / validate / run`.

The root command in `cmd/clusteradm/clusteradm.go` groups subcommands into "General", "Registration", and "Cluster Management" sections via `ktemplates.CommandGroups`. To add a new top-level command, create a package under `pkg/cmd/<name>/`, expose `NewCmd`, and register it in the appropriate group.

### Shared plumbing

- `pkg/genericclioptions/ClusteradmFlags` carries the `cmdutil.Factory`, `DryRun`, `Timeout`, and `Context`. Pass this struct into every subcommand's `Options`; do not stash clients on the options struct — derive them inside `run()` via `o.ClusteradmFlags.KubectlFactory` or the `helpers.GetClients(...)` shortcut.
- Every command **must support `--dry-run`** (project convention). Honor `o.ClusteradmFlags.DryRun` before any apply/write. `pkg/helpers/reader.ResourceReader` already short-circuits applies when constructed with `dryRun=true`, and accumulates the raw YAML for `--output-file`-style flags.
- Logging uses `k8s.io/klog/v2`. Use `klog.V(x).InfoS(...)` for normal messages; `klog.Warning` / `klog.Error` only when truly warranted. Do not use `fmt.Println` for diagnostic output — write to `o.Streams.Out` for user-facing output instead.
- `pkg/genericclioptions/feature_gates.go` defines `HubMutableFeatureGate` / `SpokeMutableFeatureGate`. Wire them with `cmd.Flags()` in `NewCmd` so OCM feature gates can be toggled from the CLI.

### Embedded scenarios

Each command that ships YAML manifests stores them in a `scenario/` subpackage that exposes a `go:embed` `embed.FS`. See `pkg/cmd/install/hubaddon/scenario/resources.go` for the pattern (named filesets keyed by add-on, plus a `Values` struct passed to templates). When adding manifests, put them under the command's `scenario/` directory and add their paths to the relevant slice — `embed.FS` won't pick them up unless the `//go:embed` directive's glob covers them.

### Hub bootstrap & version bundles

`pkg/version/version.go` hard-codes a `versionBundleList` mapping bundle names (e.g. `1.3.1`, `latest`) to component versions (currently `OCM` + `PolicyAddon`), with `defaultBundleVersion` selecting the default. When bumping OCM, update both `defaultBundleVersion` and the corresponding map entry, and add new entries rather than mutating old ones — e2e workflows pin to specific bundle versions. `--bundle-version-overrides` lets users patch the JSON in at runtime.

`init` and `join` render the cluster-manager / klusterlet Helm charts from `open-cluster-management.io/ocm/pkg/operator/helpers/chart` and apply them via `ResourceReader.ApplyRaw`. CRDs are applied before the regular manifests, and `helperwait.WaitUntilCRDReady` is used to gate the second apply. Preserve that ordering when modifying init/join — the operator CRDs need to be present before the operand CRs.

## Helpers worth knowing about

- `pkg/helpers/helpers.go` (`GetClients`, `GetBootstrapToken`, `GetBootstrapTokenFromSA`, `GetExampleHeader`) — shared client construction and token retrieval used by `init`/`join`.
- `pkg/helpers/reader` — the apply pipeline (`ResourceReader.ApplyRaw`).
- `pkg/helpers/wait` — `WaitUntilCRDReady`, `WaitUntilRegistrationOperatorReady`, etc. Use these instead of hand-rolled polling.
- `pkg/helpers/preflight` + per-command `preflight/` packages — register `Checker` instances and run them via `preflight.RunChecks` before mutating anything.
- `pkg/helpers/check` — `CheckForHub` / `CheckForManagedCluster` distinguish hub vs spoke contexts.
- `pkg/config/env.go` — canonical names for OCM namespaces, secrets, and resources (`open-cluster-management`, `ClusterManagerName`, `KlusterletName`, etc.). Reference these constants instead of duplicating string literals.

## Testing notes

- Unit tests live alongside the code (`exec_test.go`, `suite_test.go`). The integration-style Ginkgo suites under `pkg/cmd/addon/*` and `pkg/cmd/install/hubaddon` start envtest in `suite_test.go` — `KUBEBUILDER_ASSETS` must be set (the `envtest-setup` target downloads the binaries and prints the path).
- The e2e harness in `test/e2e/clusteradm` is driven by a `TestE2eConfig` helper that wraps `Clusteradm().<Subcommand>(args...).Run()`. The framework parses common output (tokens, hub API server) and stores it on `e2e.CommandResult()`. After any test that mutates cluster state, call `e2e.ResetEnv()` (initial state: hub initialized, cluster1 joined+accepted) or `e2e.ClearEnv()` for an empty hub. See `test/e2e/README.md`.

## Conventions

- Sign every commit (`git commit -s`) — the repo enforces the DCO via `.github/workflows/dco.yml`.
- All source files need the project copyright header (`build/check-copyright.sh` enforces it; `build/copyright-header.txt` is the template). New Go files start with `// Copyright Contributors to the Open Cluster Management project`.
- Vendored dependencies: this repo vendors. After modifying `go.mod`, run `make vendor` and commit the `vendor/` changes. `golangci-lint` runs in `--modules-download-mode vendor`, so missing vendored sources will fail CI.
