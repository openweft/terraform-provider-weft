package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceDeployment implements weft_deployment, a naming-scope resource that
// provides a shared prefix for a group of weft_instance resources.
//
// It holds no VMs itself; its role is to carry the prefix so that instances
// can reference it and generate consistent names:
//
//	<deployment>-<label>-<index>  e.g.  M19B3D62C-debian-1
//
// Example:
//
//	resource "weft_deployment" "main" {
//	  prefix = "M19B3D62C"
//	}
//
//	resource "weft_instance" "debian" {
//	  count = 3
//	  name  = "${weft_deployment.main.prefix}-debian-${count.index + 1}"
//	  # → M19B3D62C-debian-1, -2, -3
//	}
func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		Description:   "Naming scope for a group of weft_instance resources. Provides a shared prefix used to generate instance names: <prefix>-<label>-<index>.",
		CreateContext: resourceDeploymentCreate,
		ReadContext:   resourceDeploymentRead,
		DeleteContext: resourceDeploymentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"prefix": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Naming prefix shared by all instances in this deployment (e.g. \"M19B3D62C\").",
			},
		},
	}
}

func resourceDeploymentCreate(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	d.SetId(d.Get("prefix").(string))
	return nil
}

func resourceDeploymentRead(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	return nil
}

func resourceDeploymentDelete(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	d.SetId("")
	return nil
}
