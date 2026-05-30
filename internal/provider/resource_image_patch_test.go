package provider

import (
	"context"
	"fmt"
	"testing"

	weftv1 "github.com/openweft/weft-proto"
	"google.golang.org/grpc"
)

func TestResourceImagePatchSchema(t *testing.T) {
	r := resourceImagePatch()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resourceImagePatch schema invalid: %v", err)
	}
}

func TestResourceImagePatchCreate_WithImagesList_CallsPatchImage(t *testing.T) {
	var got []*weftv1.PatchImageRequest
	mock := &mockWeftClient{
		patchImageFn: func(_ context.Context, req *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			got = append(got, req)
			return &weftv1.PatchImageResponse{}, nil
		},
	}

	res := resourceImagePatch()
	d := res.Data(nil)
	d.Set("images", []interface{}{"https://cloud.debian.org/image.raw", "https://cloud.ubuntu.com/image.img"})
	d.Set("patch", []interface{}{map[string]interface{}{
		"add": []interface{}{map[string]interface{}{"content": "x", "dst": "/a", "trigger": ""}},
		"del": []interface{}{},
		"mod": []interface{}{},
	}})

	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if len(got) != 2 {
		t.Fatalf("PatchImage called %d times, want 2", len(got))
	}
	if got[0].Url != "https://cloud.debian.org/image.raw" {
		t.Errorf("PatchImage url = %q, want %q", got[0].Url, "https://cloud.debian.org/image.raw")
	}
}

func TestResourceImagePatchCreate_NoImages_CallsListImagesAndPatchAll(t *testing.T) {
	var got []*weftv1.PatchImageRequest
	mock := &mockWeftClient{
		listImagesFn: func(_ context.Context, _ *weftv1.ListImagesRequest, _ ...grpc.CallOption) (*weftv1.ListImagesResponse, error) {
			return &weftv1.ListImagesResponse{Images: []*weftv1.ImageInfo{{Url: "u1"}, {Url: "u2"}}}, nil
		},
		patchImageFn: func(_ context.Context, req *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			got = append(got, req)
			return &weftv1.PatchImageResponse{}, nil
		},
	}

	res := resourceImagePatch()
	d := res.Data(nil)
	d.Set("patch", []interface{}{map[string]interface{}{
		"add": []interface{}{map[string]interface{}{"content": "x", "dst": "/a", "trigger": ""}},
		"del": []interface{}{},
		"mod": []interface{}{},
	}})

	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(mock))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if len(got) != 2 {
		t.Fatalf("PatchImage called %d times, want 2", len(got))
	}
}

func TestResourceImagePatchCreate_PatchError(t *testing.T) {
	mock := &mockWeftClient{
		patchImageFn: func(_ context.Context, _ *weftv1.PatchImageRequest, _ ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
			return nil, fmt.Errorf("image not cached")
		},
	}
	res := resourceImagePatch()
	d := res.Data(nil)
	d.Set("images", []interface{}{"https://cloud.example.com/image.raw"})
	d.Set("patch", []interface{}{map[string]interface{}{
		"add": []interface{}{map[string]interface{}{"content": "x", "dst": "/a", "trigger": ""}},
		"del": []interface{}{},
		"mod": []interface{}{},
	}})

	diags := resourceImagePatchCreate(context.Background(), d, newTestClient(mock))
	if !diags.HasError() {
		t.Fatal("expected error from PatchImage failure, got none")
	}
}
