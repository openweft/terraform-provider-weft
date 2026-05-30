# weft_image pulls and caches an image locally via the weft daemon, then
# (optionally) patches the cached copy once so every VM cloned from it
# inherits the changes — no per-instance disk.patch block needed.
resource "weft_image" "debian_13" {
  from     = "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-genericcloud-arm64.raw"
  checksum = "https://cloud.debian.org/images/cloud/trixie/latest/SHA512SUMS"

  patch {
    add {
      content = <<-EOF
        GRUB_TERMINAL_OUTPUT="console"
        GRUB_CMDLINE_LINUX_DEFAULT="console=tty0 console=hvc0"
      EOF
      dst     = "/etc/default/grub.d/99-console.cfg"
      trigger = "grub-mkconfig"
    }
  }
}

resource "weft_instance" "debian" {
  name = "web-01"
  disk {
    from = weft_image.debian_13.from
  }
  ssh {
    user         = "debian"
    keypair_path = "~/.ssh/id_ed25519"
  }
}
