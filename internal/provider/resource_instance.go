package provider

import (
	"context"

	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		Description:   "Provisions and manages a VM via weft. Schema mirrors the mock HCL vms block.",
		CreateContext: resourceInstanceCreate,
		ReadContext:   resourceInstanceRead,
		DeleteContext: resourceInstanceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Unique VM name.",
			},
			"cpu": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     2,
				ForceNew:    true,
				Description: "Number of vCPUs — matches mock HCL `cpu`.",
			},
			"mem": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     2,
				ForceNew:    true,
				Description: "Memory in GiB — matches mock HCL `mem`.",
			},
			// disk block mirrors `disk { from = ..., size = "20Gi" }` in mock HCL.
			"disk": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "Boot disk configuration (mirrors mock HCL disk block).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Resolved image URL for the boot disk. Use the weft_config data source to get this value from image.xxx.from.",
						},
						"size": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "20Gi",
							Description: "Disk size with unit suffix (e.g. \"20Gi\", \"100Gi\") — matches mock HCL disk.size.",
						},
						// patch block mirrors file changes to apply before first boot.
						"patch": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "File patch operations applied to the disk image before first boot.",
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
				},
			},
			// ssh block mirrors `ssh { user = "...", keypair = keypair.xxx }` in mock HCL.
			"ssh": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "SSH configuration (mirrors mock HCL ssh block).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "ubuntu",
							Description: "SSH username — matches mock HCL ssh.user.",
						},
						"keypair_path": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Path to the SSH private key file; the public key is read from <keypair_path>.pub — matches mock HCL keypair.file_path.",
						},
					},
				},
			},
			// Computed outputs.
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IP address of the first replica.",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "State of the first replica.",
			},
		},
	}
}

func resourceInstanceCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*weftClient)

	// Extract disk block.
	diskList := d.Get("disk").([]interface{})
	if len(diskList) == 0 {
		return diag.Errorf("disk block is required")
	}
	diskMap := diskList[0].(map[string]interface{})
	image := diskMap["from"].(string)
	sizeStr := diskMap["size"].(string)
	diskGiB, ok := parseSizeGiB(sizeStr)
	if !ok {
		return diag.Errorf("invalid disk size %q: use a unit suffix like 20Gi", sizeStr)
	}

	// Extract ssh block and read the public key from <keypair_path>.pub.
	sshPubKey := ""
	if sshList := d.Get("ssh").([]interface{}); len(sshList) > 0 {
		sshMap := sshList[0].(map[string]interface{})
		kp := expandHome(sshMap["keypair_path"].(string))
		pubKeyData, err := os.ReadFile(kp + ".pub")
		if err != nil {
			return diag.Errorf("read SSH public key %s.pub: %v", kp, err)
		}
		sshPubKey = strings.TrimSpace(string(pubKeyData))
	}

	vmName := d.Get("name").(string)
	// Extract patch ops from disk block.
	var fileOps []*weftv1.DiskFileOp
	var deleteOps []*weftv1.DiskDeleteOp
	var modOps []*weftv1.DiskModOp
	if patchList, ok := diskMap["patch"].([]interface{}); ok && len(patchList) > 0 {
		patchMap := patchList[0].(map[string]interface{})
		for _, item := range patchMap["add"].([]interface{}) {
			m := item.(map[string]interface{})
			fileOps = append(fileOps, &weftv1.DiskFileOp{
				Content: m["content"].(string),
				Dst:     m["dst"].(string),
				Trigger: m["trigger"].(string),
			})
		}
		for _, item := range patchMap["del"].([]interface{}) {
			m := item.(map[string]interface{})
			deleteOps = append(deleteOps, &weftv1.DiskDeleteOp{Dst: m["dst"].(string)})
		}
		for _, item := range patchMap["mod"].([]interface{}) {
			m := item.(map[string]interface{})
			modOps = append(modOps, &weftv1.DiskModOp{
				Dst: m["dst"].(string),
				Old: m["old"].(string),
				New: m["new"].(string),
			})
		}
	}

	req := &weftv1.ProvisionVMRequest{
		Name:      vmName,
		Image:     image,
		Cpu:       uint32(d.Get("cpu").(int)),
		MemMb:     uint64(d.Get("mem").(int)) * 1024,
		DiskGb:    uint64(diskGiB),
		SshPubKey: sshPubKey,
		FileOps:   fileOps,
		DeleteOps: deleteOps,
		ModOps:    modOps,
	}
	if _, err := client.vms.ProvisionVM(ctx, req); err != nil {
		return diag.Errorf("ProvisionVM failed for %s: %v", vmName, err)
	}

	d.SetId(vmName)
	return resourceInstanceRead(ctx, d, meta)
}

func resourceInstanceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*weftClient)

	vmName := d.Get("name").(string)
	resp, err := client.vms.VMStatus(ctx, &weftv1.VMStatusRequest{Name: vmName})
	if err != nil {
		// Only mark as gone for explicit not-found; keep state for transient errors.
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
		}
		return nil
	}
	if err := d.Set("ip", resp.GetVm().GetIp()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("state", resp.GetVm().GetState().String()); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceInstanceDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*weftClient)

	vmName := d.Get("name").(string)
	if _, err := client.vms.DeprovisionVM(ctx, &weftv1.DeprovisionVMRequest{Name: vmName}); err != nil {
		return diag.Errorf("DeprovisionVM failed for %s: %v", vmName, err)
	}

	d.SetId("")
	return nil
}

// parseSizeGiB parses a size string with unit suffix (e.g. "20Gi", "100G", "2Ti")
// into a GiB integer. Mirrors the logic used by the mock HCL parser.
// Supported units: Mi/M (÷1024), Gi/G (as-is), Ti/T (×1024).
func parseSizeGiB(raw string) (int, bool) {
	s := strings.Trim(raw, "\"")
	m := regexp.MustCompile(`^([0-9]+)(Mi?|Gi?|Ti?)$`).FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, err := strconv.Atoi(m[1])
	if err != nil || v <= 0 {
		return 0, false
	}
	switch strings.ToUpper(m[2])[0] {
	case 'M':
		gib := v / 1024
		if gib <= 0 {
			return 0, false
		}
		return gib, true
	case 'G':
		return v, true
	case 'T':
		return v * 1024, true
	}
	return 0, false
}

// expandHome expands a leading tilde to the user home directory.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
