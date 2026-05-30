package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceEndpoint implements weft_endpoint, which mirrors the mock HCL `endpoint` block.
//
// It is a purely declarative resource: no gRPC call is made. Its purpose is to
// declare a base URL so that weft_image resources can reference it via
// weft_endpoint.<name>.url, exactly as mock HCL uses endpoint.<name>.url in
// images blocks.
//
// Example:
//
//	resource "weft_endpoint" "debian" {
//	  url = "https://cloud.debian.org/images/cloud/"
//	}
//
//	resource "weft_image" "debian_13" {
//	  from = "${weft_endpoint.debian.url}/trixie/latest/debian-13-genericcloud-arm64.raw"
//	}
func resourceEndpoint() *schema.Resource {
	return &schema.Resource{
		Description:   "Declares a base URL endpoint — mirrors the mock HCL `endpoint` block. Use weft_endpoint.<name>.url to reference the URL in weft_image.",
		CreateContext: resourceEndpointCreate,
		ReadContext:   resourceEndpointRead,
		DeleteContext: resourceEndpointDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Base URL — matches mock HCL endpoint.url.",
			},
		},
	}
}

func resourceEndpointCreate(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	d.SetId(d.Get("url").(string))
	return nil
}

func resourceEndpointRead(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	return nil
}

func resourceEndpointDelete(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	d.SetId("")
	return nil
}
