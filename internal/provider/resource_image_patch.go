package provider

import (
	"context"
	"fmt"
	"time"

	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceImagePatch implements a top-level weft_image_patch resource which
// applies one-time patch operations to cached images. It can target a list of
// explicit image URLs via `images`, or when omitted it applies to all cached
// images returned by weft's ListImages RPC.
func resourceImagePatch() *schema.Resource {
	return &schema.Resource{
		Description:   "Apply one-time patch operations to cached images (file add/del/mod).",
		CreateContext: resourceImagePatchCreate,
		ReadContext:   resourceImagePatchRead,
		DeleteContext: resourceImagePatchDelete,
		Schema: map[string]*schema.Schema{
			"images": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "Optional list of image URLs to patch. If omitted, patches are applied to all cached images.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"patch": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "Patch operations to apply (add/del/mod).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"add": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Files to write into the disk image.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content": {Type: schema.TypeString, Required: true},
									"dst":     {Type: schema.TypeString, Required: true},
									"trigger": {Type: schema.TypeString, Optional: true, Default: ""},
								},
							},
						},
						"del": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Files to remove from the disk image.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"dst": {Type: schema.TypeString, Required: true},
								},
							},
						},
						"mod": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "In-place regex substitutions to apply inside files of the disk image.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"dst": {Type: schema.TypeString, Required: true},
									"old": {Type: schema.TypeString, Required: true},
									"new": {Type: schema.TypeString, Required: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceImagePatchCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*weftClient)

	patchList := d.Get("patch").([]interface{})
	if len(patchList) == 0 {
		return diag.Errorf("patch block required")
	}
	patchMap := patchList[0].(map[string]interface{})

	var fileOps []*weftv1.DiskFileOp
	if v, ok := patchMap["add"].([]interface{}); ok {
		for _, item := range v {
			m := item.(map[string]interface{})
			fileOps = append(fileOps, &weftv1.DiskFileOp{
				Content: m["content"].(string),
				Dst:     m["dst"].(string),
				Trigger: m["trigger"].(string),
			})
		}
	}

	var deleteOps []*weftv1.DiskDeleteOp
	if v, ok := patchMap["del"].([]interface{}); ok {
		for _, item := range v {
			m := item.(map[string]interface{})
			deleteOps = append(deleteOps, &weftv1.DiskDeleteOp{Dst: m["dst"].(string)})
		}
	}

	var modOps []*weftv1.DiskModOp
	if v, ok := patchMap["mod"].([]interface{}); ok {
		for _, item := range v {
			m := item.(map[string]interface{})
			modOps = append(modOps, &weftv1.DiskModOp{
				Dst: m["dst"].(string),
				Old: m["old"].(string),
				New: m["new"].(string),
			})
		}
	}

	if len(fileOps) == 0 && len(deleteOps) == 0 && len(modOps) == 0 {
		// Nothing to do.
		d.SetId(fmt.Sprintf("imagepatch-%d", time.Now().UnixNano()))
		return nil
	}

	// If images are explicitly provided, target those. Otherwise apply to
	// all cached images returned by ListImages.
	imgs := d.Get("images").([]interface{})
	if len(imgs) > 0 {
		for _, it := range imgs {
			url := it.(string)
			if _, err := client.vms.PatchImage(ctx, &weftv1.PatchImageRequest{Url: url, FileOps: fileOps, DeleteOps: deleteOps, ModOps: modOps}); err != nil {
				return diag.Errorf("PatchImage failed for %s: %v", url, err)
			}
		}
	} else {
		resp, err := client.vms.ListImages(ctx, &weftv1.ListImagesRequest{})
		if err != nil {
			return diag.Errorf("ListImages failed: %v", err)
		}
		for _, info := range resp.Images {
			if _, err := client.vms.PatchImage(ctx, &weftv1.PatchImageRequest{Url: info.Url, FileOps: fileOps, DeleteOps: deleteOps, ModOps: modOps}); err != nil {
				return diag.Errorf("PatchImage failed for %s: %v", info.Url, err)
			}
		}
	}

	d.SetId(fmt.Sprintf("imagepatch-%d", time.Now().UnixNano()))
	return nil
}

func resourceImagePatchRead(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	return nil
}

func resourceImagePatchDelete(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	d.SetId("")
	return nil
}
