resource "weft_instance" "debian" {
  name = "web-01"
  cpu  = 2
  mem  = 2

  disk {
    from = "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-genericcloud-arm64.raw"
    size = "20Gi"

    # Optional: per-instance disk patches applied before first boot.
    patch {
      add {
        content = "weft.local\n"
        dst     = "/etc/hostname"
      }
      mod {
        dst = "/etc/default/grub"
        old = "GRUB_TIMEOUT=5"
        new = "GRUB_TIMEOUT=0"
      }
    }
  }

  ssh {
    user         = "debian"
    keypair_path = "~/.ssh/id_ed25519"
  }
}

output "instance_ip" {
  value = weft_instance.debian.ip
}
