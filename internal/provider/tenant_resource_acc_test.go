//go:build acceptance

// tenant_resource_acc_test.go — acceptance scaffold for `weft_tenant`.
// Exercises CreateTenant / ListTenants / DeleteTenant. Import id is the
// raw UUID (no project prefix; tenants are top-level isolation units).

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTenant_basic creates a tenant with name + domain and verifies
// the Computed counters / status / uuid surface in state.
func TestAccTenant_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTenantConfig_basic("acc-tenant", "acc.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_tenant.test", "name", "acc-tenant"),
					resource.TestCheckResourceAttr("weft_tenant.test", "domain", "acc.example.com"),
					resource.TestCheckResourceAttrSet("weft_tenant.test", "id"),
					resource.TestCheckResourceAttrSet("weft_tenant.test", "uuid"),
					resource.TestCheckResourceAttrSet("weft_tenant.test", "status"),
					resource.TestCheckResourceAttrSet("weft_tenant.test", "projects"),
					resource.TestCheckResourceAttrSet("weft_tenant.test", "members"),
					resource.TestCheckResourceAttrSet("weft_tenant.test", "admins"),
					resource.TestCheckResourceAttrSet("weft_tenant.test", "created_at"),
				),
			},
			{
				ResourceName:      "weft_tenant.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTenantConfig_basic(name, domain string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_tenant" "test" {
  name   = %q
  domain = %q
}
`, name, domain)
}
