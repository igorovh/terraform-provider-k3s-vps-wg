terraform {
  required_providers { vpsk3s = { source = "igorovh/k3s-vps-wg", version = "0.1.1" } }
}

provider "vpsk3s" {
  ssh_user             = "root"
  ssh_private_key_path = "~/.ssh/id_ed25519"
}

resource "vpsk3s_cluster" "example" {
  name = "public-network"

  wireguard = {
    enabled = false
  }

  nodes = [
    { public_ip = "203.0.113.20", role = "server" },
  ]
}
