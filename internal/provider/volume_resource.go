// volume_resource.go — `weft_volume` closes the Volumes-service gap from
// GAPS.md. Proto exposes CreateVolume / RenameVolume / ResizeVolume /
// DeleteVolume; attachment (AttachVolume/DetachVolume) is a separate
// concern that will land as `weft_volume_attachment`.
//
//   Terraform op       →  weft RPC
//   ────────────────────────────────────────────────────────────────────
//   Create             →  CreateVolume
//   Read               →  ListVolumes(project) + uuid filter
//   Update name        →  RenameVolume
//   Update size_gib    →  ResizeVolume   (grow-only; shrink rejected server-side)
//   Delete             →  DeleteVolume
//   Import             →  "<project>/<uuid>"
//
// `project` and `format` are RequiresReplace (a different bucket /
// on-disk format is a different volume). `name` and `size_gib` are
// mutable.

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
)

func NewVolumeResource() resource.Resource { return &volumeResource{} }

type volumeResource struct {
	client *weftClient
}

type volumeModel struct {
	ID             types.String `tfsdk:"id"`
	UUID           types.String `tfsdk:"uuid"`
	Project        types.String `tfsdk:"project"`
	ProjectUUID    types.String `tfsdk:"project_uuid"`
	Name           types.String `tfsdk:"name"`
	SizeGiB        types.Int64  `tfsdk:"size_gib"`
	Format         types.String `tfsdk:"format"`
	AttachedToUUID types.String `tfsdk:"attached_to_uuid"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *volumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Persistent block volume within a project. Mirrors CreateVolume / RenameVolume / ResizeVolume / DeleteVolume.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Mirrors `uuid`; satisfies the Terraform ID convention.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "Server-minted volume UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				Required:      true,
				Description:   "Owning project (display name or UUID). Immutable.",
				PlanModifiers: requiresReplace,
			},
			"project_uuid": schema.StringAttribute{
				Computed:    true,
				Description: "Resolved project UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Volume name (display label). Mutable via RenameVolume.",
			},
			"size_gib": schema.Int64Attribute{
				Required:    true,
				Description: "Volume size in GiB. Mutable via ResizeVolume (grow-only — shrink is rejected server-side).",
			},
			"format": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   `On-disk format: "raw" | "qcow2". Defaults to "raw". Immutable.`,
				PlanModifiers: requiresReplace,
			},
			"attached_to_uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the VM the volume is attached to; empty when detached.",
			},
			"created_at": schema.StringAttribute{Computed: true, Description: "Unix nanosecond timestamp of creation."},
		},
	}
}

func (r *volumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.vms.CreateVolume(ctx, &weftv1.CreateVolumeRequest{
		Project: plan.Project.ValueString(),
		Name:    plan.Name.ValueString(),
		SizeGib: plan.SizeGiB.ValueInt64(),
		Format:  plan.Format.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("CreateVolume failed", err.Error())
		return
	}
	plan.applyVolumeInfo(out.GetVolume())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	info, found, err := findVolume(ctx, r.client, state.Project.ValueString(), state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ListVolumes failed", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.applyVolumeInfo(info)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state volumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	var latest *weftv1.VolumeInfo

	if !plan.Name.Equal(state.Name) {
		out, err := r.client.vms.RenameVolume(ctx, &weftv1.RenameVolumeRequest{Uuid: uuid, NewName: plan.Name.ValueString()})
		if err != nil {
			resp.Diagnostics.AddError("RenameVolume failed", err.Error())
			return
		}
		latest = out.GetVolume()
	}
	if !plan.SizeGiB.Equal(state.SizeGiB) {
		out, err := r.client.vms.ResizeVolume(ctx, &weftv1.ResizeVolumeRequest{Uuid: uuid, NewSizeGib: plan.SizeGiB.ValueInt64()})
		if err != nil {
			resp.Diagnostics.AddError("ResizeVolume failed", err.Error())
			return
		}
		latest = out.GetVolume()
	}

	if latest == nil {
		info, found, err := findVolume(ctx, r.client, state.Project.ValueString(), uuid)
		if err != nil {
			resp.Diagnostics.AddError("ListVolumes failed", err.Error())
			return
		}
		if !found {
			resp.State.RemoveResource(ctx)
			return
		}
		latest = info
	}
	plan.applyVolumeInfo(latest)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.vms.DeleteVolume(ctx, &weftv1.DeleteVolumeRequest{Uuid: state.UUID.ValueString()}); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("DeleteVolume failed", err.Error())
	}
}

func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("invalid import id", `expected "<project>/<uuid>"`)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
}

func findVolume(ctx context.Context, c *weftClient, project, uuid string) (*weftv1.VolumeInfo, bool, error) {
	var token string
	for {
		out, err := c.vms.ListVolumes(ctx, &weftv1.ListVolumesRequest{Project: project, PageToken: token})
		if err != nil {
			return nil, false, err
		}
		for _, v := range out.GetVolumes() {
			if v.GetUuid() == uuid {
				return v, true, nil
			}
		}
		token = out.GetNextPageToken()
		if token == "" {
			return nil, false, nil
		}
	}
}

func (m *volumeModel) applyVolumeInfo(v *weftv1.VolumeInfo) {
	if v == nil {
		return
	}
	m.ID = types.StringValue(v.GetUuid())
	m.UUID = types.StringValue(v.GetUuid())
	m.ProjectUUID = types.StringValue(v.GetProjectUuid())
	m.Name = types.StringValue(v.GetName())
	m.SizeGiB = types.Int64Value(v.GetSizeGib())
	m.Format = types.StringValue(v.GetFormat())
	m.AttachedToUUID = types.StringValue(v.GetAttachedToUuid())
	m.CreatedAt = types.StringValue(strconv.FormatInt(v.GetCreatedAtUnixNs(), 10))
}
