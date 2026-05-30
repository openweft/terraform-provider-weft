terraform {
  required_providers {
    weft = {
      source  = "openweft/weft"
      version = "~> 0.1"
    }
  }
}

# Local mode: connect to the weft daemon over a Unix socket.
provider "weft" {
  socket = "unix:///home/me/.weft/weft.sock"
}

# Optional: connect to a remote weft daemon over the SSH transport.
# provider "weft" {
#   ssh_socket = "~/.weft/weft-ssh.sock"
#   ssh_key    = "~/.ssh/id_ed25519"
# }

resource "weft_endpoint" "debian" {
  url = "https://cloud.debian.org/images/cloud/trixie/latest"
}

resource "weft_image" "debian_13" {
  from     = "${weft_endpoint.debian.url}/debian-13-genericcloud-arm64.raw"
  checksum = "${weft_endpoint.debian.url}/SHA512SUMS"
}

resource "weft_instance" "web" {
  name = "web-01"
  cpu  = 2
  mem  = 2

  disk {
    from = weft_image.debian_13.from
    size = "20Gi"
  }

  ssh {
    user         = "debian"
    keypair_path = "~/.ssh/id_ed25519"
  }
}
