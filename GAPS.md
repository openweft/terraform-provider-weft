# Gap Analysis — terraform-provider-weft vs weft-proto

Snapshot date: 2026-05-30. Based on `weft-proto/{weft,agent,guest,introspect}.proto`
and `internal/provider/provider.go` ResourcesMap.

## Mapped resources

| Terraform resource | gRPC RPCs invoked (WeftAgent) |
|---|---|
| `weft_instance` | `ProvisionVM` (Create) · `VMStatus` (Read) · `DeprovisionVM` (Delete) |
| `weft_deployment` | _none — local-only naming-scope resource_ |
| `weft_image` | `PullImage` (Create) · `PatchImage` (Create, optional patch block) |
| `weft_image_patch` | `PatchImage` (Create) · `ListImages` (Create, when `images` omitted) |
| `weft_images` | `PullImages` (Create) |
| `weft_endpoint` | _none — local-only URL declaration_ |
| `weft_keypair` | _none — reads `<file_path>.pub` from disk_ |
| `weft_config` (data) | _none — parses local HCL via `openweft/hclconfig`_ |

**Used WeftAgent RPCs:** `ProvisionVM`, `DeprovisionVM`, `VMStatus`, `PullImage`,
`PullImages`, `PatchImage`, `ListImages` (7 of 70 total).

## Unmapped RPCs

### WeftAgent — VM lifecycle

| RPC | Suggested Terraform surface |
|---|---|
| `ListVMs` | `data.weft_vms` (data source) |
| `StartVM` | (action, not stateful — Terraform-incompatible; use `terraform_data` + provisioner, or expose `desired_state = "running" \| "stopped"` on `weft_instance`) |
| `StopVM` | (action, not stateful — Terraform-incompatible; same note) |
| `CreateVM` | _superset of `ProvisionVM`_ — fold into `weft_instance` once framework migration lands (project, scheduling_rule, network attributes) |
| `DeleteVM` | _covered by `weft_instance` delete_ (DeprovisionVM is the post-CreateVM equivalent) |
| `WaitVM` | (action — wrap as `data.weft_vm_state` poller, or absorb into `weft_instance` create timeout) |
| `RegisterMicroVM` | (internal scheduling primitive — keep off Terraform surface) |
| `VMTimings` | `data.weft_vm_timings` |
| `VMLogs` | `data.weft_vm_logs` |
| `CleanImages` | (action — `weft_image_clean` with `triggers` or pure `terraform_data`) |

### WeftAgent — Projects & ACL

| RPC | Suggested resource |
|---|---|
| `ListProjects` | `data.weft_projects` |
| `CreateProject` / `DeleteProject` | `weft_project` |
| `RenameProject` | _Update on `weft_project.name`_ |
| `AddProjectMember` / `RemoveProjectMember` | `weft_project_member` |
| `ListProjectMembers` | `data.weft_project_members` |

### WeftAgent — Users

| RPC | Suggested resource |
|---|---|
| `ListUsers` / `GetUser` / `Me` | `data.weft_users`, `data.weft_user`, `data.weft_me` |
| `SetUserDisplayName` | _Update on `weft_user.display_name`_ |
| `DeleteUser` | `weft_user` (Delete) |

### WeftAgent — Networks

| RPC | Suggested resource |
|---|---|
| `ListNetworks` | `data.weft_networks` |
| `CreateNetwork` / `DeleteNetwork` | `weft_network` |
| `RenameNetwork` | _Update on `weft_network.name`_ |
| `SetNetworkDNS` | _Update on `weft_network.dns`_ |
| `SetNetworkDefaultSecurityGroups` | _Update on `weft_network.default_security_groups`_ |

### WeftAgent — Security groups

| RPC | Suggested resource |
|---|---|
| `ListSecurityGroups` | `data.weft_security_groups` |
| `CreateSecurityGroup` / `DeleteSecurityGroup` | `weft_security_group` |
| `RenameSecurityGroup` / `SetSecurityGroupDescription` / `SetSecurityGroupRules` | _Update on `weft_security_group.{name,description,rules}`_ |

### WeftAgent — Volumes

| RPC | Suggested resource |
|---|---|
| `ListVolumes` | `data.weft_volumes` |
| `CreateVolume` / `DeleteVolume` | `weft_volume` |
| `RenameVolume` / `ResizeVolume` | _Update on `weft_volume.{name,size_gb}`_ |
| `AttachVolume` / `DetachVolume` | `weft_volume_attachment` |

### WeftAgent — Events & infra

| RPC | Suggested resource |
|---|---|
| `WatchEvents` | (streaming — Terraform-incompatible; consume via separate sidecar or `external` data source one-shot) |
| `RenderNATSAuthorization` | `data.weft_nats_authorization` |

### WeftAgent — Hosts (compute pool)

| RPC | Suggested resource |
|---|---|
| `RegisterHost` | `weft_host` (Create) |
| `ListHosts` / `GetHost` | `data.weft_hosts`, `data.weft_host` |
| `HeartbeatHost` | (internal agent loop — not a Terraform concern) |
| `SetHostState` | _Update on `weft_host.state`_ (drain/cordon semantics) |
| `SetHostLabels` | _Update on `weft_host.labels`_ |
| `DeleteHost` | `weft_host` (Delete) |

### WeftAgent — Shares

| RPC | Suggested resource |
|---|---|
| `ListShares` | `data.weft_shares` |
| `CreateShare` / `DeleteShare` | `weft_share` |
| `PublishShareToProject` | `weft_share_publication` |

### WeftAgent — Tenants

| RPC | Suggested resource |
|---|---|
| `ListTenants` | `data.weft_tenants` |
| `CreateTenant` / `DeleteTenant` | `weft_tenant` |
| `AddTenantAdmin` / `RemoveTenantAdmin` | `weft_tenant_admin` |
| `AddTenantMember` / `RemoveTenantMember` | `weft_tenant_member` |

### WeftAgent — Quotas

| RPC | Suggested resource |
|---|---|
| `GetTenantQuota` / `SetTenantQuota` | `weft_tenant_quota` |
| `GetProjectQuota` / `SetProjectQuota` | `weft_project_quota` |

### WeftAgent — Floating IPs

| RPC | Suggested resource |
|---|---|
| `ListFloatingIPs` | `data.weft_floating_ips` |
| `AllocateFloatingIP` / `ReleaseFloatingIP` | `weft_floating_ip` |
| `MapFloatingIP` / `UnmapFloatingIP` | `weft_floating_ip_attachment` |

### WeftAgent — Flavors

| RPC | Suggested resource |
|---|---|
| `ListFlavors` / `GetFlavor` | `data.weft_flavors`, `data.weft_flavor` |
| `SetFlavor` (upsert) / `DeleteFlavor` | `weft_flavor` |

### WeftAgent — Scripts

| RPC | Suggested resource |
|---|---|
| `ListScripts` / `GetScript` | `data.weft_scripts`, `data.weft_script` |
| `SetScript` (upsert) / `DeleteScript` | `weft_script` |

### WeftAgent — VM properties / UEFI / SSH keys

| RPC | Suggested resource |
|---|---|
| `ListVMProperties` | `data.weft_vm_properties` |
| `SetVMProperty` (upsert) / `DeleteVMProperty` | `weft_vm_property` |
| `ListUEFIVars` | `data.weft_uefi_vars` |
| `SetUEFIVar` (upsert) / `DeleteUEFIVar` | `weft_uefi_var` |
| `ListVMSSHKeys` | `data.weft_vm_ssh_keys` |
| `AddVMSSHKey` (idempotent on fingerprint) / `RemoveVMSSHKey` | `weft_vm_ssh_key` |

### Other services (out of Terraform scope)

| Service.RPC | Note |
|---|---|
| `AgentDispatch.Connect` | bidi-stream agent control — internal, not Terraform-mappable |
| `AgentControlPlane.RegisterAgent` | agent-side bootstrap — internal |
| `AgentControlPlane.Heartbeat` | agent-side liveness — internal |
| `AgentControlPlane.AttachDrivers` | bidi-stream driver attach — internal |
| `GuestPodPlane.Attach` | guest-side pod plane — runs inside microVM, not on the operator's machine |
| `Introspect.ListProcesses` | dev/debug RPC — could be `data.weft_processes` (low priority) |

## Coverage summary

- **WeftAgent service:** 7 / 70 RPCs covered (10%).
- **All proto services combined:** 7 / 76 RPCs covered (9%).
  - WeftAgent: 70 RPCs
  - AgentDispatch: 1 RPC (internal)
  - AgentControlPlane: 3 RPCs (internal)
  - GuestPodPlane: 1 RPC (internal)
  - Introspect: 1 RPC (debug)

If we restrict the denominator to **stateful, Terraform-shaped RPCs** (excluding
imperative actions `StartVM`/`StopVM`/`WaitVM`/`CleanImages`/`HeartbeatHost`,
streaming `WatchEvents`/`Connect`/`Attach*`, and pure-internal
`RegisterMicroVM`/`RegisterAgent`/`Heartbeat`):

- **Stateful surface:** 7 / ~58 RPCs covered (≈ 12%).

The biggest unmapped categories — by RPC count — are: Hosts (7), SecurityGroups (6),
Volumes (7), Tenants (7), Quotas (4), Networks (6), Projects (7), FloatingIPs (5),
Flavors (4), Scripts (4), VM properties + UEFI + SSH keys (9). These map cleanly
to a standard cloud-style resource set and should be the next milestone, ideally
on top of the in-progress terraform-plugin-framework migration.
