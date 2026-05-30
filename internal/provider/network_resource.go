// network_resource.go — `weft_network` closes the Networks-service gap
// from GAPS.md. The proto exposes Create/Delete plus three dedicated
// mutators (RenameNetwork, SetNetworkDNS, SetNetworkDefaultSecurityGroups);
// there is no GetNetwork RPC, so Read paginates ListNetworks and filters
// by UUID — the cost is one extra RPC at refresh time, the benefit is
// not bloating the proto with Get* boilerplate the server doesn't need.
//
// Lifecycle mapping:
//
//   Terraform op  →  weft RPC
//   ────────────────────────────────────────────────────────────────────
//   Create        →  CreateNetwork
//   Read          →  ListNetworks(project) + uuid filter
//   Update name   →  RenameNetwork
//   Update dns    →  SetNetworkDNS
//   Update sgs    →  SetNetworkDefaultSecurityGroups
//   Delete        →  DeleteNetwork
//   Import        →  "<project>/<uuid>"
//
// `project`, `cidr`, `gateway` and `type` are RequiresReplace (changing
// any of those means a different network in the proto's data model).
// `name`, `dns_servers` and `default_security_group_uuids` are mutable.

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

func NewNetworkResource() resource.Resource { return &networkResource{} }

type networkResource struct {
	client *weftClient
}

type networkModel struct {
	ID                  types.String `tfsdk:"id"`
	UUID                types.String `tfsdk:"uuid"`
	Project             types.String `tfsdk:"project"`
	ProjectUUID         types.String `tfsdk:"project_uuid"`
	Name                types.String `tfsdk:"name"`
	CIDR                types.String `tfsdk:"cidr"`
	Gateway             types.String `tfsdk:"gateway"`
	DNSServers          types.List   `tfsdk:"dns_servers"`
	Type                types.String `tfsdk:"type"`
	DefaultSecurityGrps types.List   `tfsdk:"default_security_group_uuids"`
	CreatedAt           types.String `tfsdk:"created_at"`
}

func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Virtual network within a project. Mirrors CreateNetwork / RenameNetwork / SetNetworkDNS / SetNetworkDefaultSecurityGroups / DeleteNetwork.",
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
				Description: "Server-minted network UUID.",
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
				Description: "Resolved project UUID (computed from `project`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Network name (display label). Mutable via RenameNetwork.",
			},
			"cidr": schema.StringAttribute{
				Required:      true,
				Description:   "IPv4/IPv6 CIDR (e.g. \"10.42.0.0/24\"). Immutable.",
				PlanModifiers: requiresReplace,
			},
			"gateway": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Gateway address inside the CIDR. Immutable.",
				PlanModifiers: requiresReplace,
			},
			"dns_servers": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Resolvers handed to VMs joining this network. Mutable via SetNetworkDNS.",
			},
			"type": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   `Network backend: "nat" | "bridged" | "isolated". Defaults to "nat". Immutable.`,
				PlanModifiers: requiresReplace,
			},
			"default_security_group_uuids": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "SG UUIDs auto-applied to every NIC joining this network. Mutable via SetNetworkDefaultSecurityGroups.",
			},
			"created_at": schema.StringAttribute{Computed: true, Description: "Unix nanosecond timestamp of creation."},
		},
	}
}

func (r *networkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createReq := &weftv1.CreateNetworkRequest{
		Project: plan.Project.ValueString(),
		Name:    plan.Name.ValueString(),
		Cidr:    plan.CIDR.ValueString(),
		Gateway: plan.Gateway.ValueString(),
		Type:    plan.Type.ValueString(),
	}
	if !plan.DNSServers.IsNull() && !plan.DNSServers.IsUnknown() {
		resp.Diagnostics.Append(plan.DNSServers.ElementsAs(ctx, &createReq.DnsServers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	out, err := r.client.vms.CreateNetwork(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("CreateNetwork failed", err.Error())
		return
	}
	plan.applyNetworkInfo(ctx, out.GetNetwork())

	// Default SGs are not part of CreateNetwork; if the operator supplied
	// some, apply them in a follow-up so create-with-SGs is one apply.
	if !plan.DefaultSecurityGrps.IsNull() && !plan.DefaultSecurityGrps.IsUnknown() {
		var sgs []string
		resp.Diagnostics.Append(plan.DefaultSecurityGrps.ElementsAs(ctx, &sgs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		out2, err := r.client.vms.SetNetworkDefaultSecurityGroups(ctx, &weftv1.SetNetworkDefaultSecurityGroupsRequest{
			Uuid:               plan.UUID.ValueString(),
			SecurityGroupUuids: sgs,
		})
		if err != nil {
			resp.Diagnostics.AddError("SetNetworkDefaultSecurityGroups failed", err.Error())
			return
		}
		plan.applyNetworkInfo(ctx, out2.GetNetwork())
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	info, found, err := findNetwork(ctx, r.client, state.Project.ValueString(), state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ListNetworks failed", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.applyNetworkInfo(ctx, info)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state networkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	var latest *weftv1.NetworkInfo

	if !plan.Name.Equal(state.Name) {
		out, err := r.client.vms.RenameNetwork(ctx, &weftv1.RenameNetworkRequest{Uuid: uuid, NewName: plan.Name.ValueString()})
		if err != nil {
			resp.Diagnostics.AddError("RenameNetwork failed", err.Error())
			return
		}
		latest = out.GetNetwork()
	}
	if !plan.DNSServers.Equal(state.DNSServers) {
		var dns []string
		if !plan.DNSServers.IsNull() && !plan.DNSServers.IsUnknown() {
			resp.Diagnostics.Append(plan.DNSServers.ElementsAs(ctx, &dns, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		out, err := r.client.vms.SetNetworkDNS(ctx, &weftv1.SetNetworkDNSRequest{Uuid: uuid, DnsServers: dns})
		if err != nil {
			resp.Diagnostics.AddError("SetNetworkDNS failed", err.Error())
			return
		}
		latest = out.GetNetwork()
	}
	if !plan.DefaultSecurityGrps.Equal(state.DefaultSecurityGrps) {
		var sgs []string
		if !plan.DefaultSecurityGrps.IsNull() && !plan.DefaultSecurityGrps.IsUnknown() {
			resp.Diagnostics.Append(plan.DefaultSecurityGrps.ElementsAs(ctx, &sgs, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		out, err := r.client.vms.SetNetworkDefaultSecurityGroups(ctx, &weftv1.SetNetworkDefaultSecurityGroupsRequest{Uuid: uuid, SecurityGroupUuids: sgs})
		if err != nil {
			resp.Diagnostics.AddError("SetNetworkDefaultSecurityGroups failed", err.Error())
			return
		}
		latest = out.GetNetwork()
	}

	if latest == nil {
		// No mutator was called (no diff on mutable fields). Refresh from
		// the server so Computed fields stay current.
		info, found, err := findNetwork(ctx, r.client, state.Project.ValueString(), uuid)
		if err != nil {
			resp.Diagnostics.AddError("ListNetworks failed", err.Error())
			return
		}
		if !found {
			resp.State.RemoveResource(ctx)
			return
		}
		latest = info
	}
	plan.applyNetworkInfo(ctx, latest)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.vms.DeleteNetwork(ctx, &weftv1.DeleteNetworkRequest{Uuid: state.UUID.ValueString()}); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("DeleteNetwork failed", err.Error())
	}
}

// ImportState accepts "<project>/<uuid>" because Read needs both — the
// proto's ListNetworks filters by project.
func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("invalid import id", `expected "<project>/<uuid>"`)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
}

// findNetwork pages through ListNetworks until it finds the target UUID,
// or runs out of pages. Centralised so Read and Update share the lookup.
func findNetwork(ctx context.Context, c *weftClient, project, uuid string) (*weftv1.NetworkInfo, bool, error) {
	var token string
	for {
		out, err := c.vms.ListNetworks(ctx, &weftv1.ListNetworksRequest{Project: project, PageToken: token})
		if err != nil {
			return nil, false, err
		}
		for _, n := range out.GetNetworks() {
			if n.GetUuid() == uuid {
				return n, true, nil
			}
		}
		token = out.GetNextPageToken()
		if token == "" {
			return nil, false, nil
		}
	}
}

func (m *networkModel) applyNetworkInfo(ctx context.Context, n *weftv1.NetworkInfo) diag.Diagnostics {
	var diags diag.Diagnostics
	if n == nil {
		return diags
	}
	m.ID = types.StringValue(n.GetUuid())
	m.UUID = types.StringValue(n.GetUuid())
	m.ProjectUUID = types.StringValue(n.GetProjectUuid())
	m.Name = types.StringValue(n.GetName())
	m.CIDR = types.StringValue(n.GetCidr())
	m.Gateway = types.StringValue(n.GetGateway())
	m.Type = types.StringValue(n.GetType())
	m.DNSServers, _ = types.ListValueFrom(ctx, types.StringType, n.GetDnsServers())
	m.DefaultSecurityGrps, _ = types.ListValueFrom(ctx, types.StringType, n.GetDefaultSecurityGroupUuids())
	m.CreatedAt = types.StringValue(strconv.FormatInt(n.GetCreatedAtUnixNs(), 10))
	return diags
}
