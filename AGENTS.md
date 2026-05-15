[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# AGENTS.md

This file provides guidance to coding agents (e.g. Claude Code, claude.ai/code) when working with code in this repository.

## Repository purpose

Go module `open-cluster-management.io/clusteradm` — the official command-line tool for [Open Cluster Management (OCM)](https://open-cluster-management.io/). It bootstraps and operates an OCM hybrid-cloud control plane:

- `clusteradm init` — install the OCM hub.
- `clusteradm join` / `unjoin` — register or remove a managed cluster (the "klusterlet" side).
- `clusteradm accept` — approve a joining cluster on the hub (signs the CSR).
- `clusteradm get` / `delete` / `create` — CRUD against OCM CRs.
- `clusteradm install` — install hub-side addons.
- `clusteradm addon`, `clusteradm clusterset`, `clusteradm proxy`, `clusteradm upgrade`, `clusteradm clean`, `clusteradm version` — the rest of the subcommand tree.

Distributed as a standalone binary, a `kubectl-clusteradm` plugin (loaded by `kubectl-cli` or `kubectl-krew`), and via an install script (`install.sh`).

Fork is mirrored to `kluster-manager/clusteradm`; **upstream is `open-cluster-management-io/clusteradm`** and this repo tracks it.

## Architecture

- `cmd/clusteradm/main.go` — entry point; assembles the Cobra command tree from `pkg/cmd/`.
- `pkg/cmd/` — **one Cobra subcommand per directory**: `accept/`, `addon/`, `clean/`, `clusterset/`, `create/`, `delete/`, `get/`, `init/`, `install/`, `join/`, `proxy/`, `unjoin/`, `upgrade/`, `version/`. Each typically has `cmd.go`, `exec.go`, `options.go`, an `exec_test.go`, and a `scenario/` directory with embedded manifests/templates.
- `pkg/config/` — bootstrap-token, kubeconfig, and addon config helpers.
- `pkg/genericclioptions/` — extends `k8s.io/cli-runtime/pkg/genericclioptions` with `clusteradm`-flavored flags (hub/spoke kubeconfig, output formats, dry-run).
- `pkg/helpers/` — shared utilities (apply/wait helpers, asset reading, RBAC, version detection, etc.).
- `build/` — build/CI helpers.
- `install.sh` — the one-liner installer used in the README quick start.
- `version.go` + `VERSION.txt` — embedded version metadata.
- `test/`:
  - `e2e/` — end-to-end suite.
  - `integration-test.mk` — integration target included by the main Makefile.
- `vendor/` — checked-in deps.

## Common commands

This repo uses a hand-rolled Makefile with a **local Go toolchain** (no AppsCode Docker harness).

- `make build` — build the `clusteradm` binary into `bin/`.
- `make build-bin` — alternate binary build.
- `make release` — produce release artifacts (used by CI).
- `make install` — `build`, then install into `$GOPATH/bin`.
- `make plugin` — install as a kubectl plugin (`kubectl-clusteradm`).
- `make build-krew` — produce krew-plugin artifacts; needs `make krew-tools`.
- `make krew-tools` — fetch krew packaging tooling.
- `make test` — `deps`, then run Go tests + integration suite (from `test/integration-test.mk`).
- `make verify` — verify scripts/lint.
- `make deps` — fetch test dependencies.
- `make clean` — `clean-test clean-e2e`.
- `make check` (alias `check-copyright`) — verify license headers on all source files.
- `make vendor` / `make copy-crd` — refresh vendor and copy upstream CRDs into the embedded scenario dirs.

Run a single Go test:

```
go test ./pkg/cmd/init/... -run TestName -v
```

End-to-end suite (against a real OCM hub):

```
go test ./test/e2e/... -v
```

## Conventions

- Module path is `open-cluster-management.io/clusteradm` (**upstream**); imports must use that, not the GitHub URL.
- **Upstream-tracking** fork (mirrored as `kluster-manager/clusteradm`). Prefer rebasing onto upstream over diverging; isolate AppsCode-only patches so they replay cleanly.
- License: Apache-2.0 (`LICENSE`); every source file should carry the `// Copyright Contributors to the Open Cluster Management project` header — `make check-copyright` enforces it.
- Sign off commits (`git commit -s`); contributions follow the DCO (`DCO`, `CONTRIBUTING.md`).
- Vendor directory is checked in — keep it tidy.
- Go 1.17+ is required because of `go:embed` use throughout `pkg/cmd/*/scenario/`. Bumping `go.mod` requires double-checking the embed directives still resolve.
- New subcommand: drop a new directory under `pkg/cmd/<name>/` with the canonical `cmd.go`/`exec.go`/`options.go`/`exec_test.go` files, wire it in from `cmd/clusteradm/main.go`, and add a `scenario/` directory for embedded manifests.
- Scenario manifests under `pkg/cmd/*/scenario/` are loaded with `go:embed`; **don't rename files** without updating the matching embed directives.
- The kubectl-plugin entry point relies on the binary being named `kubectl-clusteradm` on `$PATH`; `make plugin` installs it that way.
