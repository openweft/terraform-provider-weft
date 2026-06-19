package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestHostResource_Metadata(t *testing.T) {
	r := NewHostResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_host" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_host")
	}
}

func TestHostResource_Schema(t *testing.T) {
	r := NewHostResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	want := []string{
		"id", "uuid", "hostname", "az", "rack", "endpoint", "hypervisor",
		"architecture", "network_types", "volume_backends", "properties",
		"state", "created_at", "last_seen_at",
	}
	for _, name := range want {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestHostResource_ImplementsImporter(t *testing.T) {
	r := NewHostResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("hostResource must satisfy ResourceWithImportState")
	}
}
