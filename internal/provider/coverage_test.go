package provider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
)

// ---------------------------------------------------------------------------
// provider.go: configureProvider, defaultSSHSocket
// ---------------------------------------------------------------------------

func TestDefaultSSHSocket(t *testing.T) {
	s := defaultSSHSocket()
	if !strings.HasSuffix(s, "/.weft/weft-ssh.sock") {
		t.Errorf("defaultSSHSocket() = %q, want suffix /.weft/weft-ssh.sock", s)
	}
	if !strings.HasPrefix(s, "/") {
		t.Errorf("defaultSSHSocket() = %q, expected absolute path", s)
	}
}

func TestConfigureProvider_PlainSocket(t *testing.T) {
	p := New()
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"socket": "unix:///tmp/weft.sock",
	})
	meta, diags := configureProvider(context.Background(), d)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if meta == nil {
		t.Fatal("expected non-nil client")
	}
	if _, ok := meta.(*weftClient); !ok {
		t.Errorf("expected *weftClient, got %T", meta)
	}
}

func TestConfigureProvider_SSHKeyError(t *testing.T) {
	// Provide an ssh_key path to a missing file so sshtransport.DialOption fails.
	p := New()
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"socket":     "unix:///tmp/weft.sock",
		"ssh_socket": "/tmp/weft-ssh.sock",
		"ssh_key":    "/nonexistent/path/to/missing-key",
	})
	_, diags := configureProvider(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("expected error from missing SSH key, got none")
	}
	if !strings.Contains(diags[0].Summary, "ssh dial option") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestConfigureProvider_SSHKeyDefaultSocket(t *testing.T) {
	// ssh_socket is empty so the code path that calls defaultSSHSocket() runs.
	// We still expect an error because the key file is missing, but the branch
	// is exercised.
	p := New()
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"socket":  "unix:///tmp/weft.sock",
		"ssh_key": "/nonexistent/path/to/missing-key",
		// ssh_socket left empty → defaultSSHSocket() branch
	})
	_, diags := configureProvider(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("expected error from missing SSH key, got none")
	}
}

// writeED25519PrivateKey writes a freshly generated ed25519 private key in
// OpenSSH PEM format so that sshtransport.DialOption's authMethods accepts it.
// grpc.NewClient is lazy — it does not connect — so this lets us cover the
// success branch in configureProvider without a real SSH server.
func writeED25519PrivateKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519 keygen: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("ssh marshal: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return path
}

func TestConfigureProvider_PlainSocketGRPCError(t *testing.T) {
	// A socket address containing a control character makes grpc.NewClient
	// fail synchronously via URL parsing.
	p := New()
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"socket": "\x00bad",
	})
	_, diags := configureProvider(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("expected error from grpc.NewClient with malformed address, got none")
	}
	if !strings.Contains(diags[0].Summary, "cannot connect to weft") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestConfigureProvider_SSHTransportSuccess(t *testing.T) {
	keyPath := writeED25519PrivateKey(t)

	p := New()
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"socket":     "unix:///tmp/weft.sock",
		"ssh_socket": "/tmp/weft-ssh.sock",
		"ssh_key":    keyPath,
	})
	meta, diags := configureProvider(context.Background(), d)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if meta == nil {
		t.Fatal("expected non-nil client")
	}
	c, ok := meta.(*weftClient)
	if !ok {
		t.Fatalf("expected *weftClient, got %T", meta)
	}
	// Close the connection to avoid leaking resources between tests.
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// ---------------------------------------------------------------------------
// resource_deployment.go: Read, Delete
// ---------------------------------------------------------------------------

func TestResourceDeploymentRead(t *testing.T) {
	res := resourceDeployment()
	d := res.Data(&terraform.InstanceState{ID: "M19B3D62C"})
	diags := resourceDeploymentRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceDeploymentDelete(t *testing.T) {
	res := resourceDeployment()
	d := res.Data(&terraform.InstanceState{ID: "M19B3D62C"})
	diags := resourceDeploymentDelete(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after delete, got %q", d.Id())
	}
}

// ---------------------------------------------------------------------------
// resource_endpoint.go: Read, Delete
// ---------------------------------------------------------------------------

func TestResourceEndpointRead(t *testing.T) {
	res := resourceEndpoint()
	d := res.Data(&terraform.InstanceState{ID: "https://example.com/"})
	diags := resourceEndpointRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceEndpointDelete(t *testing.T) {
	res := resourceEndpoint()
	d := res.Data(&terraform.InstanceState{ID: "https://example.com/"})
	diags := resourceEndpointDelete(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after delete, got %q", d.Id())
	}
}

// ---------------------------------------------------------------------------
// resource_image.go: Read, Delete
// ---------------------------------------------------------------------------

func TestResourceImageRead(t *testing.T) {
	res := resourceImage()
	d := res.Data(&terraform.InstanceState{ID: "https://example.com/image.raw"})
	diags := resourceImageRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceImageDelete(t *testing.T) {
	res := resourceImage()
	d := res.Data(&terraform.InstanceState{ID: "https://example.com/image.raw"})
	diags := resourceImageDelete(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after delete, got %q", d.Id())
	}
}

// ---------------------------------------------------------------------------
// resource_image_patch.go: Read, Delete, MissingPatch, NoOp, ListImagesError
// ---------------------------------------------------------------------------

func TestResourceImagePatchRead(t *testing.T) {
	res := resourceImagePatch()
	d := res.Data(&terraform.InstanceState{ID: "imagepatch-1"})
	diags := resourceImagePatchRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceImagePatchDelete(t *testing.T) {
	res := resourceImagePatch()
	d := res.Data(&terraform.InstanceState{ID: "imagepatch-1"})
	diags := resourceImagePatchDelete(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after delete, got %q", d.Id())
	}
}

func TestResourceImagePatchCreate_MissingPatch(t *testing.T) {
	res := resourceImagePatch()
	d := res.Data(&terraform.InstanceState{})
	// patch block intentionally left empty
	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(&mockWeftClient{}))
	if !diags.HasError() {
		t.Fatal("expected error when patch block is missing, got none")
	}
	if !strings.Contains(diags[0].Summary, "patch block required") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceImagePatchCreate_NoOp(t *testing.T) {
	// All three ops lists empty → no RPC calls, but ID is still set.
	patchImageCalled := false
	mock := &mockWeftClient{
		patchImageFn: func(_ context.Context, _ *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			patchImageCalled = true
			return &weftv1.PatchImageResponse{}, nil
		},
	}

	res := resourceImagePatch()
	d := res.Data(&terraform.InstanceState{})
	d.Set("patch", []interface{}{map[string]interface{}{
		"add": []interface{}{},
		"del": []interface{}{},
		"mod": []interface{}{},
	}})

	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if patchImageCalled {
		t.Error("PatchImage should not be called when all ops are empty")
	}
	if d.Id() == "" {
		t.Error("ID should be set even on no-op")
	}
}

func TestResourceImagePatchCreate_ListImagesError(t *testing.T) {
	mock := &mockWeftClient{
		listImagesFn: func(_ context.Context, _ *weftv1.ListImagesRequest, _ ...grpc.CallOption) (*weftv1.ListImagesResponse, error) {
			return nil, fmt.Errorf("list images network error")
		},
	}
	res := resourceImagePatch()
	d := res.Data(&terraform.InstanceState{})
	// No images specified → ListImages branch
	d.Set("patch", []interface{}{map[string]interface{}{
		"add": []interface{}{map[string]interface{}{"content": "x", "dst": "/a", "trigger": ""}},
		"del": []interface{}{},
		"mod": []interface{}{},
	}})

	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error from ListImages, got none")
	}
	if !strings.Contains(diags[0].Summary, "ListImages failed") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceImagePatchCreate_NoImagesPatchError(t *testing.T) {
	// When no images are specified, ListImages succeeds, but PatchImage fails
	// on the resulting list.
	mock := &mockWeftClient{
		listImagesFn: func(_ context.Context, _ *weftv1.ListImagesRequest, _ ...grpc.CallOption) (*weftv1.ListImagesResponse, error) {
			return &weftv1.ListImagesResponse{Images: []*weftv1.ImageInfo{{Url: "u1"}}}, nil
		},
		patchImageFn: func(_ context.Context, _ *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			return nil, fmt.Errorf("patch failed")
		},
	}
	res := resourceImagePatch()
	d := res.Data(&terraform.InstanceState{})
	d.Set("patch", []interface{}{map[string]interface{}{
		"add": []interface{}{map[string]interface{}{"content": "x", "dst": "/a", "trigger": ""}},
		"del": []interface{}{},
		"mod": []interface{}{},
	}})

	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error from PatchImage, got none")
	}
	if !strings.Contains(diags[0].Summary, "PatchImage failed for u1") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceImagePatchCreate_WithDelAndMod(t *testing.T) {
	// Cover the del and mod loops which are exercised only when those lists are non-empty.
	var got *weftv1.PatchImageRequest
	mock := &mockWeftClient{
		patchImageFn: func(_ context.Context, req *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			got = req
			return &weftv1.PatchImageResponse{}, nil
		},
	}
	res := resourceImagePatch()
	d := res.Data(&terraform.InstanceState{})
	d.Set("images", []interface{}{"https://example.com/img"})
	d.Set("patch", []interface{}{map[string]interface{}{
		"add": []interface{}{},
		"del": []interface{}{map[string]interface{}{"dst": "/etc/old"}},
		"mod": []interface{}{map[string]interface{}{"dst": "/etc/hosts", "old": "a", "new": "b"}},
	}})

	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if got == nil {
		t.Fatal("PatchImage was not called")
	}
	if len(got.DeleteOps) != 1 || got.DeleteOps[0].Dst != "/etc/old" {
		t.Errorf("DeleteOps not propagated: %+v", got.DeleteOps)
	}
	if len(got.ModOps) != 1 || got.ModOps[0].New != "b" {
		t.Errorf("ModOps not propagated: %+v", got.ModOps)
	}
}

// ---------------------------------------------------------------------------
// resource_images.go: Pulled list propagation
// ---------------------------------------------------------------------------

// makeMockHCLDir writes a self-contained HCL config in a temp dir so that
// mockconfig.ReadVMs returns a non-empty row list with an image reference.
// The fixture mirrors the structure used in the openweft/hclconfig testdata.
func makeMockHCLDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	hcl := `version = "1"

mock "test" {
  ssh {
    user    = "ubuntu"
    keypair = keypair.default
  }
}

keypair default {
  file_path = "/tmp/fake-key"
}

vms web {
  count  = 1
  cpu    = 2
  memory = 1024
  disk {
    from = "registry/debian:13"
    size = "20Gi"
  }
  ssh {
    user    = "ubuntu"
    keypair = keypair.default
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "config.hcl"), []byte(hcl), 0644); err != nil {
		t.Fatalf("write hcl: %v", err)
	}
	return dir
}

func TestResourceImagesCreate_PulledList(t *testing.T) {
	dir := makeMockHCLDir(t)

	res := resourceImages()
	d := res.Data(&terraform.InstanceState{})
	d.Set("config_dir", dir)
	d.Set("parallel", 1)

	mock := &mockWeftClient{
		pullImagesFn: func(_ context.Context, _ *weftv1.PullImagesRequest, _ ...grpc.CallOption) (*weftv1.PullImagesResponse, error) {
			return &weftv1.PullImagesResponse{}, nil
		},
	}
	diags := resourceImagesCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	pulled := d.Get("pulled").([]interface{})
	if len(pulled) == 0 {
		t.Error("expected at least one pulled image, got none")
	}
}

// Coverage of resource_instance.go (sdk/v2 path) — superseded by the
// framework migration of weft_instance (see FRAMEWORK_MIGRATION.md);
// instance_resource_test.go covers the framework schema. The pre-existing
// disk-patch end-to-end test asserted the sdk/v2 ProvisionVMRequest
// construction; equivalent coverage on the framework side will land with
// acceptance tests in a follow-up. Leaving the helper-level tests
// (TestExpandHome_NoHome below) since expandHome lives in
// instance_resource.go now.

func TestExpandHome_NoHome(t *testing.T) {
	// When os.UserHomeDir() returns an error, expandHome must return the
	// original tilde path unchanged.
	if runtime.GOOS == "windows" {
		t.Skip("HOME env var not used on windows")
	}
	t.Setenv("HOME", "")
	got := expandHome("~/foo")
	if got != "~/foo" {
		t.Errorf("expandHome with empty HOME = %q, want %q", got, "~/foo")
	}
}

// TestResourceInstanceRead_TransientError was removed alongside the sdk/v2
// resource_instance.go path. Equivalent coverage on the framework side
// belongs in acceptance tests (TF_ACC), tracked in FRAMEWORK_MIGRATION.md.

// ---------------------------------------------------------------------------
// resource_keypair.go: Delete, file_path with tilde, read error after resolved_path set
// ---------------------------------------------------------------------------

func TestResourceKeypairDelete(t *testing.T) {
	res := resourceKeypair()
	d := res.Data(&terraform.InstanceState{ID: "mock"})
	diags := resourceKeypairDelete(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after delete, got %q", d.Id())
	}
}

// ---------------------------------------------------------------------------
// data_source_config.go: full happy path with real HCL fixture
// ---------------------------------------------------------------------------

func TestDataSourceConfigRead_Success(t *testing.T) {
	dir := makeMockHCLDir(t)

	res := dataSourceConfig()
	d := res.Data(&terraform.InstanceState{})
	d.Set("config_dir", dir)

	diags := dataSourceConfigRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != dir {
		t.Errorf("ID = %q, want %q", d.Id(), dir)
	}
	vms := d.Get("vms").([]interface{})
	if len(vms) == 0 {
		t.Fatal("expected at least one VM, got none")
	}
	vm0 := vms[0].(map[string]interface{})
	if vm0["name"].(string) == "" {
		t.Error("vm name should not be empty")
	}
	if vm0["disk_size"].(string) != "20Gi" {
		t.Errorf("disk_size = %q, want 20Gi", vm0["disk_size"])
	}
}

