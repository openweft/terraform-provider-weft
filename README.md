<p align="center"><img src="https://raw.githubusercontent.com/openweft/brand/main/social/openweft.png" alt="openweft" width="720"></p>

# terraform-provider-weft

Terraform provider for `weft`. Provisions and manages VMs via the weft gRPC API, with optional SSH transport for remote access.

## Module

```
github.com/openweft/terraform-provider-weft
```

## Resources

| Resource | Description |
|----------|-------------|
| `weft_instance` | VM lifecycle (create = ProvisionVM, delete = DeprovisionVM) |
| `weft_deployment` | Group of VM instances from an HCL config |
| `weft_image` | Pull and cache a single image |
| `weft_images` | Pull all images referenced in an HCL config directory |
| `weft_endpoint` | Named endpoint (image registry URL) |
| `weft_keypair` | SSH keypair for VM injection |

## Data sources

| Data source | Description |
|-------------|-------------|
| `weft_config` | Load VM definitions from an HCL config directory |

## Provider configuration

```hcl
provider "weft" {
  socket             = "~/.weft/weft.sock"
  ssh_socket         = "~/.weft/weft-ssh.sock"  # optional
  ssh_key            = "~/.ssh/id_ed25519"     # optional
}
```

## Example

```hcl
resource "weft_instance" "web" {
  name    = "web-01"
  image   = "oci://ghcr.io/org/debian:latest"
  cpu     = 2
  mem_mb  = 2048
  disk_gb = 20
}
```

## Build

```sh
go build -mod=mod -o terraform-provider-weft .
```

## Install (local dev)

```sh
task install
```

Installs the binary to `~/.terraform.d/plugins/registry.terraform.io/openweft/weft/0.1.0/darwin_arm64/terraform-provider-weft`.

## Related

- [`weft`](../weft) — daemon
- [`weft-proto`](../weft-proto) — gRPC service definition
- [`ssh`](../../grpc-transports/ssh) — SSH transport
