package provider

import (
	"context"

	weftv1 "github.com/openweft/weft-proto"
	"google.golang.org/grpc"
)

// mockWeftClient implements weftv1.WeftAgentClient for unit testing.
type mockWeftClient struct {
	listVMsFn       func(context.Context, *weftv1.ListVMsRequest, ...grpc.CallOption) (*weftv1.ListVMsResponse, error)
	listImagesFn    func(context.Context, *weftv1.ListImagesRequest, ...grpc.CallOption) (*weftv1.ListImagesResponse, error)
	vmStatusFn      func(context.Context, *weftv1.VMStatusRequest, ...grpc.CallOption) (*weftv1.VMStatusResponse, error)
	startVMFn       func(context.Context, *weftv1.StartVMRequest, ...grpc.CallOption) (*weftv1.StartVMResponse, error)
	stopVMFn        func(context.Context, *weftv1.StopVMRequest, ...grpc.CallOption) (*weftv1.StopVMResponse, error)
	createVMFn      func(context.Context, *weftv1.CreateVMRequest, ...grpc.CallOption) (*weftv1.CreateVMResponse, error)
	deleteVMFn      func(context.Context, *weftv1.DeleteVMRequest, ...grpc.CallOption) (*weftv1.DeleteVMResponse, error)
	provisionVMFn   func(context.Context, *weftv1.ProvisionVMRequest, ...grpc.CallOption) (*weftv1.ProvisionVMResponse, error)
	deprovisionVMFn func(context.Context, *weftv1.DeprovisionVMRequest, ...grpc.CallOption) (*weftv1.DeprovisionVMResponse, error)
	pullImagesFn    func(context.Context, *weftv1.PullImagesRequest, ...grpc.CallOption) (*weftv1.PullImagesResponse, error)
	pullImageFn     func(context.Context, *weftv1.PullImageRequest, ...grpc.CallOption) (*weftv1.PullImageResponse, error)
	patchImageFn    func(context.Context, *weftv1.PatchImageRequest, ...grpc.CallOption) (*weftv1.PatchImageResponse, error)
	cleanImagesFn   func(context.Context, *weftv1.CleanImagesRequest, ...grpc.CallOption) (*weftv1.CleanImagesResponse, error)
	waitVMFn        func(context.Context, *weftv1.WaitVMRequest, ...grpc.CallOption) (*weftv1.WaitVMResponse, error)
}

func (m *mockWeftClient) ListVMs(ctx context.Context, in *weftv1.ListVMsRequest, opts ...grpc.CallOption) (*weftv1.ListVMsResponse, error) {
	if m.listVMsFn != nil {
		return m.listVMsFn(ctx, in, opts...)
	}
	return &weftv1.ListVMsResponse{}, nil
}

func (m *mockWeftClient) VMStatus(ctx context.Context, in *weftv1.VMStatusRequest, opts ...grpc.CallOption) (*weftv1.VMStatusResponse, error) {
	if m.vmStatusFn != nil {
		return m.vmStatusFn(ctx, in, opts...)
	}
	return &weftv1.VMStatusResponse{}, nil
}

func (m *mockWeftClient) StartVM(ctx context.Context, in *weftv1.StartVMRequest, opts ...grpc.CallOption) (*weftv1.StartVMResponse, error) {
	if m.startVMFn != nil {
		return m.startVMFn(ctx, in, opts...)
	}
	return &weftv1.StartVMResponse{}, nil
}

func (m *mockWeftClient) StopVM(ctx context.Context, in *weftv1.StopVMRequest, opts ...grpc.CallOption) (*weftv1.StopVMResponse, error) {
	if m.stopVMFn != nil {
		return m.stopVMFn(ctx, in, opts...)
	}
	return &weftv1.StopVMResponse{}, nil
}

func (m *mockWeftClient) CreateVM(ctx context.Context, in *weftv1.CreateVMRequest, opts ...grpc.CallOption) (*weftv1.CreateVMResponse, error) {
	if m.createVMFn != nil {
		return m.createVMFn(ctx, in, opts...)
	}
	return &weftv1.CreateVMResponse{}, nil
}

func (m *mockWeftClient) DeleteVM(ctx context.Context, in *weftv1.DeleteVMRequest, opts ...grpc.CallOption) (*weftv1.DeleteVMResponse, error) {
	if m.deleteVMFn != nil {
		return m.deleteVMFn(ctx, in, opts...)
	}
	return &weftv1.DeleteVMResponse{}, nil
}

func (m *mockWeftClient) ProvisionVM(ctx context.Context, in *weftv1.ProvisionVMRequest, opts ...grpc.CallOption) (*weftv1.ProvisionVMResponse, error) {
	if m.provisionVMFn != nil {
		return m.provisionVMFn(ctx, in, opts...)
	}
	return &weftv1.ProvisionVMResponse{}, nil
}

func (m *mockWeftClient) DeprovisionVM(ctx context.Context, in *weftv1.DeprovisionVMRequest, opts ...grpc.CallOption) (*weftv1.DeprovisionVMResponse, error) {
	if m.deprovisionVMFn != nil {
		return m.deprovisionVMFn(ctx, in, opts...)
	}
	return &weftv1.DeprovisionVMResponse{}, nil
}

func (m *mockWeftClient) PullImages(ctx context.Context, in *weftv1.PullImagesRequest, opts ...grpc.CallOption) (*weftv1.PullImagesResponse, error) {
	if m.pullImagesFn != nil {
		return m.pullImagesFn(ctx, in, opts...)
	}
	return &weftv1.PullImagesResponse{}, nil
}

func (m *mockWeftClient) PullImage(ctx context.Context, in *weftv1.PullImageRequest, opts ...grpc.CallOption) (*weftv1.PullImageResponse, error) {
	if m.pullImageFn != nil {
		return m.pullImageFn(ctx, in, opts...)
	}
	return &weftv1.PullImageResponse{}, nil
}

func (m *mockWeftClient) PatchImage(ctx context.Context, in *weftv1.PatchImageRequest, opts ...grpc.CallOption) (*weftv1.PatchImageResponse, error) {
	if m.patchImageFn != nil {
		return m.patchImageFn(ctx, in, opts...)
	}
	return &weftv1.PatchImageResponse{}, nil
}

func (m *mockWeftClient) ListImages(_ context.Context, _ *weftv1.ListImagesRequest, _ ...grpc.CallOption) (*weftv1.ListImagesResponse, error) {
	if m.listImagesFn != nil {
		return m.listImagesFn(context.Background(), &weftv1.ListImagesRequest{})
	}
	return &weftv1.ListImagesResponse{}, nil
}

func (m *mockWeftClient) CleanImages(ctx context.Context, in *weftv1.CleanImagesRequest, opts ...grpc.CallOption) (*weftv1.CleanImagesResponse, error) {
	if m.cleanImagesFn != nil {
		return m.cleanImagesFn(ctx, in, opts...)
	}
	return &weftv1.CleanImagesResponse{}, nil
}

func (m *mockWeftClient) WaitVM(ctx context.Context, in *weftv1.WaitVMRequest, opts ...grpc.CallOption) (*weftv1.WaitVMResponse, error) {
	if m.waitVMFn != nil {
		return m.waitVMFn(ctx, in, opts...)
	}
	return &weftv1.WaitVMResponse{}, nil
}

// Stubs for RPCs not exercised by terraform-provider-weft tests. Returning
// zero-value responses keeps the mock conforming to weftv1.WeftAgentClient
// without requiring per-test overrides. Add a typed *Fn field above and an
// `if m.xxxFn != nil { return m.xxxFn(…) }` guard here when a test needs to
// drive one of these methods.

func (m *mockWeftClient) RegisterMicroVM(_ context.Context, _ *weftv1.RegisterMicroVMRequest, _ ...grpc.CallOption) (*weftv1.RegisterMicroVMResponse, error) {
	return &weftv1.RegisterMicroVMResponse{}, nil
}

func (m *mockWeftClient) VMTimings(_ context.Context, _ *weftv1.VMTimingsRequest, _ ...grpc.CallOption) (*weftv1.VMTimingsResponse, error) {
	return &weftv1.VMTimingsResponse{}, nil
}

func (m *mockWeftClient) VMLogs(_ context.Context, _ *weftv1.VMLogsRequest, _ ...grpc.CallOption) (*weftv1.VMLogsResponse, error) {
	return &weftv1.VMLogsResponse{}, nil
}

func (m *mockWeftClient) ListProjects(_ context.Context, _ *weftv1.ListProjectsRequest, _ ...grpc.CallOption) (*weftv1.ListProjectsResponse, error) {
	return &weftv1.ListProjectsResponse{}, nil
}

func (m *mockWeftClient) CreateProject(_ context.Context, _ *weftv1.CreateProjectRequest, _ ...grpc.CallOption) (*weftv1.CreateProjectResponse, error) {
	return &weftv1.CreateProjectResponse{}, nil
}

func (m *mockWeftClient) RenameProject(_ context.Context, _ *weftv1.RenameProjectRequest, _ ...grpc.CallOption) (*weftv1.RenameProjectResponse, error) {
	return &weftv1.RenameProjectResponse{}, nil
}

func (m *mockWeftClient) DeleteProject(_ context.Context, _ *weftv1.DeleteProjectRequest, _ ...grpc.CallOption) (*weftv1.DeleteProjectResponse, error) {
	return &weftv1.DeleteProjectResponse{}, nil
}

func (m *mockWeftClient) AddProjectMember(_ context.Context, _ *weftv1.AddProjectMemberRequest, _ ...grpc.CallOption) (*weftv1.AddProjectMemberResponse, error) {
	return &weftv1.AddProjectMemberResponse{}, nil
}

func (m *mockWeftClient) RemoveProjectMember(_ context.Context, _ *weftv1.RemoveProjectMemberRequest, _ ...grpc.CallOption) (*weftv1.RemoveProjectMemberResponse, error) {
	return &weftv1.RemoveProjectMemberResponse{}, nil
}

func (m *mockWeftClient) ListProjectMembers(_ context.Context, _ *weftv1.ListProjectMembersRequest, _ ...grpc.CallOption) (*weftv1.ListProjectMembersResponse, error) {
	return &weftv1.ListProjectMembersResponse{}, nil
}

func (m *mockWeftClient) ListUsers(_ context.Context, _ *weftv1.ListUsersRequest, _ ...grpc.CallOption) (*weftv1.ListUsersResponse, error) {
	return &weftv1.ListUsersResponse{}, nil
}

func (m *mockWeftClient) GetUser(_ context.Context, _ *weftv1.GetUserRequest, _ ...grpc.CallOption) (*weftv1.GetUserResponse, error) {
	return &weftv1.GetUserResponse{}, nil
}

func (m *mockWeftClient) Me(_ context.Context, _ *weftv1.MeRequest, _ ...grpc.CallOption) (*weftv1.MeResponse, error) {
	return &weftv1.MeResponse{}, nil
}

func (m *mockWeftClient) SetUserDisplayName(_ context.Context, _ *weftv1.SetUserDisplayNameRequest, _ ...grpc.CallOption) (*weftv1.SetUserDisplayNameResponse, error) {
	return &weftv1.SetUserDisplayNameResponse{}, nil
}

func (m *mockWeftClient) DeleteUser(_ context.Context, _ *weftv1.DeleteUserRequest, _ ...grpc.CallOption) (*weftv1.DeleteUserResponse, error) {
	return &weftv1.DeleteUserResponse{}, nil
}

func (m *mockWeftClient) ListNetworks(_ context.Context, _ *weftv1.ListNetworksRequest, _ ...grpc.CallOption) (*weftv1.ListNetworksResponse, error) {
	return &weftv1.ListNetworksResponse{}, nil
}

func (m *mockWeftClient) CreateNetwork(_ context.Context, _ *weftv1.CreateNetworkRequest, _ ...grpc.CallOption) (*weftv1.CreateNetworkResponse, error) {
	return &weftv1.CreateNetworkResponse{}, nil
}

func (m *mockWeftClient) RenameNetwork(_ context.Context, _ *weftv1.RenameNetworkRequest, _ ...grpc.CallOption) (*weftv1.RenameNetworkResponse, error) {
	return &weftv1.RenameNetworkResponse{}, nil
}

func (m *mockWeftClient) SetNetworkDNS(_ context.Context, _ *weftv1.SetNetworkDNSRequest, _ ...grpc.CallOption) (*weftv1.SetNetworkDNSResponse, error) {
	return &weftv1.SetNetworkDNSResponse{}, nil
}

func (m *mockWeftClient) DeleteNetwork(_ context.Context, _ *weftv1.DeleteNetworkRequest, _ ...grpc.CallOption) (*weftv1.DeleteNetworkResponse, error) {
	return &weftv1.DeleteNetworkResponse{}, nil
}

func (m *mockWeftClient) SetNetworkDefaultSecurityGroups(_ context.Context, _ *weftv1.SetNetworkDefaultSecurityGroupsRequest, _ ...grpc.CallOption) (*weftv1.SetNetworkDefaultSecurityGroupsResponse, error) {
	return &weftv1.SetNetworkDefaultSecurityGroupsResponse{}, nil
}

func (m *mockWeftClient) ListSecurityGroups(_ context.Context, _ *weftv1.ListSecurityGroupsRequest, _ ...grpc.CallOption) (*weftv1.ListSecurityGroupsResponse, error) {
	return &weftv1.ListSecurityGroupsResponse{}, nil
}

func (m *mockWeftClient) CreateSecurityGroup(_ context.Context, _ *weftv1.CreateSecurityGroupRequest, _ ...grpc.CallOption) (*weftv1.CreateSecurityGroupResponse, error) {
	return &weftv1.CreateSecurityGroupResponse{}, nil
}

func (m *mockWeftClient) RenameSecurityGroup(_ context.Context, _ *weftv1.RenameSecurityGroupRequest, _ ...grpc.CallOption) (*weftv1.RenameSecurityGroupResponse, error) {
	return &weftv1.RenameSecurityGroupResponse{}, nil
}

func (m *mockWeftClient) SetSecurityGroupDescription(_ context.Context, _ *weftv1.SetSecurityGroupDescriptionRequest, _ ...grpc.CallOption) (*weftv1.SetSecurityGroupDescriptionResponse, error) {
	return &weftv1.SetSecurityGroupDescriptionResponse{}, nil
}

func (m *mockWeftClient) SetSecurityGroupRules(_ context.Context, _ *weftv1.SetSecurityGroupRulesRequest, _ ...grpc.CallOption) (*weftv1.SetSecurityGroupRulesResponse, error) {
	return &weftv1.SetSecurityGroupRulesResponse{}, nil
}

func (m *mockWeftClient) DeleteSecurityGroup(_ context.Context, _ *weftv1.DeleteSecurityGroupRequest, _ ...grpc.CallOption) (*weftv1.DeleteSecurityGroupResponse, error) {
	return &weftv1.DeleteSecurityGroupResponse{}, nil
}

func (m *mockWeftClient) ListVolumes(_ context.Context, _ *weftv1.ListVolumesRequest, _ ...grpc.CallOption) (*weftv1.ListVolumesResponse, error) {
	return &weftv1.ListVolumesResponse{}, nil
}

func (m *mockWeftClient) CreateVolume(_ context.Context, _ *weftv1.CreateVolumeRequest, _ ...grpc.CallOption) (*weftv1.CreateVolumeResponse, error) {
	return &weftv1.CreateVolumeResponse{}, nil
}

func (m *mockWeftClient) RenameVolume(_ context.Context, _ *weftv1.RenameVolumeRequest, _ ...grpc.CallOption) (*weftv1.RenameVolumeResponse, error) {
	return &weftv1.RenameVolumeResponse{}, nil
}

func (m *mockWeftClient) ResizeVolume(_ context.Context, _ *weftv1.ResizeVolumeRequest, _ ...grpc.CallOption) (*weftv1.ResizeVolumeResponse, error) {
	return &weftv1.ResizeVolumeResponse{}, nil
}

func (m *mockWeftClient) AttachVolume(_ context.Context, _ *weftv1.AttachVolumeRequest, _ ...grpc.CallOption) (*weftv1.AttachVolumeResponse, error) {
	return &weftv1.AttachVolumeResponse{}, nil
}

func (m *mockWeftClient) DetachVolume(_ context.Context, _ *weftv1.DetachVolumeRequest, _ ...grpc.CallOption) (*weftv1.DetachVolumeResponse, error) {
	return &weftv1.DetachVolumeResponse{}, nil
}

func (m *mockWeftClient) DeleteVolume(_ context.Context, _ *weftv1.DeleteVolumeRequest, _ ...grpc.CallOption) (*weftv1.DeleteVolumeResponse, error) {
	return &weftv1.DeleteVolumeResponse{}, nil
}

func (m *mockWeftClient) CreateVolumeSnapshot(_ context.Context, _ *weftv1.CreateVolumeSnapshotRequest, _ ...grpc.CallOption) (*weftv1.CreateVolumeSnapshotResponse, error) {
	return &weftv1.CreateVolumeSnapshotResponse{Snapshot: &weftv1.VolumeSnapshotInfo{}}, nil
}

func (m *mockWeftClient) ListVolumeSnapshots(_ context.Context, _ *weftv1.ListVolumeSnapshotsRequest, _ ...grpc.CallOption) (*weftv1.ListVolumeSnapshotsResponse, error) {
	return &weftv1.ListVolumeSnapshotsResponse{}, nil
}

func (m *mockWeftClient) RestoreVolumeSnapshot(_ context.Context, _ *weftv1.RestoreVolumeSnapshotRequest, _ ...grpc.CallOption) (*weftv1.RestoreVolumeSnapshotResponse, error) {
	return &weftv1.RestoreVolumeSnapshotResponse{Volume: &weftv1.VolumeInfo{}}, nil
}

func (m *mockWeftClient) DeleteVolumeSnapshot(_ context.Context, _ *weftv1.DeleteVolumeSnapshotRequest, _ ...grpc.CallOption) (*weftv1.DeleteVolumeSnapshotResponse, error) {
	return &weftv1.DeleteVolumeSnapshotResponse{}, nil
}

func (m *mockWeftClient) WatchEvents(_ context.Context, _ *weftv1.WatchEventsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[weftv1.PlatformEvent], error) {
	return nil, nil
}

func (m *mockWeftClient) RenderNATSAuthorization(_ context.Context, _ *weftv1.RenderNATSAuthorizationRequest, _ ...grpc.CallOption) (*weftv1.RenderNATSAuthorizationResponse, error) {
	return &weftv1.RenderNATSAuthorizationResponse{}, nil
}

func (m *mockWeftClient) RegisterHost(_ context.Context, _ *weftv1.RegisterHostRequest, _ ...grpc.CallOption) (*weftv1.RegisterHostResponse, error) {
	return &weftv1.RegisterHostResponse{}, nil
}

func (m *mockWeftClient) ListHosts(_ context.Context, _ *weftv1.ListHostsRequest, _ ...grpc.CallOption) (*weftv1.ListHostsResponse, error) {
	return &weftv1.ListHostsResponse{}, nil
}

func (m *mockWeftClient) GetHost(_ context.Context, _ *weftv1.GetHostRequest, _ ...grpc.CallOption) (*weftv1.GetHostResponse, error) {
	return &weftv1.GetHostResponse{}, nil
}

func (m *mockWeftClient) HeartbeatHost(_ context.Context, _ *weftv1.HeartbeatHostRequest, _ ...grpc.CallOption) (*weftv1.HeartbeatHostResponse, error) {
	return &weftv1.HeartbeatHostResponse{}, nil
}

func (m *mockWeftClient) SetHostState(_ context.Context, _ *weftv1.SetHostStateRequest, _ ...grpc.CallOption) (*weftv1.SetHostStateResponse, error) {
	return &weftv1.SetHostStateResponse{}, nil
}

func (m *mockWeftClient) SetHostLabels(_ context.Context, _ *weftv1.SetHostLabelsRequest, _ ...grpc.CallOption) (*weftv1.SetHostLabelsResponse, error) {
	return &weftv1.SetHostLabelsResponse{}, nil
}

func (m *mockWeftClient) DeleteHost(_ context.Context, _ *weftv1.DeleteHostRequest, _ ...grpc.CallOption) (*weftv1.DeleteHostResponse, error) {
	return &weftv1.DeleteHostResponse{}, nil
}

func (m *mockWeftClient) PublishShareToProject(_ context.Context, _ *weftv1.PublishShareToProjectRequest, _ ...grpc.CallOption) (*weftv1.PublishShareToProjectResponse, error) {
	return &weftv1.PublishShareToProjectResponse{}, nil
}

// --- Tenants ---

func (m *mockWeftClient) ListTenants(_ context.Context, _ *weftv1.ListTenantsRequest, _ ...grpc.CallOption) (*weftv1.ListTenantsResponse, error) {
	return &weftv1.ListTenantsResponse{}, nil
}

func (m *mockWeftClient) CreateTenant(_ context.Context, _ *weftv1.CreateTenantRequest, _ ...grpc.CallOption) (*weftv1.CreateTenantResponse, error) {
	return &weftv1.CreateTenantResponse{}, nil
}

func (m *mockWeftClient) DeleteTenant(_ context.Context, _ *weftv1.DeleteTenantRequest, _ ...grpc.CallOption) (*weftv1.DeleteTenantResponse, error) {
	return &weftv1.DeleteTenantResponse{}, nil
}

func (m *mockWeftClient) AddTenantAdmin(_ context.Context, _ *weftv1.AddTenantAdminRequest, _ ...grpc.CallOption) (*weftv1.AddTenantAdminResponse, error) {
	return &weftv1.AddTenantAdminResponse{}, nil
}

func (m *mockWeftClient) RemoveTenantAdmin(_ context.Context, _ *weftv1.RemoveTenantAdminRequest, _ ...grpc.CallOption) (*weftv1.RemoveTenantAdminResponse, error) {
	return &weftv1.RemoveTenantAdminResponse{}, nil
}

func (m *mockWeftClient) AddTenantMember(_ context.Context, _ *weftv1.AddTenantMemberRequest, _ ...grpc.CallOption) (*weftv1.AddTenantMemberResponse, error) {
	return &weftv1.AddTenantMemberResponse{}, nil
}

func (m *mockWeftClient) RemoveTenantMember(_ context.Context, _ *weftv1.RemoveTenantMemberRequest, _ ...grpc.CallOption) (*weftv1.RemoveTenantMemberResponse, error) {
	return &weftv1.RemoveTenantMemberResponse{}, nil
}

// --- Quotas ---

func (m *mockWeftClient) GetTenantQuota(_ context.Context, _ *weftv1.GetTenantQuotaRequest, _ ...grpc.CallOption) (*weftv1.GetTenantQuotaResponse, error) {
	return &weftv1.GetTenantQuotaResponse{}, nil
}

func (m *mockWeftClient) SetTenantQuota(_ context.Context, _ *weftv1.SetTenantQuotaRequest, _ ...grpc.CallOption) (*weftv1.SetTenantQuotaResponse, error) {
	return &weftv1.SetTenantQuotaResponse{}, nil
}

func (m *mockWeftClient) GetProjectQuota(_ context.Context, _ *weftv1.GetProjectQuotaRequest, _ ...grpc.CallOption) (*weftv1.GetProjectQuotaResponse, error) {
	return &weftv1.GetProjectQuotaResponse{}, nil
}

func (m *mockWeftClient) SetProjectQuota(_ context.Context, _ *weftv1.SetProjectQuotaRequest, _ ...grpc.CallOption) (*weftv1.SetProjectQuotaResponse, error) {
	return &weftv1.SetProjectQuotaResponse{}, nil
}

// --- Shares ---

func (m *mockWeftClient) ListShares(_ context.Context, _ *weftv1.ListSharesRequest, _ ...grpc.CallOption) (*weftv1.ListSharesResponse, error) {
	return &weftv1.ListSharesResponse{}, nil
}

func (m *mockWeftClient) CreateShare(_ context.Context, _ *weftv1.CreateShareRequest, _ ...grpc.CallOption) (*weftv1.CreateShareResponse, error) {
	return &weftv1.CreateShareResponse{}, nil
}

func (m *mockWeftClient) DeleteShare(_ context.Context, _ *weftv1.DeleteShareRequest, _ ...grpc.CallOption) (*weftv1.DeleteShareResponse, error) {
	return &weftv1.DeleteShareResponse{}, nil
}

// --- Floating IPs ---

func (m *mockWeftClient) ListFloatingIPs(_ context.Context, _ *weftv1.ListFloatingIPsRequest, _ ...grpc.CallOption) (*weftv1.ListFloatingIPsResponse, error) {
	return &weftv1.ListFloatingIPsResponse{}, nil
}

func (m *mockWeftClient) AllocateFloatingIP(_ context.Context, _ *weftv1.AllocateFloatingIPRequest, _ ...grpc.CallOption) (*weftv1.AllocateFloatingIPResponse, error) {
	return &weftv1.AllocateFloatingIPResponse{}, nil
}

func (m *mockWeftClient) ReleaseFloatingIP(_ context.Context, _ *weftv1.ReleaseFloatingIPRequest, _ ...grpc.CallOption) (*weftv1.ReleaseFloatingIPResponse, error) {
	return &weftv1.ReleaseFloatingIPResponse{}, nil
}

func (m *mockWeftClient) MapFloatingIP(_ context.Context, _ *weftv1.MapFloatingIPRequest, _ ...grpc.CallOption) (*weftv1.MapFloatingIPResponse, error) {
	return &weftv1.MapFloatingIPResponse{}, nil
}

func (m *mockWeftClient) UnmapFloatingIP(_ context.Context, _ *weftv1.UnmapFloatingIPRequest, _ ...grpc.CallOption) (*weftv1.UnmapFloatingIPResponse, error) {
	return &weftv1.UnmapFloatingIPResponse{}, nil
}

// --- Flavors ---

func (m *mockWeftClient) ListFlavors(_ context.Context, _ *weftv1.ListFlavorsRequest, _ ...grpc.CallOption) (*weftv1.ListFlavorsResponse, error) {
	return &weftv1.ListFlavorsResponse{}, nil
}

func (m *mockWeftClient) GetFlavor(_ context.Context, _ *weftv1.GetFlavorRequest, _ ...grpc.CallOption) (*weftv1.GetFlavorResponse, error) {
	return &weftv1.GetFlavorResponse{}, nil
}

func (m *mockWeftClient) SetFlavor(_ context.Context, _ *weftv1.SetFlavorRequest, _ ...grpc.CallOption) (*weftv1.SetFlavorResponse, error) {
	return &weftv1.SetFlavorResponse{}, nil
}

func (m *mockWeftClient) DeleteFlavor(_ context.Context, _ *weftv1.DeleteFlavorRequest, _ ...grpc.CallOption) (*weftv1.DeleteFlavorResponse, error) {
	return &weftv1.DeleteFlavorResponse{}, nil
}

// --- Scripts ---

func (m *mockWeftClient) ListScripts(_ context.Context, _ *weftv1.ListScriptsRequest, _ ...grpc.CallOption) (*weftv1.ListScriptsResponse, error) {
	return &weftv1.ListScriptsResponse{}, nil
}

func (m *mockWeftClient) GetScript(_ context.Context, _ *weftv1.GetScriptRequest, _ ...grpc.CallOption) (*weftv1.GetScriptResponse, error) {
	return &weftv1.GetScriptResponse{}, nil
}

func (m *mockWeftClient) SetScript(_ context.Context, _ *weftv1.SetScriptRequest, _ ...grpc.CallOption) (*weftv1.SetScriptResponse, error) {
	return &weftv1.SetScriptResponse{}, nil
}

func (m *mockWeftClient) DeleteScript(_ context.Context, _ *weftv1.DeleteScriptRequest, _ ...grpc.CallOption) (*weftv1.DeleteScriptResponse, error) {
	return &weftv1.DeleteScriptResponse{}, nil
}

// --- Per-VM properties ---

func (m *mockWeftClient) ListVMProperties(_ context.Context, _ *weftv1.ListVMPropertiesRequest, _ ...grpc.CallOption) (*weftv1.ListVMPropertiesResponse, error) {
	return &weftv1.ListVMPropertiesResponse{}, nil
}

func (m *mockWeftClient) SetVMProperty(_ context.Context, _ *weftv1.SetVMPropertyRequest, _ ...grpc.CallOption) (*weftv1.SetVMPropertyResponse, error) {
	return &weftv1.SetVMPropertyResponse{}, nil
}

func (m *mockWeftClient) DeleteVMProperty(_ context.Context, _ *weftv1.DeleteVMPropertyRequest, _ ...grpc.CallOption) (*weftv1.DeleteVMPropertyResponse, error) {
	return &weftv1.DeleteVMPropertyResponse{}, nil
}

// --- UEFI vars ---

func (m *mockWeftClient) ListUEFIVars(_ context.Context, _ *weftv1.ListUEFIVarsRequest, _ ...grpc.CallOption) (*weftv1.ListUEFIVarsResponse, error) {
	return &weftv1.ListUEFIVarsResponse{}, nil
}

func (m *mockWeftClient) SetUEFIVar(_ context.Context, _ *weftv1.SetUEFIVarRequest, _ ...grpc.CallOption) (*weftv1.SetUEFIVarResponse, error) {
	return &weftv1.SetUEFIVarResponse{}, nil
}

func (m *mockWeftClient) DeleteUEFIVar(_ context.Context, _ *weftv1.DeleteUEFIVarRequest, _ ...grpc.CallOption) (*weftv1.DeleteUEFIVarResponse, error) {
	return &weftv1.DeleteUEFIVarResponse{}, nil
}

// --- Per-VM SSH keys ---

func (m *mockWeftClient) ListVMSSHKeys(_ context.Context, _ *weftv1.ListVMSSHKeysRequest, _ ...grpc.CallOption) (*weftv1.ListVMSSHKeysResponse, error) {
	return &weftv1.ListVMSSHKeysResponse{}, nil
}

func (m *mockWeftClient) AddVMSSHKey(_ context.Context, _ *weftv1.AddVMSSHKeyRequest, _ ...grpc.CallOption) (*weftv1.AddVMSSHKeyResponse, error) {
	return &weftv1.AddVMSSHKeyResponse{}, nil
}

func (m *mockWeftClient) RemoveVMSSHKey(_ context.Context, _ *weftv1.RemoveVMSSHKeyRequest, _ ...grpc.CallOption) (*weftv1.RemoveVMSSHKeyResponse, error) {
	return &weftv1.RemoveVMSSHKeyResponse{}, nil
}

// newTestClient wraps a mockWeftClient in a weftClient ready to pass as meta.
func newTestClient(mock *mockWeftClient) *weftClient {
	return &weftClient{vms: mock}
}
