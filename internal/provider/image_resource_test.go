package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestImageResource_Metadata(t *testing.T) {
	r := NewImageResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_image" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_image")
	}
}

func TestImageResource_Schema(t *testing.T) {
	r := NewImageResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	for _, name := range []string{"id", "from", "checksum"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if _, ok := sresp.Schema.Blocks["patch"]; !ok {
		t.Errorf("missing block %q", "patch")
	}
}

func TestImageResource_ImplementsImporter(t *testing.T) {
	r := NewImageResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("imageResource must satisfy ResourceWithImportState")
	}
}
