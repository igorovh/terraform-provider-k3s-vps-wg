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

resource "vpsk3s_cluster" "example" {
  name = "minimal"

  nodes = [
    { public_ip = "203.0.113.10", role = "server", hostname = "k3s-1" },
    { public_ip = "203.0.113.11", role = "server", hostname = "k3s-2" },
    { public_ip = "203.0.113.12", role = "server", hostname = "k3s-3" },
  ]
}

output "kubeconfig" {
  value     = vpsk3s_cluster.example.kubeconfig
  sensitive = true
}
