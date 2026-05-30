// tenant_resource.go — `weft_tenant` closes the Tenants-service gap from
// GAPS.md. Proto exposes CreateTenant / DeleteTenant; there is no
// RenameTenant, SetTenantDomain or similar mutator, so every field of
// the create request is RequiresReplace (changing the name or domain
// means a different tenant in the data model). Membership / admin
// management lives in dedicated `weft_tenant_member` /
// `weft_tenant_admin` resources (not in this commit).
//
//   Terraform op       →  weft RPC
//   ────────────────────────────────────────────────────────────────────
//   Create             →  CreateTenant
//   Read               →  ListTenants + uuid filter (no GetTenant in proto)
//   Update             →  RequiresReplace forces destroy/create
//   Delete             →  DeleteTenant
//   Import             →  "<uuid>"

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
)

func NewTenantResource() resource.Resource { return &tenantResource{} }

type tenantResource struct {
	client *weftClient
}

type tenantModel struct {
	ID        types.String `tfsdk:"id"`
	UUID      types.String `tfsdk:"uuid"`
	Name      types.String `tfsdk:"name"`
	Domain    types.String `tfsdk:"domain"`
	Status    types.String `tfsdk:"status"`
	Projects  types.Int64  `tfsdk:"projects"`
	Members   types.Int64  `tfsdk:"members"`
	Admins    types.Int64  `tfsdk:"admins"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *tenantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (r *tenantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Tenant (top-level isolation unit). Mirrors CreateTenant / DeleteTenant. The proto has no mutators for name / domain — both are RequiresReplace.",
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
				Description: "Server-minted tenant UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Tenant display name. Immutable (no RenameTenant in the proto).",
				PlanModifiers: requiresReplace,
			},
			"domain": schema.StringAttribute{
				Required:      true,
				Description:   "Tenant DNS domain (e.g. \"acme.example.com\"). Immutable.",
				PlanModifiers: requiresReplace,
			},
			"status":     schema.StringAttribute{Computed: true, Description: `"active" | "disabled".`},
			"projects":   schema.Int64Attribute{Computed: true, Description: "Count of projects in this tenant."},
			"members":    schema.Int64Attribute{Computed: true, Description: "Count of members (admins included)."},
			"admins":     schema.Int64Attribute{Computed: true, Description: "Count of admins."},
			"created_at": schema.StringAttribute{Computed: true, Description: "Unix nanosecond timestamp of creation."},
		},
	}
}

func (r *tenantResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tenantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tenantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.vms.CreateTenant(ctx, &weftv1.CreateTenantRequest{
		Name:   plan.Name.ValueString(),
		Domain: plan.Domain.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("CreateTenant failed", err.Error())
		return
	}
	plan.applyTenantInfo(out.GetTenant())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tenantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tenantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	info, found, err := findTenant(ctx, r.client, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ListTenants failed", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.applyTenantInfo(info)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only handles refresh — name/domain mutations are
// RequiresReplace, so the framework will Destroy+Create on changes.
// Keeping Update wired (instead of nil-ing it out) means a no-op plan
// refresh still walks through here cleanly.
func (r *tenantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tenantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	info, found, err := findTenant(ctx, r.client, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ListTenants failed", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	plan.applyTenantInfo(info)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tenantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tenantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.vms.DeleteTenant(ctx, &weftv1.DeleteTenantRequest{Uuid: state.UUID.ValueString()}); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("DeleteTenant failed", err.Error())
	}
}

func (r *tenantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
}

func findTenant(ctx context.Context, c *weftClient, uuid string) (*weftv1.TenantInfo, bool, error) {
	out, err := c.vms.ListTenants(ctx, &weftv1.ListTenantsRequest{})
	if err != nil {
		return nil, false, err
	}
	for _, t := range out.GetTenants() {
		if t.GetUuid() == uuid {
			return t, true, nil
		}
	}
	return nil, false, nil
}

func (m *tenantModel) applyTenantInfo(t *weftv1.TenantInfo) {
	if t == nil {
		return
	}
	m.ID = types.StringValue(t.GetUuid())
	m.UUID = types.StringValue(t.GetUuid())
	m.Name = types.StringValue(t.GetName())
	m.Domain = types.StringValue(t.GetDomain())
	m.Status = types.StringValue(t.GetStatus())
	m.Projects = types.Int64Value(int64(t.GetProjects()))
	m.Members = types.Int64Value(int64(t.GetMembers()))
	m.Admins = types.Int64Value(int64(t.GetAdmins()))
	m.CreatedAt = types.StringValue(strconv.FormatInt(t.GetCreatedAtUnixNs(), 10))
}
