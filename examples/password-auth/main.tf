terraform {
  required_providers { vpsk3s = { source = "igorovh/k3s-vps-wg", version = "0.1.1" } }
}

variable "ssh_password" {
  type      = string
  sensitive = true
}

provider "vpsk3s" {
  ssh_user     = "root"
  ssh_password = var.ssh_password
}

resource "vpsk3s_cluster" "example" {
  name = "password-auth"
  nodes = [
    { public_ip = "203.0.113.30", role = "server" },
  ]
}
