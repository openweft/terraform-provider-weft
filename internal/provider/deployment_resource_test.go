package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDeploymentResource_Metadata(t *testing.T) {
	r := NewDeploymentResource()
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_deployment" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_deployment")
	}
}

func TestDeploymentResource_Schema(t *testing.T) {
	r := NewDeploymentResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	for _, name := range []string{"id", "prefix"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestDeploymentResource_ImplementsImporter(t *testing.T) {
	r := NewDeploymentResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatalf("deploymentResource must satisfy ResourceWithImportState")
	}
}
