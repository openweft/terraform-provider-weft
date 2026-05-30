package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestSecurityGroupResource_Metadata(t *testing.T) {
	r := NewSecurityGroupResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_security_group" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_security_group")
	}
}

func TestSecurityGroupResource_Schema(t *testing.T) {
	r := NewSecurityGroupResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	want := []string{
		"id", "uuid", "project", "project_uuid", "name", "description",
		"rules", "created_at",
	}
	for _, name := range want {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestSecurityGroupResource_ImplementsImporter(t *testing.T) {
	r := NewSecurityGroupResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("securityGroupResource must satisfy ResourceWithImportState")
	}
}
