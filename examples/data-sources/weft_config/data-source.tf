# weft_config parses the weft HCL config directory (same format as
# state/hcl) and returns a fully-resolved list of VMs, suitable for
# driving weft_instance resources with for_each — without duplicating
# the configuration in both HCL flavours.
data "weft_config" "weft" {
  config_dir = "state/hcl"
}

resource "weft_instance" "vms" {
  for_each = { for vm in data.weft_config.weft.vms : vm.name => vm }
  name     = each.key
  cpu      = each.value.cpu
  mem      = each.value.mem

  disk {
    from = each.value.image
    size = each.value.disk_size
  }

  ssh {
    user         = each.value.ssh_user
    keypair_path = each.value.keypair_path
  }
}
