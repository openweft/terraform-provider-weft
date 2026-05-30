# Migration: `terraform-plugin-sdk/v2` → `terraform-plugin-framework`

## Status (as of v0.2.0-dev)

| Resource / data source | Backend | Status |
|---|---|---|
| `weft_instance` | **framework** | migrated |
| `weft_image` | **framework** | migrated |
| `weft_image_patch` | **framework** | migrated |
| `weft_images` | **framework** | migrated |
| `weft_endpoint` | **framework** | migrated |
| `weft_keypair` | **framework** | migrated |
| `weft_deployment` | **framework** | migrated |
| data `weft_config` | **framework** | migrated |

Both backends are served from one provider binary via
`terraform-plugin-mux/tf6muxserver`. HashiCorp's recommended path:
operators see no behavioural change — the protocol version (6) is the
same; only the in-process implementation differs.

## Why migrate

`terraform-plugin-sdk/v2` is the legacy SDK. HashiCorp recommends
`terraform-plugin-framework` for all new providers and for production-
ready migrations. Concrete wins relevant to weft:

- **Strong typing** — schema → Go struct mapping is generated from the
  schema declaration; no more `.(string)` / `.([]interface{})` type
  asserts. `weft_instance.disk.patch.add` was the worst offender in the
  sdk/v2 version (4 levels deep).
- **Plan modifiers** — `RequiresReplace`, `UseStateForUnknown`,
  `RequiresReplaceIfFunc` are first-class, declarative, and composable.
  In sdk/v2 the `ForceNew: true` toggle is binary and you can't condition it.
- **Validators** — per-attribute declarative validators in
  `terraform-plugin-framework-validators` (string format, numeric ranges,
  list size, mutually-exclusive, exactly-one-of). sdk/v2's `ValidateFunc`
  is per-attribute imperative; reuse across resources is awkward.
- **Ephemeral resources + write-only attrs** — only available in
  framework. Write-only is the right shape for `keypair.private_key` (we
  never want to put a private key in state).
- **Protocol 6** — required for some newer Terraform features (deferred
  actions, function provider, `moved` blocks for cross-provider rename).

## Migration pattern (one resource at a time)

For each resource currently in `internal/provider/resource_<thing>.go`:

1. Write `internal/provider/<thing>_resource.go` implementing
   `resource.Resource` (Metadata + Schema + Configure + Create + Read +
   Update + Delete) and `resource.ResourceWithImportState`.
2. Register it in `framework_provider.go`'s `Resources` method.
3. Remove the sdk/v2 entry from `provider.go`'s `ResourcesMap`.
4. Delete `resource_<thing>.go` and `resource_<thing>_test.go`.
5. Add a framework-style schema test in `<thing>_resource_test.go`
   (calls the resource's Schema method, asserts attribute set + types).
6. `go vet ./...`, `go test ./...`, `go build ./...` green.
7. Commit one-resource-per-commit so a partial migration can be
   bisected if a regression shows up.

## What's left

In rough order of payoff:

1. `weft_keypair` — simplest schema, validates the pattern for "small
   resource with a write-only-style secret field".
2. `weft_image` — straightforward CRUD over `PullImage` RPC.
3. `weft_image_patch` — variant of image with the patch ops.
4. `weft_endpoint` — config-bag resource, no RPC.
5. `weft_images` — bulk variant.
6. `weft_deployment` — composite that delegates to multiple instances.
7. data source `weft_config` — last because data sources have their own
   interface (`datasource.DataSource`), small wrinkle vs resources.

Once all are migrated, the sdk/v2 dependency can come out of `go.mod`
entirely and `tf5to6server.UpgradeServer` disappears from `main.go`.

## Operator-facing impact

Zero, by design. HCL syntax doesn't change. State file format doesn't
change (protocol 6 was already what the sdk/v2 server was being
upgraded to via `tf5to6server`). `terraform plan` against a migrated
resource on an existing state produces no diff.
