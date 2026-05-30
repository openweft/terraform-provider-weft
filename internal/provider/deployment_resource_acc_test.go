//go:build acceptance

// deployment_resource_acc_test.go — acceptance scaffold for
// `weft_deployment`. No-gRPC naming-scope resource; the test confirms
// the `prefix` is echoed into id and survives an import roundtrip.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDeployment_basic creates a deployment scope and verifies the
// Computed id mirrors the prefix.
func TestAccDeployment_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig_basic("M19B3D62C"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_deployment.test", "prefix", "M19B3D62C"),
					resource.TestCheckResourceAttr("weft_deployment.test", "id", "M19B3D62C"),
				),
			},
			{
				ResourceName:      "weft_deployment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDeploymentConfig_basic(prefix string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_deployment" "test" {
  prefix = %q
}
`, prefix)
}
