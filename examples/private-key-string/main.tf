terraform {
  required_providers { vpsk3s = { source = "igorovh/k3s-vps-wg", version = "0.1.0" } }
}

variable "ssh_private_key" {
  type      = string
  sensitive = true
}

provider "vpsk3s" {
  ssh_user        = "root"
  ssh_private_key = var.ssh_private_key
}

resource "vpsk3s_cluster" "example" {
  name = "inline-key"
  nodes = [
    { public_ip = "203.0.113.40", role = "server" },
  ]
}
