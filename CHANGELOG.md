# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to adhere to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-05-31

### Added

- Initial Terraform provider scaffold with build/test workflow, goreleaser config and registry manifest.
- Resources migrated to terraform-plugin-framework: `weft_instance` (muxed server), then remaining 6 + `weft_config` data source.
- New framework resource `weft_host` with acceptance test harness.
- Gap resources: `weft_network`, `weft_volume`, `weft_security_group`, `weft_tenant`.
- `weft_volume_snapshot` resource — CoW snapshot via weft-proto v0.2.0 RPCs.
- Acceptance scaffolds for all 12 remaining resources and the `weft_config` data source.
- Acceptance test for `weft_volume_snapshot` (Create + Import-by-uuid).
- `task acceptance` runbook and `RELEASING.md` pre-flight reminder.
- `examples/` directory and tfplugindocs scaffold; regenerated Terraform Registry docs.
- `RELEASING.md` runbook for cutting registry-grade releases.
- `GAPS.md` tracking parity against weft-proto.

### Changed

- Drop the sdk/v2 shell — provider is now framework-only on a single protocol.

### Fixed

- `mockWeftClient` extended with VolumeSnapshot RPCs to keep tests compiling.
