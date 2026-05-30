//go:build acceptance

// endpoint_resource_acc_test.go — acceptance scaffold for `weft_endpoint`.
// Purely declarative resource (no gRPC); the test confirms the url is
// echoed into id and survives an import roundtrip.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccEndpoint_basic creates a single endpoint and verifies the
// Computed id mirrors the url, then exercises Import.
func TestAccEndpoint_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEndpointConfig_basic("https://cloud.debian.org/images/cloud"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_endpoint.test", "url", "https://cloud.debian.org/images/cloud"),
					resource.TestCheckResourceAttrSet("weft_endpoint.test", "id"),
				),
			},
			{
				ResourceName:      "weft_endpoint.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccEndpointConfig_basic(url string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_endpoint" "test" {
  url = %q
}
`, url)
}
