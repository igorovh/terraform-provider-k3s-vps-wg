terraform {
  required_providers { vpsk3s = { source = "igorovh/k3s-vps-wg", version = "0.1.0" } }
}

provider "vpsk3s" {
  ssh_user             = "root"
  ssh_private_key_path = "~/.ssh/id_ed25519"
}

resource "vpsk3s_cluster" "example" {
  name = "traefik-helm"

  addons = {
    traefik = {
      mode      = "install"
      namespace = "traefik"
    }
  }

  nodes = [
    { public_ip = "203.0.113.90", role = "server" },
  ]
}
