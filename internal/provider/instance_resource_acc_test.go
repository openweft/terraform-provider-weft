//go:build acceptance

// instance_resource_acc_test.go — first concrete TF_ACC test, exercising
// `weft_instance` against a live weft daemon. Documents the pattern for
// acceptance tests of other resources (keypair, image, …); copy this
// file's structure when adding TestAcc<Resource>_basic.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccInstance_basic creates a VM, asserts the computed `ip` + `state`
// attributes are populated by Read, then destroys it. Minimal yet
// end-to-end: covers Create + Read + Delete + the SSH key file reading
// path. Patch operations are exercised by a separate TestAcc.
func TestAccInstance_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfig_basic("acc-vm-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("weft_instance.test", "name", "acc-vm-1"),
					resource.TestCheckResourceAttr("weft_instance.test", "cpu", "2"),
					resource.TestCheckResourceAttr("weft_instance.test", "mem", "2"),
					resource.TestCheckResourceAttrSet("weft_instance.test", "id"),
					// `ip` + `state` are populated by Read; we only check
					// they're set (the actual values are environment-
					// dependent — the test runner doesn't pin the IP).
					resource.TestCheckResourceAttrSet("weft_instance.test", "ip"),
					resource.TestCheckResourceAttrSet("weft_instance.test", "state"),
				),
			},
			{
				// Import roundtrip: name is the import ID.
				ResourceName:      "weft_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				// `disk.from` / `ssh.keypair_path` can't be recovered from
				// the server (they're inputs, not stored on the VM), so
				// the harness diff between import and config would fail.
				// Allow the harness to skip those.
				ImportStateVerifyIgnore: []string{
					"disk.from",
					"ssh.keypair_path",
				},
			},
		},
	})
}

// testAccInstanceConfig_basic is the minimal HCL exercising the resource.
// Operator-style keypair_path points at a known SSH key the runner is
// expected to have; for a CI-style env, set TF_VAR_keypair_path and
// reference it via a Terraform variable (left as a follow-up if/when CI
// runs TF_ACC).
func testAccInstanceConfig_basic(name string) string {
	return fmt.Sprintf(`
provider "weft" {}

resource "weft_instance" "test" {
  name = %q
  cpu  = 2
  mem  = 2

  disk {
    from = "oci://ghcr.io/openweft/weft-test-fixture:debian-arm64"
    size = "20Gi"
  }

  ssh {
    user         = "ubuntu"
    keypair_path = "~/.ssh/id_ed25519"
  }
}
`, name)
}
