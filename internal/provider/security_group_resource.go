// security_group_resource.go — `weft_security_group` closes the
// SecurityGroups-service gap from GAPS.md. Proto exposes
// CreateSecurityGroup / RenameSecurityGroup / SetSecurityGroupDescription /
// SetSecurityGroupRules / DeleteSecurityGroup; Read paginates
// ListSecurityGroups (no GetSecurityGroup in the proto).
//
//   Terraform op       →  weft RPC
//   ────────────────────────────────────────────────────────────────────
//   Create             →  CreateSecurityGroup
//   Read               →  ListSecurityGroups(project) + uuid filter
//   Update name        →  RenameSecurityGroup
//   Update description →  SetSecurityGroupDescription
//   Update rules       →  SetSecurityGroupRules (replace-all semantics)
//   Delete             →  DeleteSecurityGroup
//   Import             →  "<project>/<uuid>"
//
// `project` is RequiresReplace. Everything else mutates in place.

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
)

func NewSecurityGroupResource() resource.Resource { return &securityGroupResource{} }

type securityGroupResource struct {
	client *weftClient
}

type securityGroupModel struct {
	ID          types.String `tfsdk:"id"`
	UUID        types.String `tfsdk:"uuid"`
	Project     types.String `tfsdk:"project"`
	ProjectUUID types.String `tfsdk:"project_uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Rules       types.List   `tfsdk:"rules"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

// securityRuleAttrTypes is the canonical object shape for one entry in
// `rules`. Names match the proto SecurityRule field-for-field so an
// operator reading weft.proto can write HCL without a cheat sheet.
var securityRuleAttrTypes = map[string]attr.Type{
	"direction":         types.StringType,
	"protocol":          types.StringType,
	"port_min":          types.Int64Type,
	"port_max":          types.Int64Type,
	"remote_cidr":       types.StringType,
	"remote_group_uuid": types.StringType,
}

func (r *securityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (r *securityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Security group within a project. Mirrors CreateSecurityGroup / RenameSecurityGroup / SetSecurityGroupDescription / SetSecurityGroupRules / DeleteSecurityGroup.",
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
				Description: "Server-minted SG UUID.",
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
				Description: "Security group name. Mutable via RenameSecurityGroup.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Free-form description. Mutable via SetSecurityGroupDescription.",
			},
			"rules": schema.ListNestedAttribute{
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"direction":         schema.StringAttribute{Required: true, Description: `"ingress" | "egress".`},
						"protocol":          schema.StringAttribute{Required: true, Description: `"tcp" | "udp" | "icmp" | "any".`},
						"port_min":          schema.Int64Attribute{Optional: true, Computed: true, Description: "Inclusive lower port. 0 when N/A."},
						"port_max":          schema.Int64Attribute{Optional: true, Computed: true, Description: "Inclusive upper port. 0 when N/A."},
						"remote_cidr":       schema.StringAttribute{Optional: true, Computed: true, Description: "CIDR block of the peer. Mutually exclusive with `remote_group_uuid`."},
						"remote_group_uuid": schema.StringAttribute{Optional: true, Computed: true, Description: "UUID of a peer security group. Mutually exclusive with `remote_cidr`."},
					},
				},
				Description: "Replace-all rule list. Mutable via SetSecurityGroupRules.",
			},
			"created_at": schema.StringAttribute{Computed: true, Description: "Unix nanosecond timestamp of creation."},
		},
	}
}

func (r *securityGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *securityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan securityGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, diags := planRulesToProto(ctx, plan.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.vms.CreateSecurityGroup(ctx, &weftv1.CreateSecurityGroupRequest{
		Project:     plan.Project.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Rules:       rules,
	})
	if err != nil {
		resp.Diagnostics.AddError("CreateSecurityGroup failed", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.applySecurityGroupInfo(ctx, out.GetGroup())...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *securityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state securityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	info, found, err := findSecurityGroup(ctx, r.client, state.Project.ValueString(), state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ListSecurityGroups failed", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(state.applySecurityGroupInfo(ctx, info)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *securityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state securityGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	var latest *weftv1.SecurityGroupInfo

	if !plan.Name.Equal(state.Name) {
		out, err := r.client.vms.RenameSecurityGroup(ctx, &weftv1.RenameSecurityGroupRequest{Uuid: uuid, NewName: plan.Name.ValueString()})
		if err != nil {
			resp.Diagnostics.AddError("RenameSecurityGroup failed", err.Error())
			return
		}
		latest = out.GetGroup()
	}
	if !plan.Description.Equal(state.Description) {
		out, err := r.client.vms.SetSecurityGroupDescription(ctx, &weftv1.SetSecurityGroupDescriptionRequest{Uuid: uuid, Description: plan.Description.ValueString()})
		if err != nil {
			resp.Diagnostics.AddError("SetSecurityGroupDescription failed", err.Error())
			return
		}
		latest = out.GetGroup()
	}
	if !plan.Rules.Equal(state.Rules) {
		rules, diags := planRulesToProto(ctx, plan.Rules)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		out, err := r.client.vms.SetSecurityGroupRules(ctx, &weftv1.SetSecurityGroupRulesRequest{Uuid: uuid, Rules: rules})
		if err != nil {
			resp.Diagnostics.AddError("SetSecurityGroupRules failed", err.Error())
			return
		}
		latest = out.GetGroup()
	}

	if latest == nil {
		info, found, err := findSecurityGroup(ctx, r.client, state.Project.ValueString(), uuid)
		if err != nil {
			resp.Diagnostics.AddError("ListSecurityGroups failed", err.Error())
			return
		}
		if !found {
			resp.State.RemoveResource(ctx)
			return
		}
		latest = info
	}
	resp.Diagnostics.Append(plan.applySecurityGroupInfo(ctx, latest)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *securityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state securityGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.vms.DeleteSecurityGroup(ctx, &weftv1.DeleteSecurityGroupRequest{Uuid: state.UUID.ValueString()}); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("DeleteSecurityGroup failed", err.Error())
	}
}

func (r *securityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("invalid import id", `expected "<project>/<uuid>"`)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
}

func findSecurityGroup(ctx context.Context, c *weftClient, project, uuid string) (*weftv1.SecurityGroupInfo, bool, error) {
	var token string
	for {
		out, err := c.vms.ListSecurityGroups(ctx, &weftv1.ListSecurityGroupsRequest{Project: project, PageToken: token})
		if err != nil {
			return nil, false, err
		}
		for _, g := range out.GetGroups() {
			if g.GetUuid() == uuid {
				return g, true, nil
			}
		}
		token = out.GetNextPageToken()
		if token == "" {
			return nil, false, nil
		}
	}
}

// planRulesToProto converts the model's typed `rules` list into proto
// SecurityRule values. Centralised so Create and Update share the
// conversion.
func planRulesToProto(ctx context.Context, list types.List) ([]*weftv1.SecurityRule, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	type rowModel struct {
		Direction       types.String `tfsdk:"direction"`
		Protocol        types.String `tfsdk:"protocol"`
		PortMin         types.Int64  `tfsdk:"port_min"`
		PortMax         types.Int64  `tfsdk:"port_max"`
		RemoteCidr      types.String `tfsdk:"remote_cidr"`
		RemoteGroupUuid types.String `tfsdk:"remote_group_uuid"`
	}
	var rows []rowModel
	diags.Append(list.ElementsAs(ctx, &rows, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]*weftv1.SecurityRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, &weftv1.SecurityRule{
			Direction:       r.Direction.ValueString(),
			Protocol:        r.Protocol.ValueString(),
			PortMin:         int32(r.PortMin.ValueInt64()),
			PortMax:         int32(r.PortMax.ValueInt64()),
			RemoteCidr:      r.RemoteCidr.ValueString(),
			RemoteGroupUuid: r.RemoteGroupUuid.ValueString(),
		})
	}
	return out, diags
}

func (m *securityGroupModel) applySecurityGroupInfo(ctx context.Context, g *weftv1.SecurityGroupInfo) diag.Diagnostics {
	var diags diag.Diagnostics
	if g == nil {
		return diags
	}
	m.ID = types.StringValue(g.GetUuid())
	m.UUID = types.StringValue(g.GetUuid())
	m.ProjectUUID = types.StringValue(g.GetProjectUuid())
	m.Name = types.StringValue(g.GetName())
	m.Description = types.StringValue(g.GetDescription())
	m.CreatedAt = types.StringValue(strconv.FormatInt(g.GetCreatedAtUnixNs(), 10))

	elemType := types.ObjectType{AttrTypes: securityRuleAttrTypes}
	elems := make([]attr.Value, 0, len(g.GetRules()))
	for _, r := range g.GetRules() {
		obj, d := types.ObjectValue(securityRuleAttrTypes, map[string]attr.Value{
			"direction":         types.StringValue(r.GetDirection()),
			"protocol":          types.StringValue(r.GetProtocol()),
			"port_min":          types.Int64Value(int64(r.GetPortMin())),
			"port_max":          types.Int64Value(int64(r.GetPortMax())),
			"remote_cidr":       types.StringValue(r.GetRemoteCidr()),
			"remote_group_uuid": types.StringValue(r.GetRemoteGroupUuid()),
		})
		diags.Append(d...)
		elems = append(elems, obj)
	}
	lv, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	m.Rules = lv
	_ = ctx
	return diags
}
