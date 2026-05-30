# A weft_deployment is a naming scope shared by a group of weft_instance
# resources. It carries no VMs itself — its only role is the `prefix`
# attribute used to generate instance names of the form:
#     <prefix>-<label>-<index>
resource "weft_deployment" "main" {
  prefix = "M19B3D62C"
}

resource "weft_instance" "debian" {
  count = 3
  name  = "${weft_deployment.main.prefix}-debian-${count.index + 1}"
  cpu   = 2
  mem   = 2

  disk {
    from = "oci://ghcr.io/example/debian:13"
    size = "20Gi"
  }

  ssh {
    user         = "debian"
    keypair_path = "~/.ssh/id_ed25519"
  }
}
