package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestVolumeResource_Metadata(t *testing.T) {
	r := NewVolumeResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_volume" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_volume")
	}
}

func TestVolumeResource_Schema(t *testing.T) {
	r := NewVolumeResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	want := []string{
		"id", "uuid", "project", "project_uuid", "name", "size_gib",
		"format", "attached_to_uuid", "created_at",
	}
	for _, name := range want {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestVolumeResource_ImplementsImporter(t *testing.T) {
	r := NewVolumeResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("volumeResource must satisfy ResourceWithImportState")
	}
}
