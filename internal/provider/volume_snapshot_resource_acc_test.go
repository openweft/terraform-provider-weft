//go:build acceptance

// volume_snapshot_resource_acc_test.go — acceptance scaffold for the
// `weft_volume_snapshot` resource. Exercises CreateVolumeSnapshot via
// a freshly-created parent volume + the snapshot's own import-by-uuid
// path. Companion to volume_resource_acc_test.go.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccVolumeSnapshot_basic creates a parent volume, snapshots it,
// and verifies the snapshot's computed fields (uuid, size_gib,
// created_at) land in state. Then exercises ImportState by uuid.
func TestAccVolumeSnapshot_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolumeSnapshotConfig_basic("acc-snap-parent", "acc-snap-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("weft_volume.parent", "uuid"),
					resource.TestCheckResourceAttrSet("weft_volume_snapshot.test", "uuid"),
					resource.TestCheckResourceAttr("weft_volume_snapshot.test", "name", "acc-snap-1"),
					// size_gib is computed — assert it surfaces.
					resource.TestCheckResourceAttrSet("weft_volume_snapshot.test", "size_gib"),
					// created_at is computed (unix-ns); assert non-empty.
					resource.TestCheckResourceAttrSet("weft_volume_snapshot.test", "created_at"),
					// volume_uuid binds parent.
					resource.TestCheckResourceAttrPair("weft_volume_snapshot.test", "volume_uuid",
						"weft_volume.parent", "uuid"),
				),
			},
			{
				// Import roundtrip by uuid. project + volume_uuid get
				// re-fetched by the next Read (via ListVolumeSnapshots),
				// so they're not part of the import id — same shape as
				// weft_host's import.
				ResourceName:                         "weft_volume_snapshot.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func testAccVolumeSnapshotConfig_basic(parentName, snapName string) string {
	return fmt.Sprintf(`
resource "weft_volume" "parent" {
  name     = %q
  size_gib = 10
  format   = "raw"
}

resource "weft_volume_snapshot" "test" {
  volume_uuid = weft_volume.parent.uuid
  name        = %q
}
`, parentName, snapName)
}
