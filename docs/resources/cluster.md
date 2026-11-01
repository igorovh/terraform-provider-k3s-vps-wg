---
page_title: "Resource: vpsk3s_cluster"
subcategory: "Cluster"
---

# vpsk3s_cluster

Creates and manages a K3s cluster on existing VPS nodes.

## Minimal Usage

```hcl
resource "vpsk3s_cluster" "test" {
  name = "test"

  nodes = [
    { public_ip = "203.0.113.10", role = "server", hostname = "k3s-1" },
    { public_ip = "203.0.113.11", role = "server", hostname = "k3s-2" },
    { public_ip = "203.0.113.12", role = "server", hostname = "k3s-3" },
  ]
}
```

## Arguments

- `name` is required.
- `nodes` is required. Stable identity is `hostname` when set, otherwise `public_ip`; this drives WireGuard IP assignment.
- `wireguard` is optional and enabled by default.
- `k3s` is optional.
- `firewall` is optional and disabled by default.
- `addons` is optional.
- `admin_peer` is optional and requires WireGuard.
- `admin_peers` is optional and is intended for multiple developer or pipeline WireGuard peers.
- `dns` is optional. When set, K3s uses a provider-managed `resolv.conf`.
- `sysctl` is optional and sets kernel parameters on every node.

## Node Object

```hcl
nodes = [
  {
    public_ip            = "203.0.113.10"
    role                 = "server"
    hostname             = "k3s-1"
    ssh_user             = "root"
    ssh_port             = 22
    ssh_password         = null
    ssh_private_key      = null
    ssh_private_key_path = null
  }
]
```

`role` must be `server` or `agent`. At least one server is required, and the number of servers must be odd.

### Server Vs Agent

`server` nodes run the Kubernetes control plane: API server, scheduler, controller manager, and embedded etcd for HA clusters.

`agent` nodes are workers. They run application pods but do not run the Kubernetes control plane or embedded etcd.

Use one `server` for a simple test cluster. Use three `server` nodes for a highly available control plane. Add `agent` nodes when you want additional worker capacity without adding more control-plane members.

The provider rejects an even number of `server` nodes because embedded etcd depends on quorum. Use `1`, `3`, `5`, etc.

## WireGuard

```hcl
wireguard = {
  enabled   = true
  interface = "wg0"
  subnet    = "10.10.0.0/24"
  port      = 51820
  mtu       = 1420
}
```

When enabled, each node receives an automatically assigned stable WireGuard IP. K3s uses WireGuard IPs for node traffic and kubeconfig points at the first server's WireGuard IP.

## K3s

```hcl
k3s = {
  channel               = "stable"
  cluster_cidr          = "10.42.0.0/16"
  service_cidr          = "10.43.0.0/16"
  disable_components    = ["servicelb"]
  write_kubeconfig_mode = "0644"
  install_open_iscsi    = true
  install_nfs_common    = true
  extra_server_args     = []
  extra_agent_args      = []
}
```

K3s config is written to `/etc/rancher/k3s/config.yaml` with mode `0600`.

`token` (optional, sensitive) sets the join token used by every node. Without it the provider
generates one and keeps it in state as `k3s_token`. To rotate the token of a running cluster:

1. Run `k3s token rotate --new-token <new>` on one server.
2. Set `k3s.token = "<new>"` and apply. Every node gets the new token and restarts K3s.

Applying a new `token` without step 1 breaks servers: they cannot decrypt the datastore bootstrap
data with the new token and fail to start. The plan shows a warning when `token` differs from
the token in state. Etcd snapshots taken before the rotation still need the old token to restore.

## DNS

Some VPS images add a `search` domain to the host network config, for example through cloud-init and netplan. K3s copies the host `resolv.conf` into CoreDNS and into pods with `dnsPolicy: Default`, and pods use `ndots:5`. If that domain has wildcard DNS, lookups of external names such as `s3.example.com` can resolve to the wrong host.

```hcl
dns = {
  nameservers    = []
  search_domains = []
}
```

Setting `dns` (even `dns = {}`) makes the provider write `/etc/vpsk3s/resolv.conf` on every node and set `resolv-conf` in the K3s config:

- `nameservers`: up to 3 IP addresses. When empty, the provider reads the node's upstream nameservers from `/run/systemd/resolve/resolv.conf` (or `/etc/resolv.conf`) and skips loopback addresses such as the `127.0.0.53` stub.
- `search_domains`: search domains for CoreDNS and `dnsPolicy: Default` pods. Default: none, so host search domains are ignored.

The file is refreshed on every apply and K3s does not restart when only the file changes. Adding or removing the `dns` block changes `config.yaml`, so K3s restarts once on each node. Running pods keep their old `resolv.conf` until they are recreated. Do not also pass `resolv-conf` in `extra_server_args` or `extra_agent_args`.

The provider does not change the host's own resolver. To stop cloud-init from writing the network config again, disable it on the host (`network: {config: disabled}` in `/etc/cloud/cloud.cfg.d/`).

## Sysctl

```hcl
sysctl = {
  "net.mptcp.enabled" = "0"
}
```

Values are written to `/etc/sysctl.d/90-vpsk3s.conf` and applied with `sysctl -p` on every apply, so they also persist across reboots. An unknown key fails the apply. Removing the map deletes the file, but the current runtime values stay until the next reboot.

## Firewall

```hcl
firewall = {
  enabled           = true
  backend           = "ufw"
  ssh_allowed_cidrs = ["198.51.100.10/32"]
  admin_cidrs       = []
  allow_http        = true
  allow_https       = true
  allow_kube_api    = false
  extra_tcp_ports   = []
  extra_udp_ports   = []
}
```

Only UFW is implemented. Be careful not to block SSH.

## Addons

Traefik modes:

- `default`: leave K3s packaged Traefik unchanged.
- `disabled`: add `traefik` to K3s disabled components.
- `install`: disable packaged Traefik and install Traefik through Helm.

Longhorn is disabled by default. If enabled and no storage classes are provided, two storage classes are created: `longhorn-critical-r2` and `longhorn-cheap-r1`.

## Admin Peer

```hcl
admin_peer = {
  enabled = true
  name    = "admin-laptop"
}
```

The generated `admin_wireguard_config` output is sensitive. The provider does not install anything on the admin machine.

## Multiple Admin Peers

Use `admin_peers` for teams and automation:

```hcl
admin_peers = {
  igor = {
    public_key = "IGOR_WIREGUARD_PUBLIC_KEY"
    wg_ip      = "10.10.0.250"
  }

  ania = {
    public_key = "ANIA_WIREGUARD_PUBLIC_KEY"
    wg_ip      = "10.10.0.251"
  }

  pipeline = {
    public_key = var.pipeline_wireguard_public_key
    wg_ip      = "10.10.0.254"
  }
}
```

If `public_key` is set, the provider adds the peer to all VPS WireGuard configs but never stores the peer private key. Each developer or pipeline should keep its private key locally or in CI secrets.

If `public_key` is omitted, the provider generates a keypair and returns a ready client config in:

```hcl
vpsk3s_cluster.example.admin_wireguard_configs["peer-name"]
```

Do not configure `admin_peer` and `admin_peers` at the same time. Use `admin_peer` only as a simple single-user shortcut.

## Computed Attributes

- `kubeconfig` sensitive string.
- `server_url`.
- `nodes` computed fields: `name`, `private_ip`, `wireguard_public_key`, `status`.
- `admin_wireguard_config` sensitive string.
- `admin_wireguard_configs` sensitive map for generated `admin_peers` configs.
- `wireguard_enabled`.
- `firewall_enabled`.
- `addons_status`.

Internal sensitive attributes are stored in state for idempotency: `k3s_token`, `wireguard_private_keys`, and admin WireGuard keys.
