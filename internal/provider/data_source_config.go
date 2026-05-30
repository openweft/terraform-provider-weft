package provider

import (
	"context"
	"fmt"

	mockconfig "github.com/openweft/hclconfig"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceConfig implements the weft_config data source.
// It reads the mock HCL config directory (same format as .mock/hcl) and
// returns the fully-resolved list of VMs so users can drive weft_instance resources
// with for_each without duplicating configuration.
//
// Example usage:
//
//	data "weft_config" "mock" {
//	  config_dir = ".mock/hcl"
//	}
//
//	resource "weft_instance" "vms" {
//	  for_each = { for vm in data.weft_config.mock.vms : vm.name => vm }
//	  name     = each.key
//	  cpu      = each.value.cpu
//	  mem      = each.value.mem
//	  disk {
//	    from = each.value.image
//	    size = each.value.disk_size
//	  }
//	  ssh {
//	    user         = each.value.ssh_user
//	    keypair_path = each.value.keypair_path
//	  }
//	}
func dataSourceConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceConfigRead,
		Schema: map[string]*schema.Schema{
			"config_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     ".mock/hcl",
				Description: "Path to the mock HCL config directory (or single file).",
			},
			"vms": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of VM definitions resolved from the config.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Fully-qualified VM name (e.g. mock-<id>-debian-1).",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Number of vCPUs.",
						},
						"mem": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Memory in GiB.",
						},
						"disk_gb": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Boot disk size in GiB.",
						},
						"disk_size": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Boot disk size as string (e.g. \"20Gi\") — suitable for the weft_vm disk.size attribute.",
						},
						"image": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Resolved image URL for the boot disk.",
						},
						"ssh_user": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SSH username (from vms ssh.user).",
						},
						"keypair_path": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Resolved path to the SSH private key file (from keypair.file_path).",
						},
						"ssh_pub_key": {
							Type:        schema.TypeString,
							Computed:    true,
							Sensitive:   true,
							Description: "Content of the SSH public key file (<keypair_path>.pub).",
						},
					},
				},
			},
		},
	}
}

func dataSourceConfigRead(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
	configDir := d.Get("config_dir").(string)

	rows, err := mockconfig.ReadVMs(configDir)
	if err != nil {
		return diag.Errorf("parse mock HCL config at %q: %v", configDir, err)
	}

	vms := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		diskSize := "20Gi"
		if r.Disk > 0 {
			diskSize = fmt.Sprintf("%dGi", r.Disk)
		}
		vm := map[string]interface{}{
			"name":         r.Name,
			"cpu":          r.CPU,
			"mem":          r.Mem,
			"disk_gb":      r.Disk,
			"disk_size":    diskSize,
			"image":        r.Image,
			"ssh_user":     r.SSHUser,
			"keypair_path": r.SSHKeyPath,
			"ssh_pub_key":  r.SSHPubKey,
		}
		vms = append(vms, vm)
	}

	if err := d.Set("vms", vms); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(configDir)
	return nil
}
