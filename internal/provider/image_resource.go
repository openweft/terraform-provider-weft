// image_resource.go — `weft_image` ported from sdk/v2 to
// terraform-plugin-framework. Triggers PullImage on Create and, if a `patch`
// block is declared, PatchImage to apply image-level patches once so every
// VM cloned from this image inherits them.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
)

func NewImageResource() resource.Resource { return &imageResource{} }

type imageResource struct {
	client *weftClient
}

type imageModel struct {
	ID       types.String     `tfsdk:"id"`
	From     types.String     `tfsdk:"from"`
	Checksum types.String     `tfsdk:"checksum"`
	Patch    *imagePatchBlock `tfsdk:"patch"`
}

type imagePatchBlock struct {
	Add []addOp `tfsdk:"add"`
	Del []delOp `tfsdk:"del"`
	Mod []modOp `tfsdk:"mod"`
}

func (r *imageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (r *imageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplaceStr := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		Description: "Declares an image URL — mirrors the mock HCL `image` block. Pulls the image via weft's PullImage RPC and optionally applies image-level patches.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Image ID (mirrors `from`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"from": schema.StringAttribute{
				Required:      true,
				Description:   "Fully-resolved image URL — matches mock HCL image.from.",
				PlanModifiers: requiresReplaceStr,
			},
			"checksum": schema.StringAttribute{
				Optional:      true,
				Description:   "Checksum file URL — matches mock HCL image.checksum.",
				PlanModifiers: requiresReplaceStr,
			},
		},
		Blocks: map[string]schema.Block{
			"patch": schema.SingleNestedBlock{
				Description: "Patch operations applied once to the cached image (shared by all VMs cloned from it).",
				Blocks: map[string]schema.Block{
					"add": schema.ListNestedBlock{
						Description: "Files to write into the disk image.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"content": schema.StringAttribute{Required: true, Description: "Literal file content."},
								"dst":     schema.StringAttribute{Required: true, Description: "Absolute destination path inside the disk image."},
								"trigger": schema.StringAttribute{Optional: true, Computed: true, Description: `Post-add trigger. Supported: "grub-mkconfig".`},
							},
						},
					},
					"del": schema.ListNestedBlock{
						Description: "Files to remove from the disk image.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"dst": schema.StringAttribute{Required: true, Description: "Absolute path to delete."},
							},
						},
					},
					"mod": schema.ListNestedBlock{
						Description: "RE2-regex substitutions applied in-place.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"dst": schema.StringAttribute{Required: true, Description: "Absolute path of the file to modify."},
								"old": schema.StringAttribute{Required: true, Description: "RE2 pattern to match."},
								"new": schema.StringAttribute{Required: true, Description: "Replacement text. May reference $1, ${name}."},
							},
						},
					},
				},
			},
		},
	}
}

func (r *imageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *imageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan imageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	from := plan.From.ValueString()
	checksum := ""
	if !plan.Checksum.IsNull() && !plan.Checksum.IsUnknown() {
		checksum = plan.Checksum.ValueString()
	}
	if _, err := r.client.vms.PullImage(ctx, &weftv1.PullImageRequest{Url: from, Checksum: checksum}); err != nil {
		resp.Diagnostics.AddError("PullImage failed", fmt.Sprintf("for %s: %v", from, err))
		return
	}

	if plan.Patch != nil {
		var fileOps []*weftv1.DiskFileOp
		for _, a := range plan.Patch.Add {
			fileOps = append(fileOps, &weftv1.DiskFileOp{
				Content: a.Content.ValueString(),
				Dst:     a.Dst.ValueString(),
				Trigger: a.Trigger.ValueString(),
			})
		}
		var deleteOps []*weftv1.DiskDeleteOp
		for _, d := range plan.Patch.Del {
			deleteOps = append(deleteOps, &weftv1.DiskDeleteOp{Dst: d.Dst.ValueString()})
		}
		var modOps []*weftv1.DiskModOp
		for _, m := range plan.Patch.Mod {
			modOps = append(modOps, &weftv1.DiskModOp{
				Dst: m.Dst.ValueString(),
				Old: m.Old.ValueString(),
				New: m.New.ValueString(),
			})
		}
		if len(fileOps) > 0 || len(deleteOps) > 0 || len(modOps) > 0 {
			if _, err := r.client.vms.PatchImage(ctx, &weftv1.PatchImageRequest{
				Url:       from,
				FileOps:   fileOps,
				DeleteOps: deleteOps,
				ModOps:    modOps,
			}); err != nil {
				resp.Diagnostics.AddError("PatchImage failed", fmt.Sprintf("for %s: %v", from, err))
				return
			}
		}
		// Back-fill computed trigger defaults so state matches plan.
		for i := range plan.Patch.Add {
			if plan.Patch.Add[i].Trigger.IsNull() || plan.Patch.Add[i].Trigger.IsUnknown() {
				plan.Patch.Add[i].Trigger = types.StringValue("")
			}
		}
	}

	plan.ID = types.StringValue(from)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state imageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *imageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan imageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imageResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *imageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("from"), req.ID)...)
}
