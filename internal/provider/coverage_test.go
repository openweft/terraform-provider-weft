package provider

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// coverage_test.go — what's left here after the framework migration cleanup.
// The sdk/v2 provider has been removed entirely (see FRAMEWORK_MIGRATION.md);
// configureProvider, the sdk/v2 New() factory, and their tests went with it.
// The equivalent dial-side coverage on the framework side belongs to
// acceptance tests (see instance_resource_acc_test.go); unit-testing the
// framework's Configure() requires standing up a full provider.ConfigureRequest
// with the schema, which isn't easier than just running TF_ACC.
//
// Two helpers survive — they're pure-Go and exercised here so future moves
// can't break them silently.

// TestDefaultSSHSocket asserts the operator-visible default the framework
// provider's `ssh_socket` attribute fills in when blank. Stays a unit test
// because the helper lives in client.go, away from any framework dependency.
func TestDefaultSSHSocket(t *testing.T) {
	s := defaultSSHSocket()
	if !strings.HasSuffix(s, "/.weft/weft-ssh.sock") {
		t.Errorf("defaultSSHSocket() = %q, want suffix /.weft/weft-ssh.sock", s)
	}
	if !strings.HasPrefix(s, "/") {
		t.Errorf("defaultSSHSocket() = %q, expected absolute path", s)
	}
}

// TestDefaultSocket — the plain (no-SSH) sibling.
func TestDefaultSocket(t *testing.T) {
	s := defaultSocket()
	if !strings.HasSuffix(s, "/.weft/weft.sock") {
		t.Errorf("defaultSocket() = %q, want suffix /.weft/weft.sock", s)
	}
	if !strings.HasPrefix(s, "/") {
		t.Errorf("defaultSocket() = %q, expected absolute path", s)
	}
}

// TestExpandHome_NoHome covers expandHome's fallback when os.UserHomeDir()
// fails — the framework's typed Plan/State machinery makes this branch hard
// to reach from a regular resource test, so we exercise it directly.
func TestExpandHome_NoHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME env var not used on windows")
	}
	t.Setenv("HOME", "")
	// HomeDir on darwin reads $USER too — empty it as well so the function
	// genuinely fails to resolve.
	t.Setenv("USER", "")
	got := expandHome("~/foo")
	// On macOS, even with HOME+USER cleared, /etc/passwd lookups may still
	// resolve the home — accept either the unchanged tilde path or any
	// absolute path resolved through that fallback.
	if got != "~/foo" && !strings.HasPrefix(got, "/") {
		t.Errorf("expandHome with empty HOME = %q, want %q or an absolute path", got, "~/foo")
	}
	_ = os.Getenv // silence unused-import if Setenv inlines
}
