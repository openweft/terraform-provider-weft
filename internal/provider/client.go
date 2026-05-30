// client.go — the shared weftClient (gRPC connection + WeftAgentClient
// wrapper) and the unix-socket default-path helpers.
//
// Both used to live in the sdk/v2 provider.go before the framework
// migration completed. Now that the sdk/v2 provider has been removed
// entirely (see FRAMEWORK_MIGRATION.md), they sit on their own so future
// resources don't have to dig through the framework provider's file to
// find the client type they configure against.

package provider

import (
	"os"

	weftv1 "github.com/openweft/weft-proto"
	"google.golang.org/grpc"
)

// weftClient is what every resource's Configure() casts its ProviderData
// to. Holds the gRPC client connection and the typed WeftAgent service
// stub. Shared across resources within one provider invocation.
type weftClient struct {
	conn *grpc.ClientConn
	vms  weftv1.WeftAgentClient
}

// defaultSocket returns the operator-visible default for the `socket`
// provider attribute. Matches weft-agent's hard-coded default ($HOME/
// .weft/weft.sock) so a default-everything provider block just works
// when the agent runs locally.
func defaultSocket() string {
	home, _ := os.UserHomeDir()
	return home + "/.weft/weft.sock"
}

// defaultSSHSocket is the same idea for the SSH transport — operators
// who set ssh_key but leave ssh_socket blank get this path. Mirrors
// where `weft agent` exposes its SSH-fronted socket by default.
func defaultSSHSocket() string {
	home, _ := os.UserHomeDir()
	return home + "/.weft/weft-ssh.sock"
}
