terraform {
  required_providers { vpsk3s = { source = "igorovh/k3s-vps-wg", version = "0.1.0" } }
}

provider "vpsk3s" {
  ssh_user             = "root"
  ssh_private_key_path = "~/.ssh/id_ed25519"
}

resource "vpsk3s_cluster" "example" {
  name = "longhorn"

  addons = {
    longhorn = {
      enabled = true
      storage_classes = {
        critical = {
          name            = "longhorn-critical-r2"
          number_replicas = 2
          reclaim_policy  = "Retain"
          default_class   = false
        }
        cheap = {
          name            = "longhorn-cheap-r1"
          number_replicas = 1
          reclaim_policy  = "Delete"
          default_class   = false
        }
      }
    }
  }

  nodes = [
    { public_ip = "203.0.113.80", role = "server", hostname = "k3s-server-1" },
    { public_ip = "203.0.113.81", role = "agent", hostname = "k3s-agent-1" },
  ]
}
