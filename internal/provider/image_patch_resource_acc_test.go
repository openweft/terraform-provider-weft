//go:build acceptance

// image_patch_resource_acc_test.go — acceptance scaffold for
// `weft_image_patch`. The schema has no ImportState method, so no Import
// step here (unlike most other resources in this suite).

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccImagePatch_basic targets an explicit image list and verifies a
// synthetic id is minted plus the patch.add block survives Read.
func TestAccImagePatch_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImagePatchConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_image_patch.test", "images.#", "1"),
					resource.TestCheckResourceAttr("weft_image_patch.test", "patch.add.#", "1"),
					resource.TestCheckResourceAttrSet("weft_image_patch.test", "id"),
				),
			},
		},
	})
}

func testAccImagePatchConfig_basic() string {
	return `
provider "weft" {}

resource "weft_image_patch" "test" {
  images = [
    "oci://ghcr.io/openweft/weft-test-fixture:debian-arm64",
  ]

  patch {
    add {
      content = "PasswordAuthentication no\n"
      dst     = "/etc/ssh/sshd_config.d/99-acc.conf"
    }
  }
}
`
}
