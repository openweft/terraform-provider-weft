package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestTenantResource_Metadata(t *testing.T) {
	r := NewTenantResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_tenant" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_tenant")
	}
}

func TestTenantResource_Schema(t *testing.T) {
	r := NewTenantResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	want := []string{
		"id", "uuid", "name", "domain", "status", "projects", "members",
		"admins", "created_at",
	}
	for _, name := range want {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestTenantResource_ImplementsImporter(t *testing.T) {
	r := NewTenantResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("tenantResource must satisfy ResourceWithImportState")
	}
}
