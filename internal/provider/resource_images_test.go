package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"google.golang.org/grpc"
)

// ---------------------------------------------------------------------------
// Schema validation
// ---------------------------------------------------------------------------

func TestResourceImagesSchema(t *testing.T) {
	r := resourceImages()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceImages schema invalid: %v", err)
	}
}

func TestResourceImagesSchema_Defaults(t *testing.T) {
	s := resourceImages().Schema

	configDir, ok := s["config_dir"]
	if !ok {
		t.Fatal("schema missing config_dir")
	}
	if configDir.Default != ".mock/hcl" {
		t.Errorf("config_dir default = %v, want .mock/hcl", configDir.Default)
	}
	if !configDir.ForceNew {
		t.Error("config_dir should be ForceNew")
	}

	parallel, ok := s["parallel"]
	if !ok {
		t.Fatal("schema missing parallel")
	}
	if parallel.Default != 4 {
		t.Errorf("parallel default = %v, want 4", parallel.Default)
	}
	if !parallel.ForceNew {
		t.Error("parallel should be ForceNew")
	}

	pulled, ok := s["pulled"]
	if !ok {
		t.Fatal("schema missing pulled")
	}
	if !pulled.Computed {
		t.Error("pulled should be Computed")
	}
}

// ---------------------------------------------------------------------------
// resourceImagesCreate
// ---------------------------------------------------------------------------

func TestResourceImagesCreate_PullImagesError(t *testing.T) {
	r := resourceImages()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", ".mock/hcl")
	d.Set("parallel", 4)

	mock := &mockWeftClient{
		pullImagesFn: func(_ context.Context, _ *weftv1.PullImagesRequest, _ ...grpc.CallOption) (*weftv1.PullImagesResponse, error) {
			return nil, fmt.Errorf("pull failed: network error")
		},
	}
	diags := resourceImagesCreate(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error from PullImages, got none")
	}
	if !strings.Contains(diags[0].Summary, "PullImages failed") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceImagesCreate_CallsCorrectParams(t *testing.T) {
	const cfgDir = ".mock/hcl"
	const par = 2
	var capturedReq *weftv1.PullImagesRequest

	r := resourceImages()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", cfgDir)
	d.Set("parallel", par)

	mock := &mockWeftClient{
		pullImagesFn: func(_ context.Context, req *weftv1.PullImagesRequest, _ ...grpc.CallOption) (*weftv1.PullImagesResponse, error) {
			capturedReq = req
			return &weftv1.PullImagesResponse{}, nil
		},
	}
	// ReadVMs may fail because .mock/hcl is not necessarily accessible at
	// test time from the provider module root; errors there are non-fatal.
	resourceImagesCreate(context.Background(), d, newTestClient(mock))

	if capturedReq == nil {
		t.Fatal("PullImages was not called")
	}
	if capturedReq.ConfigDir != cfgDir {
		t.Errorf("ConfigDir = %q, want %q", capturedReq.ConfigDir, cfgDir)
	}
	if capturedReq.Parallel != int32(par) {
		t.Errorf("Parallel = %d, want %d", capturedReq.Parallel, par)
	}
}

func TestResourceImagesCreate_SetsID(t *testing.T) {
	r := resourceImages()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", "some/dir")
	d.Set("parallel", 3)

	diags := resourceImagesCreate(context.Background(), d, newTestClient(&mockWeftClient{}))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	wantID := "some/dir|3"
	if d.Id() != wantID {
		t.Errorf("ID = %q, want %q", d.Id(), wantID)
	}
}

// ---------------------------------------------------------------------------
// resourceImagesRead
// ---------------------------------------------------------------------------

func TestResourceImagesRead_RestoresConfigDir(t *testing.T) {
	r := resourceImages()
	d := r.Data(&terraform.InstanceState{ID: "my/config/dir|4"})

	diags := resourceImagesRead(context.Background(), d, newTestClient(&mockWeftClient{}))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if got := d.Get("config_dir").(string); got != "my/config/dir" {
		t.Errorf("config_dir = %q, want %q", got, "my/config/dir")
	}
}

func TestResourceImagesRead_EmptyID(t *testing.T) {
	r := resourceImages()
	d := r.Data(&terraform.InstanceState{})

	diags := resourceImagesRead(context.Background(), d, newTestClient(&mockWeftClient{}))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceImagesRead_IDWithPipe(t *testing.T) {
	// ID format: "<config_dir>|<parallel>" — config_dir itself may contain |.
	// Only the first | is used as delimiter (SplitN limit=2).
	r := resourceImages()
	d := r.Data(&terraform.InstanceState{ID: "path/with|pipe|8"})

	diags := resourceImagesRead(context.Background(), d, newTestClient(&mockWeftClient{}))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if got := d.Get("config_dir").(string); got != "path/with" {
		t.Errorf("config_dir = %q, want %q", got, "path/with")
	}
}

// ---------------------------------------------------------------------------
// resourceImagesDelete
// ---------------------------------------------------------------------------

func TestResourceImagesDelete(t *testing.T) {
	r := resourceImages()
	d := r.Data(&terraform.InstanceState{ID: "some/dir|4"})

	diags := resourceImagesDelete(context.Background(), d, newTestClient(&mockWeftClient{}))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after delete, got %q", d.Id())
	}
}
