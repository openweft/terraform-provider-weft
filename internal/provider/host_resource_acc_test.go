//go:build acceptance

// host_resource_acc_test.go — acceptance scaffold for `weft_host`.
// Exercises RegisterHost / GetHost / DeleteHost end-to-end.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHost_basic registers a host with the minimum required attrs and
// verifies the server-minted uuid + state Computed attrs land in state.
func TestAccHost_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostConfig_basic("acc-host-01"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_host.test", "hostname", "acc-host-01"),
					resource.TestCheckResourceAttr("weft_host.test", "hypervisor", "qemu-kvm"),
					resource.TestCheckResourceAttr("weft_host.test", "az", "dc1"),
					resource.TestCheckResourceAttr("weft_host.test", "rack", "r01"),
					resource.TestCheckResourceAttr("weft_host.test", "endpoint", "10.0.0.10:7777"),
					resource.TestCheckResourceAttrSet("weft_host.test", "id"),
					resource.TestCheckResourceAttrSet("weft_host.test", "uuid"),
					resource.TestCheckResourceAttrSet("weft_host.test", "state"),
					resource.TestCheckResourceAttrSet("weft_host.test", "created_at"),
				),
			},
			{
				ResourceName:      "weft_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccHost_withProperties covers the properties map + multi-value network_types
// to make sure list/map attrs are correctly round-tripped through the
// proto.
func TestAccHost_withProperties(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostConfig_withProperties("acc-host-02"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_host.test", "hostname", "acc-host-02"),
					resource.TestCheckResourceAttr("weft_host.test", "properties.tier", "edge"),
					resource.TestCheckResourceAttr("weft_host.test", "network_types.#", "2"),
					resource.TestCheckResourceAttrSet("weft_host.test", "uuid"),
				),
			},
		},
	})
}

func testAccHostConfig_basic(hostname string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_host" "test" {
  hostname     = %q
  hypervisor   = "qemu-kvm"
  architecture = "arm64"
  az           = "dc1"
  rack         = "r01"
  endpoint     = "10.0.0.10:7777"
}
`, hostname)
}

func testAccHostConfig_withProperties(hostname string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_host" "test" {
  hostname     = %q
  hypervisor   = "qemu-kvm"
  architecture = "arm64"
  az           = "dc1"
  rack         = "r02"
  endpoint     = "10.0.0.11:7777"

  network_types   = ["nat", "bridged"]
  volume_backends = ["file"]
  properties = {
    tier = "edge"
    role = "compute"
  }
}
`, hostname)
}
