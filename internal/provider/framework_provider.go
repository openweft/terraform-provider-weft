// framework_provider.go — the terraform-plugin-framework half of the muxed
// provider. New resources land here; the sdk/v2 half in provider.go shrinks
// as resources migrate. See FRAMEWORK_MIGRATION.md for the running status
// table.

package provider

import (
	"context"
	"os"

	sshtransport "github.com/grpc-transports/ssh"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	weftv1 "github.com/openweft/weft-proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewFrameworkProvider is the constructor the mux server uses to obtain
// the framework half of the provider. Returns a func per protocol-6 server
// convention so each Terraform graph traversal gets a fresh instance.
func NewFrameworkProvider(version string) func() provider.Provider {
	return func() provider.Provider { return &weftProvider{version: version} }
}

// weftProvider is the framework provider; it shares connection/config logic
// with the sdk/v2 sibling via configureClient.
type weftProvider struct {
	version string
}

// weftProviderModel mirrors the provider block in HCL. Field tags match the
// HCL attribute names — that's how the framework binds Config → struct.
type weftProviderModel struct {
	Socket    types.String `tfsdk:"socket"`
	SSHSocket types.String `tfsdk:"ssh_socket"`
	SSHKey    types.String `tfsdk:"ssh_key"`
}

func (p *weftProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "weft"
	resp.Version = p.version
}

func (p *weftProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for weft — provisions VMs via the weft gRPC API.",
		Attributes: map[string]schema.Attribute{
			"socket": schema.StringAttribute{
				Optional:    true,
				Description: "Unix socket path for weft (plain, no auth). Defaults to $WEFT_SOCKET or ~/.weft/weft.sock.",
			},
			"ssh_socket": schema.StringAttribute{
				Optional:    true,
				Description: "weft SSH socket path. Defaults to $WEFT_SSH_SOCKET or ~/.weft/weft-ssh.sock when ssh_key is set.",
			},
			"ssh_key": schema.StringAttribute{
				Optional:    true,
				Description: "Path to SSH private key for authentication. Setting this enables the SSH transport.",
			},
		},
	}
}

// Configure builds the gRPC client and stashes it on resp.ResourceData /
// resp.DataSourceData so each resource's Configure() can pick it up.
func (p *weftProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg weftProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	socket := envOr(cfg.Socket, "WEFT_SOCKET", "unix://"+defaultSocket())
	sshSocket := envOr(cfg.SSHSocket, "WEFT_SSH_SOCKET", "")
	sshKey := envOr(cfg.SSHKey, "WEFT_SSH_KEY", "")

	client, err := buildWeftClient(socket, sshSocket, sshKey)
	if err != nil {
		// Surface the error at the provider-block level so the operator
		// sees the cause attached to the `provider "weft" {}` block.
		resp.Diagnostics.AddAttributeError(path.Root("socket"), "weft: cannot connect", err.Error())
		return
	}
	resp.ResourceData = client
	resp.DataSourceData = client
}

// Resources returns the resources served by the framework half. The sdk/v2
// half's ResourcesMap is now empty — see FRAMEWORK_MIGRATION.md.
func (p *weftProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewKeypairResource,
		NewEndpointResource,
		NewDeploymentResource,
		NewImageResource,
		NewImagePatchResource,
		NewImagesResource,
		NewHostResource, // first NEW (non-migrated) framework resource — closes the biggest Hosts-service gap from GAPS.md.
		NewNetworkResource,
		NewVolumeResource,
		NewSecurityGroupResource,
		NewTenantResource,
		// weft_scheduling_rule: skipped — the proto has no SchedulingRule
		// service (scheduling_rule is a label on CreateVMRequest, not a
		// standalone CRUD resource). See commit message + GAPS.md.
	}
}

// DataSources returns the data sources served by the framework half.
func (p *weftProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewConfigDataSource,
	}
}

// envOr resolves a configured attribute against env-default-or-fallback.
// The framework doesn't have sdk/v2's EnvDefaultFunc shortcut so we do it
// here to keep the operator's `WEFT_*` env vars working unchanged.
func envOr(v types.String, env, fallback string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	if e := os.Getenv(env); e != "" {
		return e
	}
	return fallback
}

// buildWeftClient is the shared dial logic the sdk/v2 provider.go and the
// framework Configure both call. Centralised here to keep their behaviour
// in lock-step during the incremental migration.
func buildWeftClient(socket, sshSocket, sshKey string) (*weftClient, error) {
	var conn *grpc.ClientConn
	var err error

	if sshKey != "" {
		if sshSocket == "" {
			sshSocket = defaultSSHSocket()
		}
		var sshOpt grpc.DialOption
		sshOpt, err = sshtransport.DialOption("unix:"+sshSocket, sshKey, "")
		if err != nil {
			return nil, err
		}
		conn, err = grpc.NewClient("passthrough:///weft", sshOpt, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		conn, err = grpc.NewClient(socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if err != nil {
		return nil, err
	}
	return &weftClient{conn: conn, vms: weftv1.NewWeftAgentClient(conn)}, nil
}
