// keypair_resource.go — `weft_keypair` ported from sdk/v2 to
// terraform-plugin-framework. Local-only resource (no gRPC): reads the public
// key from <file_path>.pub and exposes it as a computed attribute so other
// resources can reference it without hard-coding file paths.

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewKeypairResource() resource.Resource { return &keypairResource{} }

type keypairResource struct{}

type keypairModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	FilePath     types.String `tfsdk:"file_path"`
	ResolvedPath types.String `tfsdk:"resolved_path"`
	PublicKey    types.String `tfsdk:"public_key"`
}

func (r *keypairResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keypair"
}

func (r *keypairResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplaceStr := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		Description: "Declares an SSH keypair — mirrors the mock HCL `keypair` block. Exposes the resolved file_path and public_key for reference by weft_instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Keypair identifier (mirrors `name`).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Keypair identifier — matches the mock HCL keypair block label (e.g. \"mock\").",
				PlanModifiers: requiresReplaceStr,
			},
			"file_path": schema.StringAttribute{
				Required:      true,
				Description:   "Path to the SSH private key file — matches mock HCL keypair.file_path. Tilde (~) is expanded.",
				PlanModifiers: requiresReplaceStr,
			},
			"resolved_path": schema.StringAttribute{
				Computed:    true,
				Description: "Absolute path to the private key after tilde expansion.",
			},
			"public_key": schema.StringAttribute{
				Computed:    true,
				Description: "Content of <file_path>.pub — ready to inject into authorized_keys or cloud-init.",
			},
		},
	}
}

// Configure is a no-op (no gRPC client needed), but we still need to satisfy
// the framework — Configure is called even when ProviderData isn't used.
func (r *keypairResource) Configure(_ context.Context, _ resource.ConfigureRequest, _ *resource.ConfigureResponse) {
}

func (r *keypairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keypairModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = plan.Name
	if err := readKeypairFiles(&plan); err != nil {
		resp.Diagnostics.AddError("read public key", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keypairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keypairModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := readKeypairFiles(&state); err != nil {
		resp.Diagnostics.AddError("read public key", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *keypairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All stateful attrs are RequiresReplace; Update should be a no-op refresh.
	var plan keypairModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := readKeypairFiles(&plan); err != nil {
		resp.Diagnostics.AddError("read public key", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keypairResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *keypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func readKeypairFiles(m *keypairModel) error {
	resolved := expandHome(m.FilePath.ValueString())
	pub, err := os.ReadFile(resolved + ".pub")
	if err != nil {
		return fmt.Errorf("read public key %s.pub: %w", resolved, err)
	}
	m.ResolvedPath = types.StringValue(resolved)
	m.PublicKey = types.StringValue(strings.TrimSpace(string(pub)))
	return nil
}
