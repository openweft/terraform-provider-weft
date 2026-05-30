package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestImagesResource_Metadata(t *testing.T) {
	r := NewImagesResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_images" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_images")
	}
}

func TestImagesResource_Schema(t *testing.T) {
	r := NewImagesResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	for _, name := range []string{"id", "config_dir", "parallel", "pulled"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestImagesResource_ImplementsImporter(t *testing.T) {
	r := NewImagesResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("imagesResource must satisfy ResourceWithImportState")
	}
}
