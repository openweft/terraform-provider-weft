// volume_snapshot_resource.go — `weft_volume_snapshot` wraps the
// CreateVolumeSnapshot / ListVolumeSnapshots / DeleteVolumeSnapshot
// RPCs added in weft-proto v0.2.0.
//
// Lifecycle:
//
//   Terraform op  →  weft RPC
//   ────────────────────────────────────────────────────────────────────
//   Create        →  CreateVolumeSnapshot
//   Read          →  ListVolumeSnapshots (filtered to the snapshot's uuid)
//   Update        →  none — snapshots are immutable. RequiresReplace
//                    on every operator-facing attribute so a config
//                    change forces destroy+create instead of a silent
//                    no-op.
//   Delete        →  DeleteVolumeSnapshot
//   Import        →  uuid
//
// Restore (clone the snapshot into a fresh volume) is intentionally NOT
// represented as a Terraform resource — it's a one-shot operation, not
// a stateful object. Operators run it via `weft volume restore-snapshot`
// or a future `weft_volume_from_snapshot` resource type. Modelling it
// as a stateful resource here would conflate "the snapshot exists" with
// "the restore happened", which gets messy on `terraform destroy`.

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

// NewVolumeSnapshotResource is the framework constructor.
func NewVolumeSnapshotResource() resource.Resource { return &volumeSnapshotResource{} }

type volumeSnapshotResource struct {
	client *weftClient
}

type volumeSnapshotModel struct {
	ID         types.String `tfsdk:"id"`
	UUID       types.String `tfsdk:"uuid"`
	VolumeUUID types.String `tfsdk:"volume_uuid"`
	Name       types.String `tfsdk:"name"`
	Project    types.String `tfsdk:"project"`
	SizeGiB    types.Int64  `tfsdk:"size_gib"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

func (r *volumeSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_snapshot"
}

func (r *volumeSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Every operator-facing field requires-replace: snapshots have no
	// rename/resize RPC server-side. The Terraform contract therefore is
	// "config change ⇒ new snapshot" — operators who want a rename
	// delete + recreate.
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Point-in-time CoW copy of a weft volume (reflink/FICLONE-backed).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Mirrors `uuid`; Terraform ID convention.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "Server-minted snapshot UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"volume_uuid": schema.StringAttribute{
				Required:      true,
				Description:   "UUID of the parent volume to snapshot. Immutable.",
				PlanModifiers: requiresReplace,
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Operator-chosen name. Unique inside the parent volume's project.",
				PlanModifiers: requiresReplace,
			},
			"project": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Project name or UUID. Empty inherits the caller's default project (mirrors CreateVolume).",
				PlanModifiers: requiresReplace,
			},
			"size_gib": schema.Int64Attribute{
				Computed:    true,
				Description: "Snapshot's size in GiB (same as parent at snapshot time).",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Unix nanosecond timestamp of snapshot creation.",
			},
		},
	}
}

func (r *volumeSnapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *volumeSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeSnapshotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.vms.CreateVolumeSnapshot(ctx, &weftv1.CreateVolumeSnapshotRequest{
		VolumeUuid: plan.VolumeUUID.ValueString(),
		Name:       plan.Name.ValueString(),
		Project:    plan.Project.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("CreateVolumeSnapshot failed", err.Error())
		return
	}
	applyVolumeSnapshotInfo(&plan, out.GetSnapshot())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// No GetVolumeSnapshot RPC — filter ListVolumeSnapshots by uuid.
	// Cheap enough given snapshots are dozens per volume in practice;
	// if pagination becomes painful, add a server-side Get later.
	list, err := r.client.vms.ListVolumeSnapshots(ctx, &weftv1.ListVolumeSnapshotsRequest{
		VolumeUuid: state.VolumeUUID.ValueString(),
		Project:    state.Project.ValueString(),
	})
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("ListVolumeSnapshots failed", err.Error())
		return
	}
	target := state.UUID.ValueString()
	for _, s := range list.GetSnapshots() {
		if s.GetUuid() == target {
			applyVolumeSnapshotInfo(&state, s)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	// Snapshot gone server-side — drop from state.
	resp.State.RemoveResource(ctx)
}

// Update — every operator-facing attribute is RequiresReplace, so this
// only fires for pure-Computed drift (re-Read). Honor the framework
// contract by re-reading + writing state.
func (r *volumeSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state volumeSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.refresh(ctx, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// refresh is the post-Create / Update helper that re-fetches the
// snapshot to capture server-side fields (createdAt may have advanced,
// rare but possible). Pulled out to keep Update minimal.
func (r *volumeSnapshotResource) refresh(ctx context.Context, m *volumeSnapshotModel, diags interface{ AddError(string, string) }) {
	if r.client == nil || m.UUID.ValueString() == "" {
		return
	}
	list, err := r.client.vms.ListVolumeSnapshots(ctx, &weftv1.ListVolumeSnapshotsRequest{
		VolumeUuid: m.VolumeUUID.ValueString(),
		Project:    m.Project.ValueString(),
	})
	if err != nil {
		if isNotFound(err) {
			m.UUID = types.StringValue("")
		}
		return
	}
	target := m.UUID.ValueString()
	for _, s := range list.GetSnapshots() {
		if s.GetUuid() == target {
			applyVolumeSnapshotInfo(m, s)
			return
		}
	}
	m.UUID = types.StringValue("")
}

func (r *volumeSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.vms.DeleteVolumeSnapshot(ctx, &weftv1.DeleteVolumeSnapshotRequest{Uuid: state.UUID.ValueString()})
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("DeleteVolumeSnapshot failed", err.Error())
	}
}

// ImportState accepts the snapshot UUID as the import ID. Project +
// volume_uuid are populated by the next Read so the operator only has
// to know the UUID.
func (r *volumeSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
}

// applyVolumeSnapshotInfo reflects a VolumeSnapshotInfo into the Model.
func applyVolumeSnapshotInfo(m *volumeSnapshotModel, s *weftv1.VolumeSnapshotInfo) {
	if s == nil {
		return
	}
	m.ID = types.StringValue(s.GetUuid())
	m.UUID = types.StringValue(s.GetUuid())
	m.VolumeUUID = types.StringValue(s.GetVolumeUuid())
	m.Name = types.StringValue(s.GetName())
	m.Project = types.StringValue(s.GetProject())
	m.SizeGiB = types.Int64Value(s.GetSizeGib())
	m.CreatedAt = types.StringValue(fmt.Sprintf("%d", s.GetCreatedAtUnixNs()))
}
