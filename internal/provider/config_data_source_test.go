package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestConfigDataSource_Metadata(t *testing.T) {
	d := NewConfigDataSource()
	mresp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "weft"}, mresp)
	if mresp.TypeName != "weft_config" {
		t.Fatalf("type name: got %q want %q", mresp.TypeName, "weft_config")
	}
}

func TestConfigDataSource_Schema(t *testing.T) {
	d := NewConfigDataSource()
	sresp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	for _, name := range []string{"id", "config_dir", "vms"} {
		if _, ok := sresp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}
