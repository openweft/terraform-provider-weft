//go:build acceptance

// keypair_resource_acc_test.go — acceptance scaffold for `weft_keypair`.
// Local-only resource: no gRPC dial; the test verifies the public key is
// loaded from <file_path>.pub and surfaced as a Computed attribute.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccKeypair_basic checks that Create reads ~/.ssh/id_ed25519.pub off
// the runner's filesystem and exposes resolved_path + public_key. The
// import roundtrip ignores file_path since it isn't recoverable from state.
func TestAccKeypair_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeypairConfig_basic("acc-keypair"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_keypair.test", "name", "acc-keypair"),
					resource.TestCheckResourceAttr("weft_keypair.test", "file_path", "~/.ssh/id_ed25519"),
					resource.TestCheckResourceAttrSet("weft_keypair.test", "id"),
					resource.TestCheckResourceAttrSet("weft_keypair.test", "resolved_path"),
					resource.TestCheckResourceAttrSet("weft_keypair.test", "public_key"),
				),
			},
			{
				ResourceName:      "weft_keypair.test",
				ImportState:       true,
				ImportStateVerify: true,
				// file_path / resolved_path / public_key are derived from the
				// runner's filesystem, not the server — they can't survive an
				// ImportState roundtrip from id alone.
				ImportStateVerifyIgnore: []string{
					"file_path",
					"resolved_path",
					"public_key",
				},
			},
		},
	})
}

func testAccKeypairConfig_basic(name string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_keypair" "test" {
  name      = %q
  file_path = "~/.ssh/id_ed25519"
}
`, name)
}
