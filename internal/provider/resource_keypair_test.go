package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceKeypairSchema(t *testing.T) {
	r := resourceKeypair()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceKeypair schema invalid: %v", err)
	}
}

func TestResourceKeypairSchema_Fields(t *testing.T) {
	s := resourceKeypair().Schema

	requiredForceNew := []string{"name", "file_path"}
	for _, field := range requiredForceNew {
		attr, ok := s[field]
		if !ok {
			t.Errorf("schema missing field %q", field)
			continue
		}
		if !attr.Required {
			t.Errorf("field %q should be Required", field)
		}
		if !attr.ForceNew {
			t.Errorf("field %q should be ForceNew", field)
		}
	}

	for _, field := range []string{"resolved_path", "public_key"} {
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

func TestResourceKeypairCreate(t *testing.T) {
	dir := t.TempDir()
	privKey := filepath.Join(dir, "id_ed25519")
	pubKey := privKey + ".pub"
	const pubContent = "ssh-ed25519 AAAA test@host"

	if err := os.WriteFile(privKey, []byte("private"), 0600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(pubKey, []byte(pubContent+"\n"), 0644); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	res := resourceKeypair()
	d := res.Data(nil)
	d.Set("name", "mock")
	d.Set("file_path", privKey)

	diags := resourceKeypairCreate(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "mock" {
		t.Errorf("ID = %q, want %q", d.Id(), "mock")
	}
	if got := d.Get("public_key").(string); got != pubContent {
		t.Errorf("public_key = %q, want %q", got, pubContent)
	}
	if got := d.Get("resolved_path").(string); got != privKey {
		t.Errorf("resolved_path = %q, want %q", got, privKey)
	}
}

func TestResourceKeypairCreate_MissingPubKey(t *testing.T) {
	dir := t.TempDir()
	privKey := filepath.Join(dir, "id_ed25519")
	// Only private key, no .pub file.
	if err := os.WriteFile(privKey, []byte("private"), 0600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	res := resourceKeypair()
	d := res.Data(nil)
	d.Set("name", "mock")
	d.Set("file_path", privKey)

	diags := resourceKeypairCreate(context.Background(), d, nil)
	if !diags.HasError() {
		t.Fatal("expected error when .pub file is missing, got none")
	}
}
