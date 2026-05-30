// image_patch_resource.go — `weft_image_patch` ported from sdk/v2 to
// terraform-plugin-framework. Applies one-time patch operations to cached
// images: either an explicit `images` list, or all cached images when omitted.

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
)

func NewImagePatchResource() resource.Resource { return &imagePatchResource{} }

type imagePatchResource struct {
	client *weftClient
}

type imagePatchModel struct {
	ID     types.String     `tfsdk:"id"`
	Images types.List       `tfsdk:"images"`
	Patch  *imagePatchBlock `tfsdk:"patch"`
}

func (r *imagePatchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_patch"
}

func (r *imagePatchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Apply one-time patch operations to cached images (file add/del/mod).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Synthetic identifier of the patch run.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"images": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Description:   "Optional list of image URLs to patch. If omitted, patches are applied to all cached images.",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
		},
		Blocks: map[string]schema.Block{
			"patch": schema.SingleNestedBlock{
				Description: "Patch operations to apply (add/del/mod).",
				Blocks: map[string]schema.Block{
					"add": schema.ListNestedBlock{
						Description: "Files to write into the disk image.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"content": schema.StringAttribute{Required: true},
								"dst":     schema.StringAttribute{Required: true},
								"trigger": schema.StringAttribute{Optional: true, Computed: true},
							},
						},
					},
					"del": schema.ListNestedBlock{
						Description: "Files to remove from the disk image.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"dst": schema.StringAttribute{Required: true},
							},
						},
					},
					"mod": schema.ListNestedBlock{
						Description: "In-place regex substitutions to apply inside files of the disk image.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"dst": schema.StringAttribute{Required: true},
								"old": schema.StringAttribute{Required: true},
								"new": schema.StringAttribute{Required: true},
							},
						},
					},
				},
			},
		},
	}
}

func (r *imagePatchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *imagePatchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan imagePatchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Patch == nil {
		resp.Diagnostics.AddError("patch block required", "patch block required")
		return
	}

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

	plan.ID = types.StringValue(fmt.Sprintf("imagepatch-%d", time.Now().UnixNano()))

	if len(fileOps) == 0 && len(deleteOps) == 0 && len(modOps) == 0 {
		// Nothing to do — back-fill computed defaults and write state.
		r.backfillTriggerDefaults(&plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// If images are explicitly provided, target those. Otherwise apply to all
	// cached images returned by ListImages.
	var imgs []string
	if !plan.Images.IsNull() && !plan.Images.IsUnknown() {
		resp.Diagnostics.Append(plan.Images.ElementsAs(ctx, &imgs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if len(imgs) > 0 {
		for _, url := range imgs {
			if _, err := r.client.vms.PatchImage(ctx, &weftv1.PatchImageRequest{Url: url, FileOps: fileOps, DeleteOps: deleteOps, ModOps: modOps}); err != nil {
				resp.Diagnostics.AddError("PatchImage failed", fmt.Sprintf("for %s: %v", url, err))
				return
			}
		}
	} else {
		listResp, err := r.client.vms.ListImages(ctx, &weftv1.ListImagesRequest{})
		if err != nil {
			resp.Diagnostics.AddError("ListImages failed", err.Error())
			return
		}
		for _, info := range listResp.Images {
			if _, err := r.client.vms.PatchImage(ctx, &weftv1.PatchImageRequest{Url: info.Url, FileOps: fileOps, DeleteOps: deleteOps, ModOps: modOps}); err != nil {
				resp.Diagnostics.AddError("PatchImage failed", fmt.Sprintf("for %s: %v", info.Url, err))
				return
			}
		}
	}

	r.backfillTriggerDefaults(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imagePatchResource) backfillTriggerDefaults(plan *imagePatchModel) {
	if plan.Patch == nil {
		return
	}
	for i := range plan.Patch.Add {
		if plan.Patch.Add[i].Trigger.IsNull() || plan.Patch.Add[i].Trigger.IsUnknown() {
			plan.Patch.Add[i].Trigger = types.StringValue("")
		}
	}
}

func (r *imagePatchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state imagePatchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *imagePatchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan imagePatchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imagePatchResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
