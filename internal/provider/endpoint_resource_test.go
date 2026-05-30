package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestEndpointResource_Metadata(t *testing.T) {
	r := NewEndpointResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_endpoint" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_endpoint")
	}
}

func TestEndpointResource_Schema(t *testing.T) {
	r := NewEndpointResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	for _, name := range []string{"id", "url"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestEndpointResource_ImplementsImporter(t *testing.T) {
	r := NewEndpointResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("endpointResource must satisfy ResourceWithImportState")
	}
}
