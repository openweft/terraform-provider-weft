// config_data_source.go — `data weft_config` ported from sdk/v2 to
// terraform-plugin-framework. Reads a mock HCL config dir and returns the
// fully-resolved list of VMs so consumers can drive weft_instance with
// for_each.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mockconfig "github.com/openweft/weft-hcl"
)

func NewConfigDataSource() datasource.DataSource { return &configDataSource{} }

type configDataSource struct{}

type configDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	ConfigDir types.String `tfsdk:"config_dir"`
	VMs       types.List   `tfsdk:"vms"`
}

// vmObjectAttrTypes is the schema of a single element in the `vms` list.
// Centralised so Schema() and Read() agree on the shape.
var vmObjectAttrTypes = map[string]attr.Type{
	"name":         types.StringType,
	"cpu":          types.Int64Type,
	"mem":          types.Int64Type,
	"disk_gb":      types.Int64Type,
	"disk_size":    types.StringType,
	"image":        types.StringType,
	"ssh_user":     types.StringType,
	"keypair_path": types.StringType,
	"ssh_pub_key":  types.StringType,
}

func (d *configDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (d *configDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the mock HCL config directory and returns the resolved list of VMs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic identifier (mirrors `config_dir`).",
			},
			"config_dir": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Path to the mock HCL config directory (default \".mock/hcl\").",
			},
			"vms": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of VM definitions resolved from the config.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":         schema.StringAttribute{Computed: true, Description: "Fully-qualified VM name."},
						"cpu":          schema.Int64Attribute{Computed: true, Description: "Number of vCPUs."},
						"mem":          schema.Int64Attribute{Computed: true, Description: "Memory in GiB."},
						"disk_gb":      schema.Int64Attribute{Computed: true, Description: "Boot disk size in GiB."},
						"disk_size":    schema.StringAttribute{Computed: true, Description: "Boot disk size as string (e.g. \"20Gi\")."},
						"image":        schema.StringAttribute{Computed: true, Description: "Resolved image URL."},
						"ssh_user":     schema.StringAttribute{Computed: true, Description: "SSH username."},
						"keypair_path": schema.StringAttribute{Computed: true, Description: "Resolved path to the SSH private key file."},
						"ssh_pub_key":  schema.StringAttribute{Computed: true, Sensitive: true, Description: "Content of the SSH public key file."},
					},
				},
			},
		},
	}
}

func (d *configDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
}

func (d *configDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg configDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configDir := ".mock/hcl"
	if !cfg.ConfigDir.IsNull() && !cfg.ConfigDir.IsUnknown() && cfg.ConfigDir.ValueString() != "" {
		configDir = cfg.ConfigDir.ValueString()
	}

	rows, err := mockconfig.ReadVMs(configDir)
	if err != nil {
		resp.Diagnostics.AddError("parse mock HCL config", fmt.Sprintf("at %q: %v", configDir, err))
		return
	}

	elemType := types.ObjectType{AttrTypes: vmObjectAttrTypes}
	elems := make([]attr.Value, 0, len(rows))
	for _, r := range rows {
		diskSize := "20Gi"
		if r.Disk > 0 {
			diskSize = fmt.Sprintf("%dGi", r.Disk)
		}
		obj, objDiags := types.ObjectValue(vmObjectAttrTypes, map[string]attr.Value{
			"name":         types.StringValue(r.Name),
			"cpu":          types.Int64Value(int64(r.CPU)),
			"mem":          types.Int64Value(int64(r.Mem)),
			"disk_gb":      types.Int64Value(int64(r.Disk)),
			"disk_size":    types.StringValue(diskSize),
			"image":        types.StringValue(r.Image),
			"ssh_user":     types.StringValue(r.SSHUser),
			"keypair_path": types.StringValue(r.SSHKeyPath),
			"ssh_pub_key":  types.StringValue(r.SSHPubKey),
		})
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems = append(elems, obj)
	}
	list, listDiags := types.ListValue(elemType, elems)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg.ID = types.StringValue(configDir)
	cfg.ConfigDir = types.StringValue(configDir)
	cfg.VMs = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
