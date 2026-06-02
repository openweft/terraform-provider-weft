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
	"log"
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

// Close releases the underlying gRPC connection. Callers (provider
// shutdown, tests) should invoke it once they're done with the client to
// avoid leaking a file descriptor per provider instantiation.
func (c *weftClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// defaultSocket returns the operator-visible default for the `socket`
// provider attribute. Matches weft-agent's hard-coded default ($HOME/
// .weft/weft.sock) so a default-everything provider block just works
// when the agent runs locally.
func defaultSocket() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[WARN] weft provider: cannot determine home directory for default socket: %v; set the `socket` attribute or WEFT_SOCKET explicitly", err)
		return ""
	}
	return home + "/.weft/weft.sock"
}

// defaultSSHSocket is the same idea for the SSH transport — operators
// who set ssh_key but leave ssh_socket blank get this path. Mirrors
// where `weft agent` exposes its SSH-fronted socket by default.
func defaultSSHSocket() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[WARN] weft provider: cannot determine home directory for default SSH socket: %v; set the `ssh_socket` attribute or WEFT_SSH_SOCKET explicitly", err)
		return ""
	}
	return home + "/.weft/weft-ssh.sock"
}
