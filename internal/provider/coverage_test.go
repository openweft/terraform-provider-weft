package provider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/crypto/ssh"
)

// ---------------------------------------------------------------------------
// provider.go: configureProvider, defaultSSHSocket
//
// The sdk/v2 provider remains in place (with empty ResourcesMap and
// DataSourcesMap) until a follow-up cleanup removes tf5to6server from
// main.go — its connection logic still needs coverage.
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
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// expandHome lives in instance_resource.go — its no-HOME edge case still
// matters and is hard to reach via the framework's typed Plan/State machinery.
func TestExpandHome_NoHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME env var not used on windows")
	}
	t.Setenv("HOME", "")
	got := expandHome("~/foo")
	if got != "~/foo" {
		t.Errorf("expandHome with empty HOME = %q, want %q", got, "~/foo")
	}
}
