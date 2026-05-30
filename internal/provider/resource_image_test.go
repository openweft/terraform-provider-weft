package provider

import (
	"context"
	"fmt"
	"testing"

	weftv1 "github.com/openweft/weft-proto"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"google.golang.org/grpc"
)

func TestResourceImageSchema(t *testing.T) {
	r := resourceImage()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceImage schema invalid: %v", err)
	}
}

func TestResourceImageSchema_Fields(t *testing.T) {
	s := resourceImage().Schema

	requiredForceNew := []string{"from"}
	for _, field := range requiredForceNew {
		attr, ok := s[field]
		if !ok {
			t.Errorf("schema missing required field %q", field)
			continue
		}
		if !attr.Required {
			t.Errorf("field %q should be Required", field)
		}
		if !attr.ForceNew {
			t.Errorf("field %q should be ForceNew", field)
		}
	}

	checksum, ok := s["checksum"]
	if !ok {
		t.Fatal("schema missing field \"checksum\"")
	}
	if checksum.Required {
		t.Error("field \"checksum\" should be Optional, not Required")
	}
}

func TestResourceImageCreate(t *testing.T) {
	var gotURL, gotChecksum string
	mock := &mockWeftClient{
		pullImageFn: func(_ context.Context, req *weftv1.PullImageRequest, _ ...grpc.CallOption) (*weftv1.PullImageResponse, error) {
			gotURL = req.Url
			gotChecksum = req.Checksum
			return &weftv1.PullImageResponse{}, nil
		},
	}
	res := resourceImage()
	d := res.Data(nil)
	d.Set("from", "https://cloud.debian.org/image.raw")
	d.Set("checksum", "https://cloud.debian.org/SHA512SUMS")

	diags := resourceImageCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "https://cloud.debian.org/image.raw" {
		t.Errorf("ID = %q, want %q", d.Id(), "https://cloud.debian.org/image.raw")
	}
	if gotURL != "https://cloud.debian.org/image.raw" {
		t.Errorf("PullImage url = %q, want %q", gotURL, "https://cloud.debian.org/image.raw")
	}
	if gotChecksum != "https://cloud.debian.org/SHA512SUMS" {
		t.Errorf("PullImage checksum = %q, want %q", gotChecksum, "https://cloud.debian.org/SHA512SUMS")
	}
}

func TestResourceImageCreate_NoChecksum(t *testing.T) {
	mock := &mockWeftClient{
		pullImageFn: func(_ context.Context, _ *weftv1.PullImageRequest, _ ...grpc.CallOption) (*weftv1.PullImageResponse, error) {
			return &weftv1.PullImageResponse{}, nil
		},
	}
	res := resourceImage()
	d := res.Data(nil)
	d.Set("from", "https://cloud.ubuntu.com/image.img")

	diags := resourceImageCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "https://cloud.ubuntu.com/image.img" {
		t.Errorf("ID = %q, want %q", d.Id(), "https://cloud.ubuntu.com/image.img")
	}
}

func TestResourceImageCreate_PullError(t *testing.T) {
	mock := &mockWeftClient{
		pullImageFn: func(_ context.Context, _ *weftv1.PullImageRequest, _ ...grpc.CallOption) (*weftv1.PullImageResponse, error) {
			return nil, fmt.Errorf("image not found")
		},
	}
	res := resourceImage()
	d := res.Data(nil)
	d.Set("from", "https://cloud.example.com/image.img")

	diags := resourceImageCreate(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error, got none")
	}
	if d.Id() != "" {
		t.Errorf("ID should be empty on error, got %q", d.Id())
	}
}

func TestResourceImageSchema_PatchBlock(t *testing.T) {
	s := resourceImage().Schema

	patchSchema, ok := s["patch"]
	if !ok {
		t.Fatal("schema missing \"patch\" field")
	}
	if patchSchema.Required {
		t.Error("patch should be Optional")
	}
	if !patchSchema.ForceNew {
		t.Error("patch should be ForceNew")
	}

	patchRes, ok := patchSchema.Elem.(*schema.Resource)
	if !ok {
		t.Fatal("patch Elem must be *schema.Resource")
	}
	addSchema, ok := patchRes.Schema["add"]
	if !ok {
		t.Fatal("patch sub-schema missing \"add\" field")
	}
	addRes, ok := addSchema.Elem.(*schema.Resource)
	if !ok {
		t.Fatal("patch.add Elem must be *schema.Resource")
	}
	for _, field := range []string{"content", "dst", "trigger"} {
		if _, ok := addRes.Schema[field]; !ok {
			t.Errorf("patch.add sub-schema missing field %q", field)
		}
	}
	delSchema, ok := patchRes.Schema["del"]
	if !ok {
		t.Fatal("patch sub-schema missing \"del\" field")
	}
	delRes, ok := delSchema.Elem.(*schema.Resource)
	if !ok {
		t.Fatal("patch.del Elem must be *schema.Resource")
	}
	if _, ok := delRes.Schema["dst"]; !ok {
		t.Error("patch.del sub-schema missing \"dst\" field")
	}
	modSchema, ok := patchRes.Schema["mod"]
	if !ok {
		t.Fatal("patch sub-schema missing \"mod\" field")
	}
	modRes, ok := modSchema.Elem.(*schema.Resource)
	if !ok {
		t.Fatal("patch.mod Elem must be *schema.Resource")
	}
	for _, field := range []string{"dst", "old", "new"} {
		if _, ok := modRes.Schema[field]; !ok {
			t.Errorf("patch.mod sub-schema missing field %q", field)
		}
	}
}

func TestResourceImageCreate_WithPatch_CallsPatchImage(t *testing.T) {
	var patchReq *weftv1.PatchImageRequest
	mock := &mockWeftClient{
		pullImageFn: func(_ context.Context, _ *weftv1.PullImageRequest, _ ...grpc.CallOption) (*weftv1.PullImageResponse, error) {
			return &weftv1.PullImageResponse{}, nil
		},
		patchImageFn: func(_ context.Context, req *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			patchReq = req
			return &weftv1.PatchImageResponse{}, nil
		},
	}
	res := resourceImage()
	d := res.Data(nil)
	d.Set("from", "https://cloud.debian.org/image.raw")
	d.Set("patch", []interface{}{
		map[string]interface{}{
			"add": []interface{}{
				map[string]interface{}{
					"content": "GRUB_TERMINAL_OUTPUT=\"console\"\n",
					"dst":     "/etc/default/grub.d/99-console.cfg",
					"trigger": "grub-mkconfig",
				},
			},
			"del": []interface{}{
				map[string]interface{}{"dst": "/etc/old-config"},
			},
			"mod": []interface{}{
				map[string]interface{}{"dst": "/etc/hosts", "old": "localhost", "new": "myhost"},
			},
		},
	})

	diags := resourceImageCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if patchReq == nil {
		t.Fatal("PatchImage was not called")
	}
	if patchReq.Url != "https://cloud.debian.org/image.raw" {
		t.Errorf("PatchImage URL = %q, want %q", patchReq.Url, "https://cloud.debian.org/image.raw")
	}
	if len(patchReq.FileOps) != 1 {
		t.Fatalf("FileOps len = %d, want 1", len(patchReq.FileOps))
	}
	if patchReq.FileOps[0].Dst != "/etc/default/grub.d/99-console.cfg" {
		t.Errorf("FileOp dst = %q", patchReq.FileOps[0].Dst)
	}
	if patchReq.FileOps[0].Trigger != "grub-mkconfig" {
		t.Errorf("FileOp trigger = %q, want grub-mkconfig", patchReq.FileOps[0].Trigger)
	}
	if len(patchReq.DeleteOps) != 1 {
		t.Fatalf("DeleteOps len = %d, want 1", len(patchReq.DeleteOps))
	}
	if patchReq.DeleteOps[0].Dst != "/etc/old-config" {
		t.Errorf("DeleteOp dst = %q, want /etc/old-config", patchReq.DeleteOps[0].Dst)
	}
	if len(patchReq.ModOps) != 1 {
		t.Fatalf("ModOps len = %d, want 1", len(patchReq.ModOps))
	}
	if patchReq.ModOps[0].Dst != "/etc/hosts" || patchReq.ModOps[0].Old != "localhost" || patchReq.ModOps[0].New != "myhost" {
		t.Errorf("ModOp = {%q %q %q}, want {/etc/hosts localhost myhost}",
			patchReq.ModOps[0].Dst, patchReq.ModOps[0].Old, patchReq.ModOps[0].New)
	}
}

func TestResourceImageCreate_NoPatch_PatchImageNotCalled(t *testing.T) {
	patchCalled := false
	mock := &mockWeftClient{
		pullImageFn: func(_ context.Context, _ *weftv1.PullImageRequest, _ ...grpc.CallOption) (*weftv1.PullImageResponse, error) {
			return &weftv1.PullImageResponse{}, nil
		},
		patchImageFn: func(_ context.Context, _ *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			patchCalled = true
			return &weftv1.PatchImageResponse{}, nil
		},
	}
	res := resourceImage()
	d := res.Data(nil)
	d.Set("from", "https://cloud.ubuntu.com/image.img")

	diags := resourceImageCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if patchCalled {
		t.Error("PatchImage should not be called when no patch blocks are declared")
	}
}

func TestResourceImageCreate_PatchError(t *testing.T) {
	mock := &mockWeftClient{
		pullImageFn: func(_ context.Context, _ *weftv1.PullImageRequest, _ ...grpc.CallOption) (*weftv1.PullImageResponse, error) {
			return &weftv1.PullImageResponse{}, nil
		},
		patchImageFn: func(_ context.Context, _ *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			return nil, fmt.Errorf("image not cached")
		},
	}
	res := resourceImage()
	d := res.Data(nil)
	d.Set("from", "https://cloud.example.com/image.raw")
	d.Set("patch", []interface{}{
		map[string]interface{}{
			"add": []interface{}{
				map[string]interface{}{"content": "x", "dst": "/a", "trigger": ""},
			},
			"del": []interface{}{},
			"mod": []interface{}{},
		},
	})

	diags := resourceImageCreate(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error from PatchImage failure, got none")
	}
}
