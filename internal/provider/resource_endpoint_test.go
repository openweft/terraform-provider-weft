package provider

import (
	"testing"
)

func TestResourceEndpointSchema(t *testing.T) {
	r := resourceEndpoint()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceEndpoint schema invalid: %v", err)
	}
}

func TestResourceEndpointSchema_Fields(t *testing.T) {
	s := resourceEndpoint().Schema

	attr, ok := s["url"]
	if !ok {
		t.Fatal("schema missing required field \"url\"")
	}
	if !attr.Required {
		t.Error("field \"url\" should be Required")
	}
	if !attr.ForceNew {
		t.Error("field \"url\" should be ForceNew")
	}

	if _, ok := s["name"]; ok {
		t.Error("schema should not have a \"name\" field — use the Terraform label")
	}
}

func TestResourceEndpointCreate(t *testing.T) {
	res := resourceEndpoint()
	d := res.Data(nil)
	d.Set("url", "https://cloud.debian.org/images/cloud/")

	diags := resourceEndpointCreate(nil, d, nil) //nolint:staticcheck
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "https://cloud.debian.org/images/cloud/" {
		t.Errorf("ID = %q, want %q", d.Id(), "https://cloud.debian.org/images/cloud/")
	}
}
