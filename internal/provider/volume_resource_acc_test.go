//go:build acceptance

// volume_resource_acc_test.go — acceptance scaffold for `weft_volume`.
// Exercises CreateVolume + the "<project>/<uuid>" import id form.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccVolume_basic creates a 10 GiB raw volume and verifies the
// server-minted uuid + format default surface in state.
func TestAccVolume_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolumeConfig_basic("default", "acc-vol", 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_volume.test", "project", "default"),
					resource.TestCheckResourceAttr("weft_volume.test", "name", "acc-vol"),
					resource.TestCheckResourceAttr("weft_volume.test", "size_gib", "10"),
					resource.TestCheckResourceAttrSet("weft_volume.test", "id"),
					resource.TestCheckResourceAttrSet("weft_volume.test", "uuid"),
					resource.TestCheckResourceAttrSet("weft_volume.test", "project_uuid"),
					resource.TestCheckResourceAttrSet("weft_volume.test", "format"),
					resource.TestCheckResourceAttrSet("weft_volume.test", "created_at"),
				),
			},
			{
				ResourceName: "weft_volume.test",
				ImportState:  true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["weft_volume.test"]
					if !ok {
						return "", fmt.Errorf("weft_volume.test not found in state")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["project"], rs.Primary.Attributes["uuid"]), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccVolumeConfig_basic(project, name string, sizeGiB int) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_volume" "test" {
  project  = %q
  name     = %q
  size_gib = %d
  format   = "raw"
}
`, project, name, sizeGiB)
}
