package provider

import (
	"context"

	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// resourceImage implements weft_image, which triggers image caching via weft
// and mirrors the mock HCL `image` block.
//
// On Create, it calls PullImage on the weft gRPC server so the image is cached
// locally before any VM references it. If one or more `copy` blocks are
// declared, PatchImage is called immediately after pulling so that every VM
// cloned from this image inherits the patches — without needing per-instance
// copy blocks.
//
// Example — image-level GRUB patch shared by all VMs:
//
//	resource "weft_image" "debian_13" {
//	  from     = "https://cloud.debian.org/.../debian-13-genericcloud-arm64.raw"
//	  checksum = "https://cloud.debian.org/.../SHA512SUMS"
//
//	  copy {
//	    content = <<-EOF
//	      GRUB_TERMINAL_OUTPUT="console"
//	      GRUB_CMDLINE_LINUX_DEFAULT="console=tty0 console=hvc0"
//	    EOF
//	    dst     = "/etc/default/grub.d/99-console.cfg"
//	    trigger = "grub-mkconfig"
//	  }
//	}
//
//	resource "weft_instance" "debian" {
//	  disk { from = weft_image.debian_13.from }
//	}
func resourceImage() *schema.Resource {
	return &schema.Resource{
		Description:   "Declares an image URL — mirrors the mock HCL `image` block. Use weft_image.<name>.from to reference the URL in weft_instance.",
		CreateContext: resourceImageCreate,
		ReadContext:   resourceImageRead,
		DeleteContext: resourceImageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"from": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Fully-resolved image URL — matches mock HCL image.from (e.g. the result of join(...) in images.hcl).",
			},
			"checksum": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Checksum file URL — matches mock HCL image.checksum.",
			},
			// patch block applies file add/delete operations once to the cached image
			// so every VM cloned from it inherits the changes automatically.
			"patch": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "Patch operations applied once to the cached image (shared by all VMs cloned from it).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"add": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Files to write into the disk image.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Literal file content to write into the disk image.",
									},
									"dst": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Absolute destination path inside the disk image.",
									},
									"trigger": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "",
										Description: "Post-add trigger. Supported: \"grub-mkconfig\".",
									},
								},
							},
						},
						"del": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Files to remove from the disk image.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"dst": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Absolute path inside the disk image to delete.",
									},
								},
							},
						},
						"mod": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "In-place regex substitutions to apply inside files of the disk image.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"dst": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Absolute path of the file to modify inside the disk image.",
									},
									"old": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "RE2 regular expression to match (compatible with most PCRE patterns). All non-overlapping matches are replaced.",
									},
									"new": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Replacement text. May reference capture groups: $1, ${name}.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceImageCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*weftClient)
	fromURL := d.Get("from").(string)
	checksum := d.Get("checksum").(string)
	_, err := client.vms.PullImage(ctx, &weftv1.PullImageRequest{Url: fromURL, Checksum: checksum})
	if err != nil {
		return diag.Errorf("PullImage failed for %s: %v", fromURL, err)
	}

	// Apply image-level patches if a patch block was declared.
	patchList := d.Get("patch").([]interface{})
	if len(patchList) > 0 {
		patchMap := patchList[0].(map[string]interface{})
		var fileOps []*weftv1.DiskFileOp
		for _, item := range patchMap["add"].([]interface{}) {
			m := item.(map[string]interface{})
			fileOps = append(fileOps, &weftv1.DiskFileOp{
				Content: m["content"].(string),
				Dst:     m["dst"].(string),
				Trigger: m["trigger"].(string),
			})
		}
		var deleteOps []*weftv1.DiskDeleteOp
		for _, item := range patchMap["del"].([]interface{}) {
			m := item.(map[string]interface{})
			deleteOps = append(deleteOps, &weftv1.DiskDeleteOp{Dst: m["dst"].(string)})
		}
		var modOps []*weftv1.DiskModOp
		for _, item := range patchMap["mod"].([]interface{}) {
			m := item.(map[string]interface{})
			modOps = append(modOps, &weftv1.DiskModOp{
				Dst: m["dst"].(string),
				Old: m["old"].(string),
				New: m["new"].(string),
			})
		}
		if len(fileOps) > 0 || len(deleteOps) > 0 || len(modOps) > 0 {
			if _, err := client.vms.PatchImage(ctx, &weftv1.PatchImageRequest{
				Url:       fromURL,
				FileOps:   fileOps,
				DeleteOps: deleteOps,
				ModOps:    modOps,
			}); err != nil {
				return diag.Errorf("PatchImage failed for %s: %v", fromURL, err)
			}
		}
	}

	d.SetId(fromURL)
	return nil
}

func resourceImageRead(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	return nil
}

func resourceImageDelete(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	d.SetId("")
	return nil
}
