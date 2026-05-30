//go:build acceptance

// network_resource_acc_test.go — acceptance scaffold for `weft_network`.
// Exercises CreateNetwork/ListNetworks/DeleteNetwork + the "<project>/<uuid>"
// import id form.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccNetwork_basic creates a NAT network in the default project and
// verifies the server-minted uuid + project_uuid land in state.
func TestAccNetwork_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkConfig_basic("default", "acc-net"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_network.test", "project", "default"),
					resource.TestCheckResourceAttr("weft_network.test", "name", "acc-net"),
					resource.TestCheckResourceAttr("weft_network.test", "cidr", "10.42.0.0/24"),
					resource.TestCheckResourceAttrSet("weft_network.test", "id"),
					resource.TestCheckResourceAttrSet("weft_network.test", "uuid"),
					resource.TestCheckResourceAttrSet("weft_network.test", "project_uuid"),
					resource.TestCheckResourceAttrSet("weft_network.test", "created_at"),
				),
			},
			{
				ResourceName: "weft_network.test",
				ImportState:  true,
				// Network import ID is "<project>/<uuid>" — derive it
				// dynamically from the previous step's state.
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["weft_network.test"]
					if !ok {
						return "", fmt.Errorf("weft_network.test not found in state")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["project"], rs.Primary.Attributes["uuid"]), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNetworkConfig_basic(project, name string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_network" "test" {
  project     = %q
  name        = %q
  cidr        = "10.42.0.0/24"
  gateway     = "10.42.0.1"
  type        = "nat"
  dns_servers = ["1.1.1.1", "9.9.9.9"]
}
`, project, name)
}
