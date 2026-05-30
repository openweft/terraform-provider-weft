package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	weftv1 "github.com/openweft/weft-proto"
	mockconfig "github.com/openweft/hclconfig"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceImages implements weft_images, a resource that pulls all images
// referenced in the mock HCL config directory via weft's PullImages RPC.
// Since image pulling is idempotent, this resource has no meaningful Read or
// Delete; Create is re-triggered whenever config_dir or parallel change.
//
// Example:
//
//	resource "weft_images" "all" {
//	  config_dir = ".mock/hcl"
//	  parallel   = 4
//	}
func resourceImages() *schema.Resource {
	return &schema.Resource{
		Description:   "Pulls all images referenced in the mock HCL config via weft PullImages.",
		CreateContext: resourceImagesCreate,
		ReadContext:   resourceImagesRead,
		DeleteContext: resourceImagesDelete,
		Schema: map[string]*schema.Schema{
			"config_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     ".mock/hcl",
				ForceNew:    true,
				Description: "Path to the mock HCL config directory (same as mock --config-dir).",
			},
			"parallel": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     4,
				ForceNew:    true,
				Description: "Maximum number of images to pull in parallel.",
			},
			"pulled": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of image references that were pulled.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceImagesCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*weftClient)

	req := &weftv1.PullImagesRequest{
		ConfigDir: d.Get("config_dir").(string),
		Parallel:  int32(d.Get("parallel").(int)),
	}

	_, err := client.vms.PullImages(ctx, req)
	if err != nil {
		return diag.Errorf("PullImages failed: %v", err)
	}

	// Enumerate images referenced in the config for tracking.
	rows, _ := mockconfig.ReadVMs(req.ConfigDir)
	seen := map[string]struct{}{}
	pulled := make([]interface{}, 0)
	for _, r := range rows {
		if r.Image != "" {
			if _, ok := seen[r.Image]; !ok {
				seen[r.Image] = struct{}{}
				pulled = append(pulled, r.Image)
			}
		}
	}
	if err := d.Set("pulled", pulled); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s|%d", req.ConfigDir, req.Parallel))
	return nil
}

func resourceImagesRead(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	// Image pulls are side effects; nothing to reconcile.
	// Restore config_dir and parallel from the stored ID so Terraform can
	// detect ForceNew changes.
	id := d.Id()
	if id == "" {
		return nil
	}
	parts := strings.SplitN(id, "|", 2)
	if len(parts) == 2 {
		_ = d.Set("config_dir", parts[0])
		if n, err := strconv.Atoi(parts[1]); err == nil {
			_ = d.Set("parallel", n)
		}
	}
	return nil
}

func resourceImagesDelete(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	// Images live in weft's local OCI store; deletion is handled by weft_clean.
	// Here we simply remove the resource from state.
	d.SetId("")
	return nil
}
