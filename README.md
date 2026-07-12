# terraform-provider-k3s-vps-wg

Terraform provider that creates a K3s cluster on existing Debian or Ubuntu VPS nodes over SSH. It can configure WireGuard, UFW, admin peers, and Helm addons.

## Quick Start

```hcl
terraform {
  required_providers {
    vpsk3s = {
      source  = "igorovh/k3s-vps-wg"
      version = "0.1.0"
    }
  }
}

provider "vpsk3s" {
  ssh_user             = "root"
  ssh_private_key_path = "~/.ssh/id_ed25519"
}

resource "vpsk3s_cluster" "test" {
  name = "test"

  nodes = [
    {
      public_ip = "203.0.113.10"
      role      = "server"
      hostname  = "k3s-1"
    }
  ]
}

output "kubeconfig" {
  value     = vpsk3s_cluster.test.kubeconfig
  sensitive = true
}
```

Run `terraform init` and `terraform apply`. The target node must already exist and be reachable over SSH.

## Examples and Docs

- [Examples](examples/): minimal cluster, SSH authentication, WireGuard, firewall, admin peers, Traefik, and Longhorn.
- [Provider configuration](docs/index.md)
- [`vpsk3s_cluster` resource reference](docs/resources/cluster.md)

## Defaults

- `wireguard.enabled = true`
- `wireguard.interface = "wg0"`
- `wireguard.subnet = "10.10.0.0/24"`
- `wireguard.port = 51820`
- `wireguard.mtu = 1420`
- `k3s.channel = "stable"`
- `k3s.cluster_cidr = "10.42.0.0/16"`
- `k3s.service_cidr = "10.43.0.0/16"`
- `k3s.write_kubeconfig_mode = "0644"`
- `k3s.disable_components = []`
- `k3s.install_open_iscsi = true`
- `k3s.install_nfs_common = true`
- `firewall.enabled = false`
- `addons.longhorn.enabled = false`
- `addons.traefik.mode = "default"`

With WireGuard enabled, K3s node traffic and kubeconfig use the WireGuard IP of the first server node. To use `kubectl` locally, your workstation must reach that private network, typically through `admin_peer`.

For teams, prefer `admin_peers` with developer-provided public keys:

```hcl
resource "vpsk3s_cluster" "test" {
  name = "test"

  nodes = [
    { public_ip = "203.0.113.10", role = "server", hostname = "k3s-1" },
  ]

  admin_peers = {
    igor = {
      public_key = "IGOR_WIREGUARD_PUBLIC_KEY"
      wg_ip      = "10.10.0.250"
    }

    pipeline = {
      public_key = var.pipeline_wireguard_public_key
      wg_ip      = "10.10.0.254"
    }
  }
}
```

If `public_key` is provided, the provider does not know or store that peer's private key. If `public_key` is omitted, the provider generates a keypair and returns the config in sensitive output `admin_wireguard_configs`.

If `wireguard.enabled = false`, K3s uses public IPs for `node-ip`, `advertise-address`, and kubeconfig. In that mode, node-to-node K3s traffic goes over public addresses and you must handle firewalling carefully.

## SSH Authentication

The provider supports:

- `ssh_private_key` as a sensitive string.
- `ssh_private_key_path` from disk, with `~` expansion.
- `ssh_password` password authentication.

Auth priority is:

1. `ssh_private_key`
2. `ssh_private_key_path`
3. `ssh_password`

Each node can override global SSH settings with `ssh_user`, `ssh_port`, `ssh_password`, `ssh_private_key`, and `ssh_private_key_path`.

## Security Notes

Terraform state contains sensitive material: SSH passwords, SSH private keys if provided inline, WireGuard private keys, the generated K3s token, kubeconfig, and admin WireGuard config. Use encrypted remote state and restrict access to state files.

For developer and pipeline WireGuard access, prefer generating keys outside Terraform and passing only `admin_peers.<name>.public_key`. That keeps private keys out of Terraform state.

If enabling the firewall, make sure SSH remains allowed from your admin network. The provider adds SSH rules before enabling UFW, but incorrect CIDRs can still lock you out.

## Local Development

Build the provider:

```bash
make build
```

Use Terraform `dev_overrides` in `~/.terraformrc` or `terraform.rc`:

```hcl
provider_installation {
  dev_overrides {
    "igorovh/k3s-vps-wg" = "/absolute/path/to/terraform-provider-k3s-vps-wg/bin"
  }

  direct {}
}
```

Then use:

```hcl
terraform {
  required_providers {
    vpsk3s = {
      source = "igorovh/k3s-vps-wg"
    }
  }
}
```

You can also install into Terraform's local plugin directory:

```bash
make install-local
```

This copies the binary to Terraform's local plugin directory:

```text
<terraform-plugin-root>/registry.terraform.io/igorovh/k3s-vps-wg/0.1.0/<os>_<arch>/terraform-provider-k3s-vps-wg_v0.1.0
```

On Windows the plugin root is `%APPDATA%/terraform.d/plugins`. On Linux and macOS it is `~/.terraform.d/plugins`.

## Requirements On VPS Nodes

- Debian or Ubuntu.
- Root SSH access, or a user with passwordless sudo-compatible privileges. Current implementation assumes commands can run directly as the SSH user.
- `apt-get`, `systemd`, `curl`, and network access to download packages and K3s.
- Public UDP connectivity between nodes for WireGuard if WireGuard is enabled.

## Server Vs Agent Nodes

K3s has two node roles: `server` and `agent`.

`server` nodes run the Kubernetes control plane. They host components such as the Kubernetes API server, scheduler, controller manager, and, in HA setups, the embedded etcd datastore. Server nodes are the cluster's control layer.

`agent` nodes are worker nodes. They join the cluster and run application workloads, but they do not run the Kubernetes control plane or embedded etcd.

For a small test cluster, one `server` node is enough. For a highly available control plane, use an odd number of `server` nodes, usually three. The provider rejects an even number of server nodes because embedded etcd needs quorum, and two server nodes are not a useful HA setup.

Common layouts:

```text
1 server
```

Small test or development cluster.

```text
3 servers
```

Highly available control plane where server nodes can also run workloads.

```text
3 servers + N agents
```

Highly available control plane with separate worker capacity for applications.

## Known Limitations

- Destroy is best-effort and stops K3s/WireGuard services but does not wipe nodes.
- Refresh does not SSH into nodes; remote drift is handled during apply.
- Firewall backend support is currently limited to UFW.
- Helm addons are installed by executing Helm and kubectl on the first server node over SSH.
- No cloud provider API is used. Servers must already exist.

See `docs/` and `examples/` for more details.
