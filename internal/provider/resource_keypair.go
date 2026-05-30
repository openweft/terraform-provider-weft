package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceKeypair implements weft_keypair, which mirrors the mock HCL `keypair` block.
//
// It is a local-only resource: no gRPC call is made. It reads the public key
// from <file_path>.pub at plan/apply time and exposes it as a computed attribute
// so that weft_instance can reference it without hard-coding file paths.
//
// Example:
//
//	resource "weft_keypair" "mock" {
//	  name      = "mock"
//	  file_path = "~/.ssh/id_ed25519"
//	}
//
//	resource "weft_instance" "debian" {
//	  ssh {
//	    user         = "debian"
//	    keypair_path = weft_keypair.mock.file_path
//	  }
//	}
func resourceKeypair() *schema.Resource {
	return &schema.Resource{
		Description:   "Declares an SSH keypair — mirrors the mock HCL `keypair` block. Exposes the resolved file_path and public_key for reference by weft_instance.",
		CreateContext: resourceKeypairCreate,
		ReadContext:   resourceKeypairRead,
		DeleteContext: resourceKeypairDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Keypair identifier — matches the mock HCL keypair block label (e.g. \"mock\").",
			},
			"file_path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Path to the SSH private key file — matches mock HCL keypair.file_path. Tilde (~) is expanded.",
			},
			// Computed: resolved absolute path (tilde expanded).
			"resolved_path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Absolute path to the private key after tilde expansion.",
			},
			// Computed: content of <file_path>.pub.
			"public_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Content of <file_path>.pub — ready to inject into authorized_keys or cloud-init.",
			},
		},
	}
}

func resourceKeypairCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	d.SetId(d.Get("name").(string))
	return resourceKeypairRead(ctx, d, meta)
}

func resourceKeypairRead(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	resolved := expandHome(d.Get("file_path").(string))
	if err := d.Set("resolved_path", resolved); err != nil {
		return diag.FromErr(err)
	}

	pubData, err := os.ReadFile(resolved + ".pub")
	if err != nil {
		return diag.Errorf("read public key %s.pub: %v", resolved, err)
	}
	if err := d.Set("public_key", strings.TrimSpace(string(pubData))); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceKeypairDelete(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	d.SetId("")
	return nil
}
