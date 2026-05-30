// instance_resource_test.go — schema-level tests for the framework-based
// weft_instance. Acceptance tests (TestAcc*) would require a real weft
// daemon and live in a separate file behind a TF_ACC env gate; landing
// those is a follow-up milestone (see FRAMEWORK_MIGRATION.md "what's left").

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestInstanceResource_Metadata(t *testing.T) {
	r := NewInstanceResource()
	mreq := resource.MetadataRequest{ProviderTypeName: "weft"}
	mresp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), mreq, mresp)
	if mresp.TypeName != "weft_instance" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_instance")
	}
}

func TestInstanceResource_Schema(t *testing.T) {
	r := NewInstanceResource()
	sresp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	// Required top-level attrs.
	for _, name := range []string{"id", "name", "cpu", "mem", "ip", "state"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	// Required nested blocks.
	for _, name := range []string{"disk", "ssh"} {
		if _, ok := sresp.Schema.Blocks[name]; !ok {
			t.Errorf("missing block %q", name)
		}
	}
}

func TestInstanceResource_ImplementsImporter(t *testing.T) {
	// The framework's ResourceWithImportState surface — we don't assert
	// the cast against the package's `resource.ResourceWithImportState`
	// because the embedded interface is satisfied implicitly; just call
	// ImportState() to make sure it's there.
	r := NewInstanceResource()
	importer, ok := r.(resource.ResourceWithImportState)
	if !ok {
		t.Fatalf("instanceResource must satisfy ResourceWithImportState")
	}
	// We don't drive a real ImportState here — that requires a fully-
	// configured framework server. The compile-time check above is the
	// load-bearing part of this test.
	_ = importer
}
