package provider

import (
	"strings"
	"testing"
)

func TestDefaultSocket(t *testing.T) {
	s := defaultSocket()
	if !strings.HasSuffix(s, "/.weft/weft.sock") {
		t.Errorf("defaultSocket() = %q, want suffix /.weft/weft.sock", s)
	}
	if !strings.HasPrefix(s, "/") {
		t.Errorf("defaultSocket() = %q, expected absolute path", s)
	}
}

func TestProviderSchema(t *testing.T) {
	p := New()
	if err := p.InternalValidate(); err != nil {
		t.Fatalf("provider schema invalid: %v", err)
	}
}

// TestProviderEmptyResourceMaps asserts that the sdk/v2 provider has shed
// all resources/data sources — they now all live in the framework half (see
// framework_provider.go and FRAMEWORK_MIGRATION.md). A follow-up cleanup
// commit removes tf5to6server from main.go and this whole sdk/v2 provider.
func TestProviderEmptyResourceMaps(t *testing.T) {
	p := New()
	if len(p.ResourcesMap) != 0 {
		t.Errorf("sdk/v2 ResourcesMap should be empty, got %d entries", len(p.ResourcesMap))
	}
	if len(p.DataSourcesMap) != 0 {
		t.Errorf("sdk/v2 DataSourcesMap should be empty, got %d entries", len(p.DataSourcesMap))
	}
}
