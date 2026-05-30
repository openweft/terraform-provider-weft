//go:build acceptance

// security_group_resource_acc_test.go — acceptance scaffold for
// `weft_security_group`. Exercises CreateSecurityGroup + the
// "<project>/<uuid>" import id form.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccSecurityGroup_basic creates a no-rules SG and verifies the
// server-minted uuid surfaces in state.
func TestAccSecurityGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_basic("default", "acc-sg"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_security_group.test", "project", "default"),
					resource.TestCheckResourceAttr("weft_security_group.test", "name", "acc-sg"),
					resource.TestCheckResourceAttr("weft_security_group.test", "description", "acceptance test SG"),
					resource.TestCheckResourceAttrSet("weft_security_group.test", "id"),
					resource.TestCheckResourceAttrSet("weft_security_group.test", "uuid"),
					resource.TestCheckResourceAttrSet("weft_security_group.test", "project_uuid"),
					resource.TestCheckResourceAttrSet("weft_security_group.test", "created_at"),
				),
			},
			{
				ResourceName: "weft_security_group.test",
				ImportState:  true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["weft_security_group.test"]
					if !ok {
						return "", fmt.Errorf("weft_security_group.test not found in state")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["project"], rs.Primary.Attributes["uuid"]), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccSecurityGroup_withRules covers the rules list to make sure
// ingress/egress entries land in SetSecurityGroupRules and round-trip
// through ListSecurityGroups on Read.
func TestAccSecurityGroup_withRules(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupConfig_withRules("default", "acc-sg-rules"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_security_group.test", "name", "acc-sg-rules"),
					resource.TestCheckResourceAttr("weft_security_group.test", "rules.#", "2"),
					resource.TestCheckResourceAttr("weft_security_group.test", "rules.0.direction", "ingress"),
					resource.TestCheckResourceAttr("weft_security_group.test", "rules.0.protocol", "tcp"),
					resource.TestCheckResourceAttrSet("weft_security_group.test", "uuid"),
				),
			},
		},
	})
}

func testAccSecurityGroupConfig_basic(project, name string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_security_group" "test" {
  project     = %q
  name        = %q
  description = "acceptance test SG"
}
`, project, name)
}

func testAccSecurityGroupConfig_withRules(project, name string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_security_group" "test" {
  project     = %q
  name        = %q
  description = "acceptance test SG with rules"

  rules = [
    {
      direction         = "ingress"
      protocol          = "tcp"
      port_min          = 22
      port_max          = 22
      remote_cidr       = "0.0.0.0/0"
      remote_group_uuid = ""
    },
    {
      direction         = "egress"
      protocol          = "any"
      port_min          = 0
      port_max          = 0
      remote_cidr       = "0.0.0.0/0"
      remote_group_uuid = ""
    },
  ]
}
`, project, name)
}
