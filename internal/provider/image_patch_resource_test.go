package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestImagePatchResource_Metadata(t *testing.T) {
	r := NewImagePatchResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_image_patch" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_image_patch")
	}
}

func TestImagePatchResource_Schema(t *testing.T) {
	r := NewImagePatchResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	for _, name := range []string{"id", "images"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if _, ok := sresp.Schema.Blocks["patch"]; !ok {
		t.Errorf("missing block %q", "patch")
	}
}
