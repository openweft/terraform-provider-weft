package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestKeypairResource_Metadata(t *testing.T) {
	r := NewKeypairResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_keypair" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_keypair")
	}
}

func TestKeypairResource_Schema(t *testing.T) {
	r := NewKeypairResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	for _, name := range []string{"id", "name", "file_path", "resolved_path", "public_key"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestKeypairResource_ImplementsImporter(t *testing.T) {
	r := NewKeypairResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("keypairResource must satisfy ResourceWithImportState")
	}
}
