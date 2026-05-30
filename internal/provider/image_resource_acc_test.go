//go:build acceptance

// image_resource_acc_test.go — acceptance scaffold for `weft_image`.
// Exercises PullImage on Create (no patch block in the basic case) and
// the import roundtrip keyed on the `from` URL.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccImage_basic pulls a fixture image and verifies id mirrors `from`.
func TestAccImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImageConfig_basic("oci://ghcr.io/openweft/weft-test-fixture:debian-arm64"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_image.test", "from", "oci://ghcr.io/openweft/weft-test-fixture:debian-arm64"),
					resource.TestCheckResourceAttrSet("weft_image.test", "id"),
				),
			},
			{
				ResourceName:      "weft_image.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccImage_withPatch covers the additional code path where a patch
// block triggers a PatchImage call after the initial Pull.
func TestAccImage_withPatch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImageConfig_withPatch("oci://ghcr.io/openweft/weft-test-fixture:debian-arm64"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_image.test", "from", "oci://ghcr.io/openweft/weft-test-fixture:debian-arm64"),
					resource.TestCheckResourceAttr("weft_image.test", "patch.add.#", "1"),
					resource.TestCheckResourceAttrSet("weft_image.test", "id"),
				),
			},
		},
	})
}

func testAccImageConfig_basic(from string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_image" "test" {
  from = %q
}
`, from)
}

func testAccImageConfig_withPatch(from string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_image" "test" {
  from = %q

  patch {
    add {
      content = "weft.local\n"
      dst     = "/etc/hostname"
    }
  }
}
`, from)
}
