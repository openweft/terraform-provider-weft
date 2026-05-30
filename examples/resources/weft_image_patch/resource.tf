# Apply one-time patch operations to cached images. If `images` is omitted,
# the patch fans out to every cached image returned by weft's ListImages RPC.
resource "weft_image_patch" "console" {
  images = [
    "oci://ghcr.io/example/debian:13",
    "oci://ghcr.io/example/ubuntu:24.04",
  ]

  patch {
    add {
      content = <<-EOF
        GRUB_TERMINAL_OUTPUT="console"
        GRUB_CMDLINE_LINUX_DEFAULT="console=tty0 console=hvc0"
      EOF
      dst     = "/etc/default/grub.d/99-console.cfg"
      trigger = "grub-mkconfig"
    }
    del {
      dst = "/etc/motd"
    }
    mod {
      dst = "/etc/ssh/sshd_config"
      old = "^#?PasswordAuthentication.*"
      new = "PasswordAuthentication no"
    }
  }
}
