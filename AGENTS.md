# AGENTS.md

## Repo Shape
- This is a Go Terraform provider plugin, not a CLI/server; entrypoint is `main.go`, provider address is `registry.terraform.io/igorovh/k3s-vps-wg`, provider type is `vpsk3s`.
- All provider logic lives in `internal/provider`; the only resource is `vpsk3s_cluster`, and there are no data sources.
- Docs are handwritten in `README.md` and `docs/resources/cluster.md`; `make docs` only prints a reminder.

## Commands
- Build: `make build` or `go build -o bin/terraform-provider-k3s-vps-wg .`.
- Test all unit tests: `make test` or `go test ./...`.
- Run one test/package: `go test ./internal/provider -run TestAssignWireGuardIPsStable`.
- Format Go files: `make fmt`.
- Lint: `make lint` runs `go vet ./...`.
- Local Terraform plugin install needs an explicit version: `make install-local VERSION=0.1.0`; without `VERSION`, the Makefile builds an invalid plugin path/name.

## Local Terraform Use
- Prefer Terraform `dev_overrides` pointing at this repo's `bin` directory after `make build`.
- `make install-local VERSION=...` copies into Terraform's plugin cache under `registry.terraform.io/igorovh/k3s-vps-wg/<version>/<os>_<arch>/`.
- Ignore `local-test/.terraform/` and state/tfvars files; they are local Terraform artifacts and are gitignored.

## Provider Behavior To Preserve
- `Read` intentionally does not SSH during refresh; remote drift is handled during apply.
- `Delete` is best-effort only: it stops `k3s`, `k3s-agent`, and `wg-quick@<interface>` but does not wipe nodes.
- Apply order is WireGuard setup/connectivity, firewall, K3s install, addons, then kubeconfig fetch from the first server.
- Node identity is `hostname` when set, otherwise `public_ip`; this drives stable WireGuard IPs and stored private keys.
- Server count must be odd and at least one server; even server counts are rejected for embedded etcd quorum.
- WireGuard is enabled by default; when enabled, kubeconfig/server URL uses the first server's WireGuard IP.
- `admin_peer` and `admin_peers` are mutually exclusive and both require WireGuard.

## Remote Host Assumptions
- Target nodes are existing Debian/Ubuntu VPS hosts reached over SSH; commands assume `apt-get`, `systemd`, `curl`, and direct privilege as the SSH user.
- SSH auth priority is inline private key, private key path, then password; `ssh_private_key_path` expands `~`.
- `insecure_ignore_host_key` defaults to true for bootstrap usage.
- Firewall support is UFW only; enabling `allow_kube_api` with no `admin_cidrs` intentionally warns that 6443/tcp is public.

## Sensitive State
- Terraform state stores generated K3s token, WireGuard node private keys, generated admin peer private keys, kubeconfig, and SSH/password values if configured.
- For `admin_peers`, providing `public_key` means no private key is generated or stored for that peer.

## Releases
- `.goreleaser.yml` runs `go mod tidy` before release builds and produces zipped provider binaries for Linux, macOS, and Windows on amd64/arm64.
- Keep the hardcoded development version in `main.go` aligned with documented examples when changing release/version guidance.
