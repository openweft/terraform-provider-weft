//go:build acceptance

// acceptance_test.go — harness for `TestAcc*` tests.
//
// Acceptance tests exercise the provider against a REAL weft daemon: each
// test stands up an HCL config, invokes terraform's apply/plan/destroy
// against it through `helper/resource`, and asserts on the resulting state.
// They're gated by the `TF_ACC=1` env var so unit-test runs (`go test`)
// don't accidentally try to dial a daemon that isn't there.
//
// Operator workflow to run them:
//
//	# 1. Start a weft agent (or point at an existing one).
//	weft agent --state-dir /tmp/acc-weft &
//	# 2. Run the acceptance suite.
//	TF_ACC=1 WEFT_SOCKET=unix:///tmp/acc-weft/weft.sock \
//	    go test -timeout 30m -count=1 -run TestAcc ./internal/provider/...
//
// Conventions:
//   - One `TestAcc<Resource>_<scenario>` per scenario, each scenario one
//     TestCase with one or more TestSteps.
//   - testAccPreCheck handles env-var validation so a missing socket fails
//     fast with a clear message instead of a confusing dial error.
//   - testAccProtoV6ProviderFactories wires the muxed provider into the
//     test harness — keep this in sync with main.go's mux setup.

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

// testAccProtoV6ProviderFactories returns the muxed-provider factory the
// helper/resource test harness consumes. Mirrors main.go's mux assembly;
// any change to main.go's provider composition should be mirrored here.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"weft": func() (tfprotov6.ProviderServer, error) {
		ctx := context.Background()
		upgradedSDKv2, err := tf5to6server.UpgradeServer(ctx, New().GRPCProvider)
		if err != nil {
			return nil, err
		}
		fwSrv := providerserver.NewProtocol6(NewFrameworkProvider("acc-test")())
		mux, err := tf6muxserver.NewMuxServer(ctx,
			func() tfprotov6.ProviderServer { return upgradedSDKv2 },
			fwSrv,
		)
		if err != nil {
			return nil, err
		}
		return mux.ProviderServer(), nil
	},
}

// testAccPreCheck enforces the env vars an acceptance test needs. Called
// at the start of every TestCase via its PreCheck hook so missing
// configuration fails with a single clear message instead of a kilobyte
// of terraform internals.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}
	if os.Getenv("WEFT_SOCKET") == "" {
		t.Fatal("WEFT_SOCKET must be set (e.g. unix:///tmp/acc-weft/weft.sock)")
	}
}
