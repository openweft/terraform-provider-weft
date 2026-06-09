# Gap Analysis — terraform-provider-weft vs weft-proto

Snapshot date: 2026-05-31. Based on `weft-proto/{weft,agent,guest,introspect}.proto`
and `internal/provider/framework_provider.go` Resources().

This is the second pass — the first one (2026-05-30) was written when only 7/70
RPCs were covered and recorded a `WeftAgent` total of 70. Since then both sides
have moved: 6 new resources landed (`weft_host`, `weft_network`, `weft_volume`,
`weft_security_group`, `weft_tenant`, `weft_volume_snapshot`), and `weft-proto`
grew the `VolumeSnapshot` quadruplet plus a handful of other RPCs. The current
WeftAgent surface is **98 RPCs**, not 70.

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
| `weft_host` | `RegisterHost` (Create/Update) · `GetHost` (Read) · `DeleteHost` (Delete) |
| `weft_network` | `CreateNetwork` · `ListNetworks` (Read) · `RenameNetwork` · `SetNetworkDNS` · `SetNetworkDefaultSecurityGroups` · `DeleteNetwork` |
| `weft_volume` | `CreateVolume` · `ListVolumes` (Read) · `RenameVolume` · `ResizeVolume` · `DeleteVolume` |
| `weft_volume_snapshot` | `CreateVolumeSnapshot` · `ListVolumeSnapshots` (Read) · `DeleteVolumeSnapshot` |
| `weft_security_group` | `CreateSecurityGroup` · `ListSecurityGroups` (Read) · `RenameSecurityGroup` · `SetSecurityGroupDescription` · `SetSecurityGroupRules` · `DeleteSecurityGroup` |
| `weft_tenant` | `CreateTenant` · `ListTenants` (Read) · `DeleteTenant` |
| `weft_config` (data) | _none — parses local HCL via `openweft/weft-hcl`_ |

**Used WeftAgent RPCs (33):** `ProvisionVM`, `DeprovisionVM`, `VMStatus`,
`PullImage`, `PullImages`, `PatchImage`, `ListImages`, `RegisterHost`,
`GetHost`, `DeleteHost`, `CreateNetwork`, `ListNetworks`, `RenameNetwork`,
`SetNetworkDNS`, `SetNetworkDefaultSecurityGroups`, `DeleteNetwork`,
`CreateVolume`, `ListVolumes`, `RenameVolume`, `ResizeVolume`, `DeleteVolume`,
`CreateVolumeSnapshot`, `ListVolumeSnapshots`, `DeleteVolumeSnapshot`,
`CreateSecurityGroup`, `ListSecurityGroups`, `RenameSecurityGroup`,
`SetSecurityGroupDescription`, `SetSecurityGroupRules`, `DeleteSecurityGroup`,
`CreateTenant`, `ListTenants`, `DeleteTenant`.

## Unmapped RPCs

### WeftAgent — VM lifecycle

| RPC | Suggested Terraform surface |
|---|---|
| `ListVMs` | `data.weft_vms` (data source) |
| `StartVM` | **imperative — Terraform-incompatible**; expose `desired_state = "running" \| "stopped"` on `weft_instance` or run via `terraform_data` provisioner |
| `StopVM` | **imperative — Terraform-incompatible**; same note |
| `CreateVM` | _superset of `ProvisionVM`_ — fold into `weft_instance` once framework migration lands (project, scheduling_rule, network attributes) |
| `DeleteVM` | _covered by `weft_instance` delete_ (DeprovisionVM is the post-CreateVM equivalent) |
| `WaitVM` | **imperative — Terraform-incompatible**; wrap as `data.weft_vm_state` poller, or absorb into `weft_instance` create timeout |
| `RegisterMicroVM` | _internal scheduling primitive — keep off Terraform surface_ |
| `VMTimings` | `data.weft_vm_timings` |
| `VMLogs` | `data.weft_vm_logs` |
| `CleanImages` | **imperative — Terraform-incompatible**; `weft_image_clean` with `triggers`, or pure `terraform_data` |

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

(CreateNetwork / DeleteNetwork / RenameNetwork / SetNetworkDNS /
SetNetworkDefaultSecurityGroups already covered by `weft_resource_network`.)

### WeftAgent — Security groups

| RPC | Suggested resource |
|---|---|
| `ListSecurityGroups` | `data.weft_security_groups` |

(CreateSecurityGroup / DeleteSecurityGroup / RenameSecurityGroup /
SetSecurityGroupDescription / SetSecurityGroupRules already covered by
`weft_security_group`.)

### WeftAgent — Volumes

| RPC | Suggested resource |
|---|---|
| `ListVolumes` | `data.weft_volumes` |
| `AttachVolume` / `DetachVolume` | `weft_volume_attachment` |

(CreateVolume / DeleteVolume / RenameVolume / ResizeVolume already covered by
`weft_volume`.)

### WeftAgent — Volume snapshots

| RPC | Suggested resource |
|---|---|
| `ListVolumeSnapshots` | `data.weft_volume_snapshots` |
| `RestoreVolumeSnapshot` | **imperative — Terraform-incompatible** (mutates an existing volume in-place); expose via `terraform_data` + provisioner, or as a one-shot `null_resource`-style trigger |

(CreateVolumeSnapshot / DeleteVolumeSnapshot already covered by
`weft_volume_snapshot`.)

### WeftAgent — Events & infra

| RPC | Suggested resource |
|---|---|
| `WatchEvents` | **streaming — Terraform-incompatible**; consume via separate sidecar or `external` data source one-shot |
| `RenderNATSAuthorization` | `data.weft_nats_authorization` |

### WeftAgent — Hosts (compute pool)

| RPC | Suggested resource |
|---|---|
| `ListHosts` | `data.weft_hosts` |
| `HeartbeatHost` | _internal agent loop — not a Terraform concern_ |
| `SetHostState` | _Update on `weft_host.state`_ (drain/cordon semantics) |
| `SetHostLabels` | _Update on `weft_host.labels`_ |

(RegisterHost / GetHost / DeleteHost already covered by `weft_host`.)

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
| `AddTenantAdmin` / `RemoveTenantAdmin` | `weft_tenant_admin` |
| `AddTenantMember` / `RemoveTenantMember` | `weft_tenant_member` |

(CreateTenant / DeleteTenant already covered by `weft_tenant`.)

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

- **WeftAgent service:** 33 / 98 RPCs covered (≈ **34%**).
- **All proto services combined:** 33 / 104 RPCs covered (≈ **32%**).
  - WeftAgent: 98 RPCs
  - AgentDispatch: 1 RPC (internal)
  - AgentControlPlane: 3 RPCs (internal)
  - GuestPodPlane: 1 RPC (internal)
  - Introspect: 1 RPC (debug)

### Terraform-incompatible RPCs (flagged explicitly)

These RPCs are imperative actions or streams — they don't have stateful
identity, so they can't be the primary subject of a Terraform resource. They
are deliberately excluded from coverage denominators.

| RPC | Reason |
|---|---|
| `StartVM` | imperative (state transition, not identity) |
| `StopVM` | imperative |
| `WaitVM` | imperative (poll) |
| `CleanImages` | imperative (GC) |
| `RestoreVolumeSnapshot` | imperative (in-place mutation of an existing volume) |
| `HeartbeatHost` | internal agent liveness loop |
| `RegisterMicroVM` | internal scheduling primitive |
| `WatchEvents` | streaming |
| `AgentDispatch.Connect` | streaming, internal |
| `AgentControlPlane.AttachDrivers` | streaming, internal |
| `AgentControlPlane.RegisterAgent` / `Heartbeat` | internal bootstrap/liveness |
| `GuestPodPlane.Attach` | streaming, guest-side |

If we restrict the denominator to **stateful, Terraform-shaped RPCs** (WeftAgent
only, excluding the 9 WeftAgent RPCs flagged above — `StartVM`, `StopVM`,
`WaitVM`, `CleanImages`, `RestoreVolumeSnapshot`, `HeartbeatHost`,
`RegisterMicroVM`, `WatchEvents`, and treating `CreateVM`/`DeleteVM` as
covered-by `ProvisionVM`/`DeprovisionVM`):

- **Stateful denominator:** 98 − 9 − 2 = **87 RPCs**.
- **Stateful coverage:** 33 / 87 ≈ **38%**.

The remaining unmapped categories — by RPC count — are: VM-properties + UEFI +
SSH keys (9), Projects (7), Floating IPs (5), Users (5), Tenant
membership/admin (4), Quotas (4), Flavors (4), Scripts (4), Shares (4),
Volume attach/detach (2), Host list/state/labels (3), List* data sources for
already-mapped resources (5). There is **no SchedulingRule service** in the
proto today — `scheduling_rule` is a free-form label on `CreateVMRequest`, not
a CRUD-shaped resource, so `weft_scheduling_rule` is deliberately not on the
roadmap.
