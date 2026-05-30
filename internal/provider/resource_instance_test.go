package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"google.golang.org/grpc"
)

// ---------------------------------------------------------------------------
// parseSizeGiB
// ---------------------------------------------------------------------------

func TestParseSizeGiB(t *testing.T) {
	tests := []struct {
		raw  string
		want int
		ok   bool
	}{
		{"20Gi", 20, true},
		{"100Gi", 100, true},
		{"1Gi", 1, true},
		{"20G", 20, true},
		{"100G", 100, true},
		{"2Ti", 2048, true},
		{"1T", 1024, true},
		{"2T", 2048, true},
		{"1024Mi", 1, true},
		{"2048Mi", 2, true},
		{"1024M", 1, true},
		// quoted form (mock HCL uses "20Gi" with quotes in strings)
		{`"20Gi"`, 20, true},
		// invalid
		{"", 0, false},
		{"abc", 0, false},
		{"20Ki", 0, false},
		{"20", 0, false},
		{"-5Gi", 0, false},
		{"0Gi", 0, false},
		// sub-GiB MiB values round to zero
		{"512Mi", 0, false},
	}
	for _, tc := range tests {
		got, ok := parseSizeGiB(tc.raw)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseSizeGiB(%q) = (%d, %v), want (%d, %v)",
				tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

// ---------------------------------------------------------------------------
// expandHome
// ---------------------------------------------------------------------------

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	tests := []struct {
		in   string
		want string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"~/a/b/c", filepath.Join(home, "a/b/c")},
		{"/abs/path", "/abs/path"},
		{"relative/path", "relative/path"},
		{"", ""},
		// bare tilde is not expanded (no trailing slash)
		{"~", "~"},
	}
	for _, tc := range tests {
		got := expandHome(tc.in)
		if got != tc.want {
			t.Errorf("expandHome(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Schema validation
// ---------------------------------------------------------------------------

func TestResourceInstanceSchema(t *testing.T) {
	r := resourceInstance()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceInstance schema invalid: %v", err)
	}
}

func TestResourceInstanceSchema_RequiredFields(t *testing.T) {
	r := resourceInstance()
	s := r.Schema

	requiredForceNew := []string{"name", "disk"}
	for _, field := range requiredForceNew {
		attr, ok := s[field]
		if !ok {
			t.Errorf("schema missing field %q", field)
			continue
		}
		if !attr.ForceNew {
			t.Errorf("field %q should be ForceNew", field)
		}
	}

	// cpu and mem must have defaults
	for _, field := range []string{"cpu", "mem"} {
		attr, ok := s[field]
		if !ok {
			t.Errorf("schema missing field %q", field)
			continue
		}
		if attr.Default == nil {
			t.Errorf("field %q should have a default", field)
		}
	}

	// ip and state must be computed
	for _, field := range []string{"ip", "state"} {
		attr, ok := s[field]
		if !ok {
			t.Errorf("schema missing computed field %q", field)
			continue
		}
		if !attr.Computed {
			t.Errorf("field %q should be Computed", field)
		}
	}
}

// ---------------------------------------------------------------------------
// resourceInstanceCreate
// ---------------------------------------------------------------------------

func TestResourceInstanceCreate_InvalidDiskSize(t *testing.T) {
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{})
	d.Set("name", "vm1")
	d.Set("disk", []interface{}{
		map[string]interface{}{"from": "docker.io/lib/debian:13", "size": "invalid"},
	})

	diags := resourceInstanceCreate(context.Background(), d, newTestClient(&mockWeftClient{}))
	if !diags.HasError() {
		t.Fatal("expected error for invalid disk size, got none")
	}
	if !strings.Contains(diags[0].Summary, "invalid disk size") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceInstanceCreate_NoDisk(t *testing.T) {
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{})
	d.Set("name", "vm1")
	// no disk block set

	diags := resourceInstanceCreate(context.Background(), d, newTestClient(&mockWeftClient{}))
	if !diags.HasError() {
		t.Fatal("expected error when disk block is missing, got none")
	}
}

func TestResourceInstanceCreate_ProvisionVMError(t *testing.T) {
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{})
	d.Set("name", "vm1")

	d.Set("disk", []interface{}{
		map[string]interface{}{"from": "docker.io/lib/debian:13", "size": "20Gi"},
	})

	mock := &mockWeftClient{
		provisionVMFn: func(_ context.Context, _ *weftv1.ProvisionVMRequest, _ ...grpc.CallOption) (*weftv1.ProvisionVMResponse, error) {
			return nil, fmt.Errorf("provisioning failed")
		},
	}
	diags := resourceInstanceCreate(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error from ProvisionVM, got none")
	}
	if !strings.Contains(diags[0].Summary, "ProvisionVM failed") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceInstanceCreate_Success(t *testing.T) {
	const vmName = "test-vm"

	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{})
	d.Set("name", vmName)
	d.Set("cpu", 2)
	d.Set("mem", 2) // 2 GiB → 2048 MiB
	d.Set("disk", []interface{}{
		map[string]interface{}{"from": "docker.io/lib/debian:13", "size": "20Gi"},
	})

	mock := &mockWeftClient{
		provisionVMFn: func(_ context.Context, req *weftv1.ProvisionVMRequest, _ ...grpc.CallOption) (*weftv1.ProvisionVMResponse, error) {
			if req.Name != vmName {
				return nil, fmt.Errorf("unexpected name %q", req.Name)
			}
			if req.DiskGb != 20 {
				return nil, fmt.Errorf("unexpected DiskGb %d", req.DiskGb)
			}
			if req.MemMb != 2*1024 {
				return nil, fmt.Errorf("unexpected MemMb %d", req.MemMb)
			}
			return &weftv1.ProvisionVMResponse{}, nil
		},
	}

	diags := resourceInstanceCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != vmName {
		t.Errorf("expected ID %q, got %q", vmName, d.Id())
	}
	// ip/state are populated by Read, not Create.
}

func TestResourceInstanceCreate_MemConversion(t *testing.T) {
	var capturedReq *weftv1.ProvisionVMRequest

	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{})
	d.Set("name", "mem-test")
	d.Set("mem", 4) // 4 GiB → should become 4096 MiB
	d.Set("disk", []interface{}{
		map[string]interface{}{"from": "img", "size": "20Gi"},
	})

	mock := &mockWeftClient{
		provisionVMFn: func(_ context.Context, req *weftv1.ProvisionVMRequest, _ ...grpc.CallOption) (*weftv1.ProvisionVMResponse, error) {
			capturedReq = req
			return &weftv1.ProvisionVMResponse{}, nil
		},
	}
	resourceInstanceCreate(context.Background(), d, newTestClient(mock))

	if capturedReq == nil {
		t.Fatal("ProvisionVM was not called")
	}
	if capturedReq.MemMb != 4096 {
		t.Errorf("MemMb = %d, want 4096", capturedReq.MemMb)
	}
}

func TestResourceInstanceCreate_SSHPublicKey(t *testing.T) {
	// Write a temporary keypair.
	dir := t.TempDir()
	privKey := filepath.Join(dir, "id_rsa")
	pubKey := privKey + ".pub"
	if err := os.WriteFile(privKey, []byte("private-key"), 0600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(pubKey, []byte("ssh-rsa AAAA test@host"), 0644); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	var capturedPubKey string
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{})
	d.Set("name", "ssh-test")
	d.Set("disk", []interface{}{
		map[string]interface{}{"from": "img", "size": "20Gi"},
	})
	d.Set("ssh", []interface{}{
		map[string]interface{}{
			"user":         "debian",
			"keypair_path": privKey,
		},
	})

	mock := &mockWeftClient{
		provisionVMFn: func(_ context.Context, req *weftv1.ProvisionVMRequest, _ ...grpc.CallOption) (*weftv1.ProvisionVMResponse, error) {
			capturedPubKey = req.SshPubKey
			return &weftv1.ProvisionVMResponse{}, nil
		},
	}
	diags := resourceInstanceCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if capturedPubKey != "ssh-rsa AAAA test@host" {
		t.Errorf("SshPubKey = %q, want %q", capturedPubKey, "ssh-rsa AAAA test@host")
	}
}

func TestResourceInstanceCreate_MissingSSHPublicKey(t *testing.T) {
	dir := t.TempDir()
	privKey := filepath.Join(dir, "id_rsa")
	// Only private key is written; no .pub file.
	if err := os.WriteFile(privKey, []byte("private-key"), 0600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{})
	d.Set("name", "ssh-nopub")
	d.Set("disk", []interface{}{
		map[string]interface{}{"from": "img", "size": "20Gi"},
	})
	d.Set("ssh", []interface{}{
		map[string]interface{}{
			"user":         "debian",
			"keypair_path": privKey,
		},
	})

	diags := resourceInstanceCreate(context.Background(), d, newTestClient(&mockWeftClient{}))
	if !diags.HasError() {
		t.Fatal("expected error for missing .pub file, got none")
	}
	if !strings.Contains(diags[0].Summary, "read SSH public key") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

// ---------------------------------------------------------------------------
// resourceInstanceRead
// ---------------------------------------------------------------------------

func TestResourceInstanceRead_VMNotFound(t *testing.T) {
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{ID: "missing-vm"})
	d.Set("name", "missing-vm")

	mock := &mockWeftClient{
		vmStatusFn: func(_ context.Context, _ *weftv1.VMStatusRequest, _ ...grpc.CallOption) (*weftv1.VMStatusResponse, error) {
			return nil, fmt.Errorf("VM not found")
		},
	}
	diags := resourceInstanceRead(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Errorf("expected no error when VM not found, got: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after VM not found, got %q", d.Id())
	}
}

func TestResourceInstanceRead_PopulatesComputedFields(t *testing.T) {
	const vmIP = "10.0.0.5"
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{ID: "my-vm"})
	d.Set("name", "my-vm")

	mock := &mockWeftClient{
		vmStatusFn: func(_ context.Context, _ *weftv1.VMStatusRequest, _ ...grpc.CallOption) (*weftv1.VMStatusResponse, error) {
			return &weftv1.VMStatusResponse{
				Vm: &weftv1.VMInfo{
					Name:  "my-vm",
					State: weftv1.VMState_VM_STATE_RUNNING,
					Ip:    vmIP,
				},
			}, nil
		},
	}
	diags := resourceInstanceRead(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if got := d.Get("ip").(string); got != vmIP {
		t.Errorf("ip = %q, want %q", got, vmIP)
	}
	if got := d.Get("state").(string); got != weftv1.VMState_VM_STATE_RUNNING.String() {
		t.Errorf("state = %q, want %q", got, weftv1.VMState_VM_STATE_RUNNING.String())
	}
}

// ---------------------------------------------------------------------------
// resourceInstanceDelete
// ---------------------------------------------------------------------------

func TestResourceInstanceDelete_Success(t *testing.T) {
	called := false
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{ID: "del-vm"})
	d.Set("name", "del-vm")

	mock := &mockWeftClient{
		deprovisionVMFn: func(_ context.Context, req *weftv1.DeprovisionVMRequest, _ ...grpc.CallOption) (*weftv1.DeprovisionVMResponse, error) {
			called = true
			if req.Name != "del-vm" {
				return nil, fmt.Errorf("unexpected name %q", req.Name)
			}
			return &weftv1.DeprovisionVMResponse{}, nil
		},
	}
	diags := resourceInstanceDelete(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if !called {
		t.Error("DeprovisionVM was not called")
	}
	if d.Id() != "" {
		t.Errorf("expected empty ID after delete, got %q", d.Id())
	}
}

func TestResourceInstanceDelete_Error(t *testing.T) {
	res := resourceInstance()
	d := res.Data(&terraform.InstanceState{ID: "del-vm"})
	d.Set("name", "del-vm")

	mock := &mockWeftClient{
		deprovisionVMFn: func(_ context.Context, _ *weftv1.DeprovisionVMRequest, _ ...grpc.CallOption) (*weftv1.DeprovisionVMResponse, error) {
			return nil, fmt.Errorf("deprovision failed")
		},
	}
	diags := resourceInstanceDelete(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error from DeprovisionVM, got none")
	}
	if !strings.Contains(diags[0].Summary, "DeprovisionVM failed") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}
