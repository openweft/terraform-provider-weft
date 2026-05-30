package provider

import (
	"testing"
)

// ---------------------------------------------------------------------------
// resourceDeployment schema
// ---------------------------------------------------------------------------

func TestResourceDeploymentSchema(t *testing.T) {
	r := resourceDeployment()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceDeployment schema invalid: %v", err)
	}
}

func TestResourceDeploymentSchema_Prefix(t *testing.T) {
	s := resourceDeployment().Schema
	attr, ok := s["prefix"]
	if !ok {
		t.Fatal("schema missing field \"prefix\"")
	}
	if !attr.Required {
		t.Error("\"prefix\" should be Required")
	}
	if !attr.ForceNew {
		t.Error("\"prefix\" should be ForceNew")
	}
}

func TestResourceDeploymentCreate(t *testing.T) {
	res := resourceDeployment()
	d := res.Data(nil)
	d.Set("prefix", "mock-M19B3D62C")

	diags := resourceDeploymentCreate(nil, d, nil) //nolint:staticcheck
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "mock-M19B3D62C" {
		t.Errorf("ID = %q, want %q", d.Id(), "mock-M19B3D62C")
	}
}
