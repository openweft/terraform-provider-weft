# Bulk-pulls every image referenced by the weft HCL config directory via
# weft's PullImages RPC. Useful when seeding a fresh weft daemon before
# applying the weft_instance resources that consume those images.
resource "weft_images" "all" {
  config_dir = "state/hcl"
  parallel   = 4
}

output "pulled_images" {
  value = weft_images.all.pulled
}
