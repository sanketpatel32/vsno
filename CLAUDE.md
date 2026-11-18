# CLAUDE.md

Guidance for working in this repository.

## Requirements

- Go 1.27+

## Build

```bash
make build      # binary with version from git tags
make test       # all tests, verbose
make lint       # golangci-lint (config in .golangci.yml; gocyclo limit 15)
make vet
make fmt        # gofmt + goimports
make coverage
```

## Architecture

Cobra CLI, same layout as `sebrandon1/succulent-cli`.

- `cmd/`: one file per command, plus `_test.go`.
  - `root.go`: global flags `--kubeconfig`, `--context`, `-n/--namespace` (default `vsno`),
    `-o/--output table|json`.
  - `precheck.go`: renders `precheck.Report` as a table or JSON, and supports
    `--write-config`.
- `lib/kube`: builds client-go typed and dynamic clients from a kubeconfig (explicit path, then
  `$KUBECONFIG`, then `~/.kube/config`).
- `lib/precheck`: the cluster checks.

### `lib/precheck` files

| File | Role |
|---|---|
| `types.go` | `Status`, `Result`, `Report`, `Suggested`, and `Requirements` (defaults: 2 VMs × 16 cores / 32 GiB, a 600 GB disk, guest CIDRs) |
| `probe.sh` | Embedded node probe script. It runs via `chroot /host` and prints `KEY=VALUE` lines (`VIRT`, `KVM`, `NPROC`, `MEM_*`, `LSBLK`, `BYID`). It must stay read-only. |
| `probe.go` | `ParseProbe` turns that output into `NodeHardware`/`Disk`. `Disk.PreferredByID` prefers `wwn-*`. |
| `prober.go` | `PodProber` creates a temporary privileged namespace, one pod per node, reads the logs, then deletes the namespace. |
| `checks.go` | Individual checks over `Env` (Kube, Dyn, `Prober`). `EvaluateNode` is pure and tested with fixtures. |
| `run.go` | `Run` orders the checks, evaluates workers, picks the best node, and fills `Suggested`. |
| `cidr.go`, `auth.go` | Pure helpers: CIDR overlap, and registry credential lookup |

### Behaviour to preserve

- Memory requested by pods in the vsno namespace counts as **available**, because an existing
  environment would be reused or replaced.
- The "StorageClass `lvms-vg-vsno` already exists" shortcut only applies to nodes whose CSINode
  lists the `topolvm.io` driver.
- When at least one node is eligible, failures on the other nodes are demoted to `info` with a
  `not usable:` prefix.

## Tests

- `lib/precheck/testdata/probe-*.txt` hold real probe output from a lab cluster:
  - an R740 with an empty disk (derived from the R740 capture)
  - the same R740 with its disk already taken by LVMS
  - a VM worker with nested KVM
- `run_test.go` builds a fake cluster with `k8s.io/client-go/kubernetes/fake` and
  `dynamic/fake`, plus a `fixtureProber`.

## Plan

Milestones (see the vSNO plan): 1 precheck ✅ → 2 setup/teardown/status → 3 create (spike
guest access through the KubeVirt VMI port-forward) → 4 prepare / seed generate / upgrade
(eco-goinfra lca/oadp builders) → 5 run + release.

The manifests for the later milestones come from the manual procedure verified on 2026-11-18. The
author has them locally; ask before assuming their location.
