//go:build acceptance

// config_data_source_acc_test.go — acceptance scaffold for the
// `weft_config` data source. Reads the mock HCL config directory and
// verifies the resolved VMs list is populated. Data sources don't import,
// so there's no ImportState step here.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccConfig_basic asserts at least one VM was parsed from the mock
// HCL config so consumers can drive weft_instance with for_each.
func TestAccConfig_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigDataSource_basic(".mock/hcl"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.weft_config.test", "config_dir", ".mock/hcl"),
					resource.TestCheckResourceAttrSet("data.weft_config.test", "id"),
					resource.TestCheckResourceAttrSet("data.weft_config.test", "vms.#"),
				),
			},
		},
	})
}

func testAccConfigDataSource_basic(configDir string) string {
	return fmt.Sprintf(`
provider "weft" {}

data "weft_config" "test" {
  config_dir = %q
}
`, configDir)
}
