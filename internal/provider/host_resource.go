// host_resource.go — `weft_host` is the first NEW (non-migrated) resource
// added to the framework half. It addresses the biggest gap surfaced by
// GAPS.md: the Hosts service had three RPCs (RegisterHost, GetHost,
// DeleteHost) that no Terraform resource covered.
//
// Lifecycle mapping:
//
//   Terraform op  →  weft RPC
//   ────────────────────────────────────────────────────────────────────
//   Create        →  RegisterHost(empty UUID)        // server mints UUID
//   Read          →  GetHost(uuid)
//   Update        →  RegisterHost(uuid)              // re-register, idempotent
//   Delete        →  DeleteHost(uuid)
//   Import        →  uuid → GetHost
//
// hostname + hypervisor + architecture are RequiresReplace (changing those
// implies a different physical host). az/rack/endpoint/properties/network_types/
// volume_backends are mutable — `weft up` may move a host between racks
// during a cluster reshuffle, or rewrite properties for selector changes, and we want
// the Terraform path to model that without forcing a destroy/recreate.

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
)

// isNotFound mirrors instance_resource.go's "not found" detection — gRPC
// errors don't carry a structured "missing" code in our server today, so
// we match on the error text. Centralised here so future RPC additions
// can reuse it.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

// NewHostResource is the framework constructor. Register it in
// framework_provider.go's Resources() once registered alongside the other
// framework resources.
func NewHostResource() resource.Resource { return &hostResource{} }

type hostResource struct {
	client *weftClient
}

type hostModel struct {
	ID             types.String `tfsdk:"id"`
	UUID           types.String `tfsdk:"uuid"`
	Hostname       types.String `tfsdk:"hostname"`
	AZ             types.String `tfsdk:"az"`
	Rack           types.String `tfsdk:"rack"`
	Endpoint       types.String `tfsdk:"endpoint"`
	Hypervisor     types.String `tfsdk:"hypervisor"`
	Architecture   types.String `tfsdk:"architecture"`
	NetworkTypes   types.List   `tfsdk:"network_types"`
	VolumeBackends types.List   `tfsdk:"volume_backends"`
	Properties     types.Map    `tfsdk:"properties"`
	State          types.String `tfsdk:"state"`
	CreatedAt      types.String `tfsdk:"created_at"`
	LastSeenAt     types.String `tfsdk:"last_seen_at"`
}

func (r *hostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (r *hostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Registers a host in the weft cluster's host registry. Mirrors the RegisterHost / GetHost / DeleteHost RPCs.",
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
				Description: "Server-minted host UUID. Stable across re-registrations.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Required:      true,
				Description:   "DNS hostname or operator-visible name. Immutable.",
				PlanModifiers: requiresReplace,
			},
			"hypervisor": schema.StringAttribute{
				Required:      true,
				Description:   `Hypervisor type: "apple-vz" | "qemu-kvm" | "libvirt" — typically "qemu-kvm" on Linux hosts. Immutable.`,
				PlanModifiers: requiresReplace,
			},
			"architecture": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   `CPU architecture: "arm64" | "amd64" | "riscv64". Inferred by the agent on first register if left blank. Immutable.`,
				PlanModifiers: requiresReplace,
			},
			"az": schema.StringAttribute{
				Required:    true,
				Description: "Availability zone (failure domain). Mutable — `weft up` may relabel during a cluster reshuffle.",
			},
			"rack": schema.StringAttribute{
				Required:    true,
				Description: "Rack (sub-AZ failure domain). Mutable. Required even in single-rack clusters; placement rules read it.",
			},
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "host:port of the agent's gRPC listener. Mutable (an IP/port change is allowed without forcing replace).",
			},
			"network_types": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: `Network backends the host supports. Defaults to ["nat","bridged","isolated","mesh"].`,
			},
			"volume_backends": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: `Volume backends the host supports. Defaults to ["file"].`,
			},
			"properties": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Operator-set free-form key/value properties. Read by scheduling rule selectors.",
			},
			"state":         schema.StringAttribute{Computed: true, Description: `active | draining | down — observed by the control plane.`},
			"created_at":    schema.StringAttribute{Computed: true, Description: "Unix nanosecond timestamp of first registration."},
			"last_seen_at":  schema.StringAttribute{Computed: true, Description: "Unix nanosecond timestamp of most recent heartbeat."},
		},
	}
}

func (r *hostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *hostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	reqMsg, diags := plan.toRegisterRequest(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Force empty UUID on Create — the server mints a fresh one. If the
	// operator somehow set uuid via config, that's a misuse; we silently
	// ignore (uuid is Computed in the schema so config can't set it).
	reqMsg.Uuid = ""
	out, err := r.client.vms.RegisterHost(ctx, reqMsg)
	if err != nil {
		resp.Diagnostics.AddError("RegisterHost failed", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.applyHostInfo(ctx, out.GetHost())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.vms.GetHost(ctx, &weftv1.GetHostRequest{Uuid: state.UUID.ValueString()})
	if err != nil {
		// "not found" → drop from state; transient → keep, retry next plan.
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GetHost failed", err.Error())
		return
	}
	resp.Diagnostics.Append(state.applyHostInfo(ctx, out.GetHost())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state hostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Carry the UUID forward so the re-register is idempotent (same UUID
	// = refresh in place, no new mint).
	reqMsg, diags := plan.toRegisterRequest(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	reqMsg.Uuid = state.UUID.ValueString()
	out, err := r.client.vms.RegisterHost(ctx, reqMsg)
	if err != nil {
		resp.Diagnostics.AddError("RegisterHost (update) failed", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.applyHostInfo(ctx, out.GetHost())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.vms.DeleteHost(ctx, &weftv1.DeleteHostRequest{Uuid: state.UUID.ValueString()}); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("DeleteHost failed", err.Error())
	}
}

// ImportState accepts the host UUID as the import ID.
func (r *hostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
}

// toRegisterRequest projects the typed model onto the proto message,
// honouring the framework's tri-state (null / unknown / value) for
// optional fields. Shared between Create and Update.
func (m *hostModel) toRegisterRequest(ctx context.Context) (*weftv1.RegisterHostRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	reqMsg := &weftv1.RegisterHostRequest{
		Hostname:     m.Hostname.ValueString(),
		Az:           m.AZ.ValueString(),
		Rack:         m.Rack.ValueString(),
		Endpoint:     m.Endpoint.ValueString(),
		Hypervisor:   m.Hypervisor.ValueString(),
		Architecture: m.Architecture.ValueString(),
	}
	if !m.NetworkTypes.IsNull() && !m.NetworkTypes.IsUnknown() {
		diags.Append(m.NetworkTypes.ElementsAs(ctx, &reqMsg.NetworkTypes, false)...)
	}
	if !m.VolumeBackends.IsNull() && !m.VolumeBackends.IsUnknown() {
		diags.Append(m.VolumeBackends.ElementsAs(ctx, &reqMsg.VolumeBackends, false)...)
	}
	if !m.Properties.IsNull() && !m.Properties.IsUnknown() {
		props := make(map[string]string)
		diags.Append(m.Properties.ElementsAs(ctx, &props, false)...)
		reqMsg.Properties = props
	}
	return reqMsg, diags
}

// applyHostInfo reflects the server's response back into the model so
// Computed fields stay in sync with state.
func (m *hostModel) applyHostInfo(ctx context.Context, h *weftv1.HostInfo) diag.Diagnostics {
	var diags diag.Diagnostics
	if h == nil {
		return diags
	}
	m.ID = types.StringValue(h.GetUuid())
	m.UUID = types.StringValue(h.GetUuid())
	m.Hostname = types.StringValue(h.GetHostname())
	m.AZ = types.StringValue(h.GetAz())
	m.Rack = types.StringValue(h.GetRack())
	m.Endpoint = types.StringValue(h.GetEndpoint())
	m.Hypervisor = types.StringValue(h.GetHypervisor())
	m.Architecture = types.StringValue(h.GetArchitecture())
	nt, d := types.ListValueFrom(ctx, types.StringType, h.GetNetworkTypes())
	diags.Append(d...)
	m.NetworkTypes = nt
	vb, d := types.ListValueFrom(ctx, types.StringType, h.GetVolumeBackends())
	diags.Append(d...)
	m.VolumeBackends = vb
	props, d := types.MapValueFrom(ctx, types.StringType, h.GetProperties())
	diags.Append(d...)
	m.Properties = props
	m.State = types.StringValue(h.GetState())
	m.CreatedAt = types.StringValue(strconv.FormatInt(h.GetCreatedAtUnixNs(), 10))
	m.LastSeenAt = types.StringValue(strconv.FormatInt(h.GetLastSeenAtUnixNs(), 10))
	return diags
}
