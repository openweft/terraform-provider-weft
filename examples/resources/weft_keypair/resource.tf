# weft_keypair is local-only — no gRPC call is made. It reads the public
# key from <file_path>.pub at apply time and exposes it as a computed
# attribute so weft_instance can reference it without hard-coding paths.
resource "weft_keypair" "main" {
  name      = "main"
  file_path = "~/.ssh/id_ed25519"
}

resource "weft_instance" "debian" {
  name = "web-01"
  disk {
    from = "oci://ghcr.io/example/debian:13"
    size = "20Gi"
  }
  ssh {
    user         = "debian"
    keypair_path = weft_keypair.main.file_path
  }
}

output "public_key" {
  value = weft_keypair.main.public_key
}
