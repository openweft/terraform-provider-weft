// instance_resource.go — `weft_instance` ported from sdk/v2 to
// terraform-plugin-framework.
//
// Schema model: nested blocks (disk { patch { add/del/mod {} } }, ssh {})
// are expressed as framework Blocks (not Attributes) so the HCL syntax stays
// identical to the sdk/v2 era — operators don't have to rewrite their .tf.
// State file shape stays identical too (protocol 6 was already in use via
// tf5to6server in the sdk/v2 half).

package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
)

// NewInstanceResource is the framework constructor.
func NewInstanceResource() resource.Resource { return &instanceResource{} }

type instanceResource struct {
	client *weftClient
}

// instanceModel is the framework's typed model for HCL ↔ state binding.
// Tag names must match the schema attribute keys.
type instanceModel struct {
	Name  types.String `tfsdk:"name"`
	Cpu   types.Int64  `tfsdk:"cpu"`
	Mem   types.Int64  `tfsdk:"mem"`
	Disk  *diskModel   `tfsdk:"disk"`
	SSH   *sshModel    `tfsdk:"ssh"`
	IP    types.String `tfsdk:"ip"`
	State types.String `tfsdk:"state"`
	ID    types.String `tfsdk:"id"`
}

type diskModel struct {
	From  types.String `tfsdk:"from"`
	Size  types.String `tfsdk:"size"`
	Patch *patchModel  `tfsdk:"patch"`
}

type patchModel struct {
	Add []addOp `tfsdk:"add"`
	Del []delOp `tfsdk:"del"`
	Mod []modOp `tfsdk:"mod"`
}

type addOp struct {
	Content types.String `tfsdk:"content"`
	Dst     types.String `tfsdk:"dst"`
	Trigger types.String `tfsdk:"trigger"`
}

type delOp struct {
	Dst types.String `tfsdk:"dst"`
}

type modOp struct {
	Dst types.String `tfsdk:"dst"`
	Old types.String `tfsdk:"old"`
	New types.String `tfsdk:"new"`
}

type sshModel struct {
	User        types.String `tfsdk:"user"`
	KeypairPath types.String `tfsdk:"keypair_path"`
}

func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Plan-modifier shortcuts. Every "force-new" field gets the
	// RequiresReplace modifier; computed fields that don't change between
	// plan-time and runtime get UseStateForUnknown so plan stays clean.
	requiresReplaceStr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	requiresReplaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		Description: "Provisions and manages a VM via weft.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "VM name (mirrors the `name` attribute; satisfies the Terraform ID convention).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Unique VM name.",
				PlanModifiers: requiresReplaceStr,
			},
			"cpu": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Number of vCPUs (default 2).",
				PlanModifiers: requiresReplaceInt,
			},
			"mem": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Memory in GiB (default 2).",
				PlanModifiers: requiresReplaceInt,
			},
			"ip":    schema.StringAttribute{Computed: true, Description: "IP address of the first replica."},
			"state": schema.StringAttribute{Computed: true, Description: "State of the first replica."},
		},
		Blocks: map[string]schema.Block{
			"disk": schema.SingleNestedBlock{
				Description: "Boot disk configuration.",
				Attributes: map[string]schema.Attribute{
					"from": schema.StringAttribute{
						Required:      true,
						Description:   "Resolved image URL for the boot disk.",
						PlanModifiers: requiresReplaceStr,
					},
					"size": schema.StringAttribute{
						Optional:      true,
						Computed:      true,
						Description:   `Disk size with unit suffix (e.g. "20Gi", "100Gi"). Defaults to "20Gi".`,
						PlanModifiers: requiresReplaceStr,
					},
				},
				Blocks: map[string]schema.Block{
					"patch": schema.SingleNestedBlock{
						Description: "File patch operations applied to the disk image before first boot.",
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
			},
			"ssh": schema.SingleNestedBlock{
				Description: "SSH configuration.",
				Attributes: map[string]schema.Attribute{
					"user":         schema.StringAttribute{Optional: true, Computed: true, Description: `SSH username (default "ubuntu").`, PlanModifiers: requiresReplaceStr},
					"keypair_path": schema.StringAttribute{Optional: true, Description: "Path to the SSH private key file; the public key is read from <keypair_path>.pub.", PlanModifiers: requiresReplaceStr},
				},
			},
		},
	}
}

func (r *instanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return // provider not configured yet; framework calls again later
	}
	client, ok := req.ProviderData.(*weftClient)
	if !ok {
		resp.Diagnostics.AddError("provider data type", fmt.Sprintf("expected *weftClient, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cpu := int64(2)
	if !plan.Cpu.IsNull() && !plan.Cpu.IsUnknown() {
		cpu = plan.Cpu.ValueInt64()
	}
	memGiB := int64(2)
	if !plan.Mem.IsNull() && !plan.Mem.IsUnknown() {
		memGiB = plan.Mem.ValueInt64()
	}

	if plan.Disk == nil {
		resp.Diagnostics.AddAttributeError(path.Root("disk"), "missing", "disk block is required")
		return
	}
	size := "20Gi"
	if !plan.Disk.Size.IsNull() && !plan.Disk.Size.IsUnknown() && plan.Disk.Size.ValueString() != "" {
		size = plan.Disk.Size.ValueString()
	}
	diskGiB, ok := parseSizeGiB(size)
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("disk").AtName("size"), "invalid",
			fmt.Sprintf("invalid disk size %q: use a unit suffix like 20Gi", size))
		return
	}

	sshPubKey := ""
	if plan.SSH != nil && !plan.SSH.KeypairPath.IsNull() && plan.SSH.KeypairPath.ValueString() != "" {
		kp := expandHome(plan.SSH.KeypairPath.ValueString())
		pubKeyData, err := os.ReadFile(kp + ".pub")
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("ssh").AtName("keypair_path"), "read pub key",
				fmt.Sprintf("read SSH public key %s.pub: %v", kp, err))
			return
		}
		sshPubKey = strings.TrimSpace(string(pubKeyData))
	}

	var fileOps []*weftv1.DiskFileOp
	var deleteOps []*weftv1.DiskDeleteOp
	var modOps []*weftv1.DiskModOp
	if plan.Disk.Patch != nil {
		for _, a := range plan.Disk.Patch.Add {
			fileOps = append(fileOps, &weftv1.DiskFileOp{
				Content: a.Content.ValueString(),
				Dst:     a.Dst.ValueString(),
				Trigger: a.Trigger.ValueString(),
			})
		}
		for _, d := range plan.Disk.Patch.Del {
			deleteOps = append(deleteOps, &weftv1.DiskDeleteOp{Dst: d.Dst.ValueString()})
		}
		for _, m := range plan.Disk.Patch.Mod {
			modOps = append(modOps, &weftv1.DiskModOp{
				Dst: m.Dst.ValueString(),
				Old: m.Old.ValueString(),
				New: m.New.ValueString(),
			})
		}
	}

	vmName := plan.Name.ValueString()
	provReq := &weftv1.ProvisionVMRequest{
		Name:      vmName,
		Image:     plan.Disk.From.ValueString(),
		Cpu:       uint32(cpu),
		MemMb:     uint64(memGiB) * 1024,
		DiskGb:    uint64(diskGiB),
		SshPubKey: sshPubKey,
		FileOps:   fileOps,
		DeleteOps: deleteOps,
		ModOps:    modOps,
	}
	if _, err := r.client.vms.ProvisionVM(ctx, provReq); err != nil {
		resp.Diagnostics.AddError("ProvisionVM failed", err.Error())
		return
	}

	// Echo back defaults + ID so the plan matches state.
	plan.ID = types.StringValue(vmName)
	plan.Cpu = types.Int64Value(cpu)
	plan.Mem = types.Int64Value(memGiB)
	plan.Disk.Size = types.StringValue(size)
	if plan.SSH != nil && (plan.SSH.User.IsNull() || plan.SSH.User.IsUnknown()) {
		plan.SSH.User = types.StringValue("ubuntu")
	}
	// Computed disk patch.add.trigger defaults — preserve operator's
	// explicit empty string but back-fill nulls so state matches plan.
	if plan.Disk.Patch != nil {
		for i := range plan.Disk.Patch.Add {
			if plan.Disk.Patch.Add[i].Trigger.IsNull() || plan.Disk.Patch.Add[i].Trigger.IsUnknown() {
				plan.Disk.Patch.Add[i].Trigger = types.StringValue("")
			}
		}
	}

	r.readInto(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readInto(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	// readInto sets state.ID to "" when the VM is gone; the framework
	// then drops the resource from state automatically.
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// readInto refreshes the computed ip/state fields and clears ID when the
// VM has disappeared on the weft side. Shared between Create and Read.
func (r *instanceResource) readInto(ctx context.Context, m *instanceModel, diags interface{ AddError(string, string) }) {
	if r.client == nil {
		return
	}
	vmName := m.Name.ValueString()
	if vmName == "" {
		// Edge case during Create plan: ID resolves to name.
		vmName = m.ID.ValueString()
	}
	resp, err := r.client.vms.VMStatus(ctx, &weftv1.VMStatusRequest{Name: vmName})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			m.ID = types.StringValue("")
		}
		return
	}
	m.IP = types.StringValue(resp.GetVm().GetIp())
	m.State = types.StringValue(resp.GetVm().GetState().String())
}

// Update — all stateful attrs of weft_instance are RequiresReplace, so
// Update never fires in practice (a config change triggers destroy+create).
// We still satisfy the interface; getting here means the framework saw a
// pure computed-attribute change, which should be a no-op.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan instanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readInto(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.vms.DeprovisionVM(ctx, &weftv1.DeprovisionVMRequest{Name: state.Name.ValueString()}); err != nil {
		resp.Diagnostics.AddError("DeprovisionVM failed", err.Error())
		return
	}
}

// ImportState accepts the VM name as the import ID. Mirrors the sdk/v2
// PassthroughContext behaviour.
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

// parseSizeGiB / expandHome stay duplicated until the sdk/v2 sibling goes
// away — moving them now would mean breaking the sdk/v2 file's compile.

func parseSizeGiB(raw string) (int, bool) {
	s := strings.Trim(raw, "\"")
	m := regexp.MustCompile(`^([0-9]+)(Mi?|Gi?|Ti?)$`).FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, err := strconv.Atoi(m[1])
	if err != nil || v <= 0 {
		return 0, false
	}
	switch strings.ToUpper(m[2])[0] {
	case 'M':
		gib := v / 1024
		if gib <= 0 {
			return 0, false
		}
		return gib, true
	case 'G':
		return v, true
	case 'T':
		return v * 1024, true
	}
	return 0, false
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
