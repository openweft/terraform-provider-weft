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

func TestProviderHasResources(t *testing.T) {
	p := New()
	resources := []string{"weft_deployment", "weft_endpoint", "weft_instance", "weft_image", "weft_images", "weft_keypair"}
	for _, name := range resources {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("provider missing resource %q", name)
		}
	}
}

func TestProviderHasDataSources(t *testing.T) {
	p := New()
	dataSources := []string{"weft_config"}
	for _, name := range dataSources {
		if _, ok := p.DataSourcesMap[name]; !ok {
			t.Errorf("provider missing data source %q", name)
		}
	}
}
