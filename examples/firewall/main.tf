terraform {
  required_providers { vpsk3s = { source = "igorovh/k3s-vps-wg", version = "0.1.0" } }
}

provider "vpsk3s" {
  ssh_user             = "root"
  ssh_private_key_path = "~/.ssh/id_ed25519"
}

resource "vpsk3s_cluster" "example" {
  name = "firewall"

  firewall = {
    enabled           = true
    ssh_allowed_cidrs = ["198.51.100.10/32"]
    admin_cidrs       = ["198.51.100.10/32"]
    allow_http        = true
    allow_https       = true
    allow_kube_api    = false
  }

  nodes = [
    { public_ip = "203.0.113.70", role = "server" },
  ]
}
