package provider

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestFrameworkProvider_Metadata(t *testing.T) {
	p := NewFrameworkProvider("test")()
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)
	if resp.TypeName != "weft" {
		t.Fatalf("type name: got %q want %q", resp.TypeName, "weft")
	}
	if resp.Version != "test" {
		t.Fatalf("version: got %q want %q", resp.Version, "test")
	}
}

func TestFrameworkProvider_Schema(t *testing.T) {
	p := NewFrameworkProvider("test")()
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"socket", "ssh_socket", "ssh_key"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing provider attribute %q", name)
		}
	}
}

// TestFrameworkProvider_HasExpectedResources pins the registered resource
// set. Adding a new resource means adjusting this list, which forces a
// maintainer to think about whether the new entry is intentional.
func TestFrameworkProvider_HasExpectedResources(t *testing.T) {
	p := NewFrameworkProvider("test")()
	got := resourceTypeNames(p.Resources(context.Background()))
	want := []string{
		"weft_deployment",
		"weft_endpoint",
		"weft_host",
		"weft_image",
		"weft_image_patch",
		"weft_images",
		"weft_instance",
		"weft_keypair",
		"weft_network",
		"weft_security_group",
		"weft_tenant",
		"weft_volume",
	}
	sort.Strings(got)
	if !equalStrings(got, want) {
		t.Errorf("resources mismatch:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestFrameworkProvider_HasExpectedDataSources(t *testing.T) {
	p := NewFrameworkProvider("test")()
	got := dataSourceTypeNames(p.DataSources(context.Background()))
	want := []string{"weft_config"}
	sort.Strings(got)
	if !equalStrings(got, want) {
		t.Errorf("data sources mismatch:\ngot:  %v\nwant: %v", got, want)
	}
}

func resourceTypeNames(ctors []func() resource.Resource) []string {
	out := make([]string, 0, len(ctors))
	for _, c := range ctors {
		mresp := &resource.MetadataResponse{}
		c().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
		out = append(out, mresp.TypeName)
	}
	return out
}

func dataSourceTypeNames(ctors []func() datasource.DataSource) []string {
	out := make([]string, 0, len(ctors))
	for _, c := range ctors {
		mresp := &datasource.MetadataResponse{}
		c().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
		out = append(out, mresp.TypeName)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
