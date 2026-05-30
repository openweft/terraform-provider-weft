package provider

import (
	"context"
	"os"

	sshtransport "github.com/grpc-transports/ssh"
	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// New returns the provider factory function.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"socket": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("WEFT_SOCKET", "unix://"+defaultSocket()),
				Description: "Unix socket path for weft (plain, no auth).",
			},
			"ssh_socket": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("WEFT_SSH_SOCKET", ""),
				Description: "weft SSH socket path. Defaults to ~/.weft/weft-ssh.sock when ssh_key is set.",
			},
			"ssh_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("WEFT_SSH_KEY", ""),
				Description: "Path to SSH private key for authentication (enables SSH transport).",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			// weft_instance moved to the framework half — see
			// FRAMEWORK_MIGRATION.md. Remaining entries migrate one at a
			// time, then this whole sdk/v2 provider goes away.
			"weft_deployment":  resourceDeployment(),
			"weft_endpoint":    resourceEndpoint(),
			"weft_image":       resourceImage(),
			"weft_image_patch": resourceImagePatch(),
			"weft_images":      resourceImages(),
			"weft_keypair":     resourceKeypair(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"weft_config": dataSourceConfig(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

// weftClient is stored in the provider meta.
type weftClient struct {
	conn *grpc.ClientConn
	vms  weftv1.WeftAgentClient
}

func configureProvider(_ context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	sshKey := d.Get("ssh_key").(string)

	var conn *grpc.ClientConn
	var err error

	if sshKey != "" {
		sshSocket := d.Get("ssh_socket").(string)
		if sshSocket == "" {
			sshSocket = defaultSSHSocket()
		}
		var sshOpt grpc.DialOption
		sshOpt, err = sshtransport.DialOption("unix:"+sshSocket, sshKey, "")
		if err != nil {
			return nil, diag.Errorf("ssh dial option: %v", err)
		}
		conn, err = grpc.NewClient("passthrough:///weft", sshOpt, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		socketAddr := d.Get("socket").(string)
		conn, err = grpc.NewClient(socketAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if err != nil {
		return nil, diag.Errorf("cannot connect to weft: %v", err)
	}

	return &weftClient{
		conn: conn,
		vms:  weftv1.NewWeftAgentClient(conn),
	}, nil
}

func defaultSocket() string {
	home, _ := os.UserHomeDir()
	return home + "/.weft/weft.sock"
}

func defaultSSHSocket() string {
	home, _ := os.UserHomeDir()
	return home + "/.weft/weft-ssh.sock"
}
