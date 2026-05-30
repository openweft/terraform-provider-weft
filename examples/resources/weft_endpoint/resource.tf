# weft_endpoint is a purely declarative resource — no gRPC call is made.
# Its role is to expose a base URL that downstream weft_image resources
# can reference, mirroring the mock HCL `endpoint` block.
resource "weft_endpoint" "debian" {
  url = "https://cloud.debian.org/images/cloud"
}

resource "weft_endpoint" "ghcr" {
  url = "oci://ghcr.io/example"
}

resource "weft_image" "debian_13" {
  from = "${weft_endpoint.debian.url}/trixie/latest/debian-13-genericcloud-arm64.raw"
}

resource "weft_image" "app" {
  from = "${weft_endpoint.ghcr.url}/app:latest"
}
