// Command terraform-provider-weft serves a muxed provider: the sdk/v2
// half (legacy resources still being migrated) speaks protocol 5 upgraded
// to 6 via tf5to6server; the framework half (where new resources land)
// speaks protocol 6 natively. The mux server multiplexes both onto one
// gRPC listener that Terraform sees as a single protocol-6 provider.
//
// See FRAMEWORK_MIGRATION.md for the migration status table.

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	"github.com/openweft/terraform-provider-weft/internal/provider"
)

// providerAddress is what `terraform init` resolves us to. It matches the
// registry path operators see in their required_providers block once the
// provider is published.
const providerAddress = "registry.terraform.io/openweft/weft"

// version is overridden at build time by goreleaser via -ldflags
// "-X main.version=v0.2.0" — leave the placeholder for dev builds.
var version = "0.2.0-dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Run as a debug server for delve attach (set in IDE debug configs)")
	flag.Parse()

	ctx := context.Background()

	// sdk/v2 half: legacy resources still being migrated. tf5to6server
	// upgrades its protocol 5 surface to protocol 6 so the mux server
	// can speak a uniform protocol to Terraform.
	upgradedSDKv2, err := tf5to6server.UpgradeServer(ctx, provider.New().GRPCProvider)
	if err != nil {
		log.Fatalf("upgrade sdk/v2 to protocol 6: %v", err)
	}

	// Framework half: new resources land here. Each migration moves one
	// resource from the sdk/v2 server's ResourcesMap to the framework
	// provider's Resources().
	frameworkServer := providerserver.NewProtocol6(provider.NewFrameworkProvider(version)())

	muxers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer { return upgradedSDKv2 },
		frameworkServer,
	}
	muxServer, err := tf6muxserver.NewMuxServer(ctx, muxers...)
	if err != nil {
		log.Fatalf("build mux server: %v", err)
	}

	serveOpts := []tf6server.ServeOpt{}
	if debug {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}
	if err := tf6server.Serve(providerAddress, muxServer.ProviderServer, serveOpts...); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
