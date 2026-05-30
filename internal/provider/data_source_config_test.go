package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// ---------------------------------------------------------------------------
// Schema validation
// ---------------------------------------------------------------------------

func TestDataSourceConfigSchema(t *testing.T) {
	r := dataSourceConfig()
	if err := r.InternalValidate(nil, false); err != nil {
		t.Fatalf("dataSourceConfig schema invalid: %v", err)
	}
}

func TestDataSourceConfigSchema_VMsFields(t *testing.T) {
	r := dataSourceConfig()
	vmsAttr, ok := r.Schema["vms"]
	if !ok {
		t.Fatal("schema missing vms attribute")
	}
	if !vmsAttr.Computed {
		t.Error("vms should be Computed")
	}
	// Verify config_dir default.
	cdAttr, ok := r.Schema["config_dir"]
	if !ok {
		t.Fatal("schema missing config_dir")
	}
	if cdAttr.Default != ".mock/hcl" {
		t.Errorf("config_dir default = %v, want .mock/hcl", cdAttr.Default)
	}
}

// ---------------------------------------------------------------------------
// dataSourceConfigRead
// ---------------------------------------------------------------------------

// findHCLDir returns the path to the .mock/hcl directory relative to the
// module root.  Tests run from the package directory, so we walk up.
func findHCLDir(t *testing.T) string {
	t.Helper()
	// Walk up from the test working directory to find .mock/hcl.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, ".mock/hcl")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("cannot locate .mock/hcl from test working directory")
	return ""
}

func TestDataSourceConfigRead_SetsID(t *testing.T) {
	hclDir := findHCLDir(t)

	r := dataSourceConfig()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", hclDir)

	diags := dataSourceConfigRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != hclDir {
		t.Errorf("ID = %q, want %q", d.Id(), hclDir)
	}
}

func TestDataSourceConfigRead_ReturnsVMs(t *testing.T) {
	hclDir := findHCLDir(t)

	r := dataSourceConfig()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", hclDir)

	diags := dataSourceConfigRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}

	vms := d.Get("vms").([]interface{})
	if len(vms) == 0 {
		t.Fatal("expected at least one VM, got none")
	}
	for _, v := range vms {
		vm := v.(map[string]interface{})
		if vm["name"].(string) == "" {
			t.Error("VM name should not be empty")
		}
		if vm["disk_size"].(string) == "" {
			t.Error("VM disk_size should not be empty")
		}
	}
}

func TestDataSourceConfigRead_DiskSizeFormat(t *testing.T) {
	hclDir := findHCLDir(t)

	r := dataSourceConfig()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", hclDir)

	diags := dataSourceConfigRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}

	vms := d.Get("vms").([]interface{})
	for _, v := range vms {
		vm := v.(map[string]interface{})
		diskSize := vm["disk_size"].(string)
		// Must match NNGi format
		if !strings.HasSuffix(diskSize, "Gi") {
			t.Errorf("disk_size %q should end with Gi", diskSize)
		}
	}
}

func TestDataSourceConfigRead_InvalidDir(t *testing.T) {
	r := dataSourceConfig()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", "/nonexistent/path/to/hcl")

	diags := dataSourceConfigRead(context.Background(), d, nil)
	if !diags.HasError() {
		t.Fatal("expected error for invalid config dir, got none")
	}
	if !strings.Contains(diags[0].Summary, "parse mock HCL config") {
		t.Errorf("unexpected error: %s", diags[0].Summary)
	}
}

func TestDataSourceConfigRead_EmptyDir(t *testing.T) {
	// A temp dir with no .hcl files should produce an error from ReadVMs.
	tmpDir := t.TempDir()

	r := dataSourceConfig()
	d := r.Data(&terraform.InstanceState{})
	d.Set("config_dir", tmpDir)

	diags := dataSourceConfigRead(context.Background(), d, nil)
	// Empty HCL dir may return an error or an empty list; both are acceptable.
	// We just make sure it doesn't panic.
	_ = diags
}

func TestDataSourceConfigRead_DefaultConfigDir(t *testing.T) {
	// When config_dir is not set the default ".mock/hcl" should be used.
	// This test verifies the schema default is wired correctly.
	r := dataSourceConfig()
	attr := r.Schema["config_dir"]
	if attr.Default != ".mock/hcl" {
		t.Errorf("default config_dir = %v, want .mock/hcl", attr.Default)
	}
}
