//go:build acceptance

// images_resource_acc_test.go — acceptance scaffold for `weft_images`,
// the bulk image-pull resource that walks a weft HCL config directory.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccImages_basic invokes PullImages against the runner's weft HCL
// directory and verifies the synthetic id and the Computed `pulled` list
// surface in state.
func TestAccImages_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImagesConfig_basic("state/hcl", 4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_images.test", "config_dir", "state/hcl"),
					resource.TestCheckResourceAttr("weft_images.test", "parallel", "4"),
					resource.TestCheckResourceAttrSet("weft_images.test", "id"),
					resource.TestCheckResourceAttrSet("weft_images.test", "pulled.#"),
				),
			},
			{
				ResourceName:      "weft_images.test",
				ImportState:       true,
				ImportStateVerify: true,
				// `pulled` is rebuilt from the config directory on Create,
				// not stored server-side, so an Import-from-id can't repop.
				ImportStateVerifyIgnore: []string{
					"pulled",
				},
			},
		},
	})
}

func testAccImagesConfig_basic(configDir string, parallel int) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_images" "test" {
  config_dir = %q
  parallel   = %d
}
`, configDir, parallel)
}
