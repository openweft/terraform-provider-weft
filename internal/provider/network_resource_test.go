package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNetworkResource_Metadata(t *testing.T) {
	r := NewNetworkResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_network" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_network")
	}
}

func TestNetworkResource_Schema(t *testing.T) {
	r := NewNetworkResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	want := []string{
		"id", "uuid", "project", "project_uuid", "name", "cidr", "gateway",
		"dns_servers", "type", "default_security_group_uuids", "created_at",
	}
	for _, name := range want {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestNetworkResource_ImplementsImporter(t *testing.T) {
	r := NewNetworkResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("networkResource must satisfy ResourceWithImportState")
	}
}
