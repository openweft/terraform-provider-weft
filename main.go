// Command terraform-provider-weft serves the framework-based provider.
//
// Earlier in the migration this file ran a muxed protocol-6 server that
// combined the sdk/v2 half (legacy resources) with the framework half.
// Now that every resource + the lone data source live on the framework
// side (see FRAMEWORK_MIGRATION.md), the mux is gone and we serve the
// framework provider directly.

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/openweft/terraform-provider-weft/internal/provider"
)

// providerAddress is what `terraform init` resolves us to. Must match the
// registry path operators see in their required_providers block.
const providerAddress = "registry.terraform.io/openweft/weft"

// version is overridden at build time by goreleaser via -ldflags
// "-X main.version=v0.2.0".
var version = "0.2.0-dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Run as a debug server for delve attach (set in IDE debug configs)")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: providerAddress,
		Debug:   debug,
	}
	if err := providerserver.Serve(context.Background(), provider.NewFrameworkProvider(version), opts); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
