---
page_title: "VPS K3s Provider"
---

# VPS K3s Provider

`vpsk3s` installs K3s on existing Debian/Ubuntu VPS nodes through SSH.

Provider source:

```hcl
source = "igorovh/k3s-vps-wg"
```

The provider exposes one resource: `vpsk3s_cluster`.

## Provider Configuration

```hcl
provider "vpsk3s" {
  ssh_user                 = "root"
  ssh_port                 = 22
  ssh_private_key_path     = "~/.ssh/id_ed25519"
  ssh_timeout              = "5m"
  insecure_ignore_host_key = true
}
```

Use exactly one SSH auth method where possible. If several are configured, priority is inline private key, private key path, then password.

## Local Development

Use `make build` plus Terraform `dev_overrides`, or run `make install-local` to install into Terraform's local plugin directory under `registry.terraform.io/igorovh/k3s-vps-wg`.

See `docs/resources/cluster.md` for the full resource reference.
