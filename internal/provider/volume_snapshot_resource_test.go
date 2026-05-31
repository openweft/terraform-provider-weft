package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestVolumeSnapshotResource_Metadata(t *testing.T) {
	r := NewVolumeSnapshotResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, resp)
	if resp.TypeName != "weft_volume_snapshot" {
		t.Fatalf("type name: got %q want weft_volume_snapshot", resp.TypeName)
	}
}

func TestVolumeSnapshotResource_Schema(t *testing.T) {
	r := NewVolumeSnapshotResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "uuid", "volume_uuid", "name", "project", "size_gib", "created_at"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestVolumeSnapshotResource_ImplementsImporter(t *testing.T) {
	r := NewVolumeSnapshotResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("volumeSnapshotResource must satisfy ResourceWithImportState")
	}
}
