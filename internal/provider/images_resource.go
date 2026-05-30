// images_resource.go — `weft_images` ported from sdk/v2 to
// terraform-plugin-framework. Bulk-pulls all images referenced in a mock HCL
// config directory via weft's PullImages RPC.

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mockconfig "github.com/openweft/hclconfig"
	weftv1 "github.com/openweft/weft-proto"
)

func NewImagesResource() resource.Resource { return &imagesResource{} }

type imagesResource struct {
	client *weftClient
}

type imagesModel struct {
	ID        types.String `tfsdk:"id"`
	ConfigDir types.String `tfsdk:"config_dir"`
	Parallel  types.Int64  `tfsdk:"parallel"`
	Pulled    types.List   `tfsdk:"pulled"`
}

func (r *imagesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func (r *imagesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Pulls all images referenced in the mock HCL config via weft PullImages.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Synthetic identifier (\"<config_dir>|<parallel>\").",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_dir": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Path to the mock HCL config directory (default \".mock/hcl\").",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"parallel": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Maximum number of images to pull in parallel (default 4).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"pulled": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of image references that were pulled.",
			},
		},
	}
}

func (r *imagesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*weftClient)
	if !ok {
		resp.Diagnostics.AddError("provider data type", fmt.Sprintf("expected *weftClient, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *imagesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan imagesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configDir := ".mock/hcl"
	if !plan.ConfigDir.IsNull() && !plan.ConfigDir.IsUnknown() && plan.ConfigDir.ValueString() != "" {
		configDir = plan.ConfigDir.ValueString()
	}
	parallel := int64(4)
	if !plan.Parallel.IsNull() && !plan.Parallel.IsUnknown() {
		parallel = plan.Parallel.ValueInt64()
	}

	if _, err := r.client.vms.PullImages(ctx, &weftv1.PullImagesRequest{
		ConfigDir: configDir,
		Parallel:  int32(parallel),
	}); err != nil {
		resp.Diagnostics.AddError("PullImages failed", err.Error())
		return
	}

	// Enumerate images referenced in the config for tracking. Errors here
	// are non-fatal — match the sdk/v2 behaviour.
	rows, _ := mockconfig.ReadVMs(configDir)
	seen := map[string]struct{}{}
	pulled := make([]string, 0)
	for _, row := range rows {
		if row.Image != "" {
			if _, ok := seen[row.Image]; !ok {
				seen[row.Image] = struct{}{}
				pulled = append(pulled, row.Image)
			}
		}
	}
	pulledList, diags := types.ListValueFrom(ctx, types.StringType, pulled)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s|%d", configDir, parallel))
	plan.ConfigDir = types.StringValue(configDir)
	plan.Parallel = types.Int64Value(parallel)
	plan.Pulled = pulledList
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imagesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state imagesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Restore config_dir / parallel from the stored ID so Terraform can
	// detect ForceNew changes — mirrors the sdk/v2 Read path.
	id := state.ID.ValueString()
	if id != "" {
		parts := strings.SplitN(id, "|", 2)
		if len(parts) == 2 {
			state.ConfigDir = types.StringValue(parts[0])
			if n, err := strconv.Atoi(parts[1]); err == nil {
				state.Parallel = types.Int64Value(int64(n))
			}
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *imagesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan imagesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imagesResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *imagesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
