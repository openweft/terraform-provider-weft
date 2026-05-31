# Releasing `terraform-provider-weft`

This document is the operator runbook for cutting a release of
`terraform-provider-weft` such that it lands on `registry.terraform.io`.
Read once end-to-end before tagging anything; the GPG step in particular
has a one-time setup that's annoying to redo.

## Architecture

```
   tag push v0.x.y on main
            │
            ▼
   .github/workflows/release.yml      ← workflow_dispatch + tag-only
            │
            ▼
   goreleaser-action@v6 + .goreleaser.yml
            │
            ▼
   binaries × 4 OS × 3 arches            ──→ .zip + .zip.sig
   + terraform-registry-manifest.json    ──→ uploaded to GitHub Release
   + SHA256SUMS                          ──→ GPG-detached signed
            │
            ▼
   registry.terraform.io polls the release endpoint,
   verifies the GPG signature against the registered key,
   indexes the new version.
```

## One-time setup (per maintainer)

These steps run **once** per person/CI environment that will produce a
release. Skip if `gpg --list-secret-keys | grep -A 1 terraform-provider-weft`
already shows a key.

### 1. Generate a GPG keypair

```sh
gpg --full-generate-key
# Pick:
#   - (1) RSA and RSA
#   - 4096 bits
#   - 0 (no expiration; if you set one, you'll have to rotate the key
#     on the registry too — keep it simple for the first release)
#   - Real name:  openweft
#   - Email:      tannevaled@users.noreply.github.com   (or your release
#                 identity — must match the git author you tag from)
#   - Comment:    terraform-provider-weft signing key
```

Confirm the key landed:

```sh
gpg --list-secret-keys --keyid-format=long
```

The output looks like:

```
sec   rsa4096/ABCDEF0123456789 2026-05-30 [SC]
      <fingerprint>
uid   openweft (terraform-provider-weft signing key) <…>
ssb   rsa4096/0123456789ABCDEF 2026-05-30 [E]
```

Note the **40-character fingerprint** (second line above) and the
**16-char keyid** (after the `/`). Both go into the GitHub repo secrets.

### 2. Export the secrets to GitHub

Export the private key + ownertrust + (a fresh) fingerprint and copy them
into the openweft/terraform-provider-weft repo settings as secrets:

```sh
gpg --armor --export-secret-keys <fingerprint> | pbcopy   # → secret GPG_PRIVATE_KEY
gpg --fingerprint <fingerprint> | grep -E "[0-9A-F]{4}( [0-9A-F]{4}){9}" | tr -d ' '
                                                          # → secret GPG_FINGERPRINT
```

GitHub repo: Settings → Secrets and variables → Actions → New repository
secret. Add `GPG_PRIVATE_KEY` and `GPG_FINGERPRINT`.

The release workflow imports the key on every run via the `Set up GPG`
step (defined in `.github/workflows/release.yml`); without those two
secrets the goreleaser sign step fails.

### 3. Register the public key on registry.terraform.io

```sh
gpg --armor --export <fingerprint>      # public block, paste into UI
```

In the Terraform Registry, the openweft namespace owner (org admin) adds
the public block under Settings → GPG keys. The registry uses this
public key to verify `SHA256SUMS.sig` on every release ingest. Without
the key being registered, the release looks valid in GitHub but the
registry refuses to index it.

## Cutting a release

### Pre-flight (do every time)

```sh
# 1. Up to date with main + clean tree
git switch main && git pull
git status   # should be clean

# 2. Regenerate the docs (catches any stale schema → docs drift)
task docs
git status   # if docs/ changed, commit + push BEFORE tagging

# 3. Verify the build green locally
go vet ./...
go test -race ./...
goreleaser release --snapshot --clean   # dry run, produces dist/

# 4. Acceptance suite against a live weft daemon — catches resource
#    regressions (schema drift, plan-time crashes, broken Read paths)
#    that unit tests can't see.
export WEFT_SOCKET=unix:///tmp/acc-weft/weft.sock
task acceptance
```

Before tagging, run `task acceptance` against a live weft daemon and
check no resource regressions surface. If `release --snapshot` succeeds
and the acceptance suite is green, the tag-driven release will too.

### Tag + push

```sh
TAG=v0.1.0
git tag -a "$TAG" -m "$TAG"
git push origin "$TAG"
```

The push triggers `.github/workflows/release.yml`. Watch the run:

```sh
gh run watch
```

Expected duration: ~5-10 minutes (depending on goreleaser cache hit).
The workflow creates a **draft** GitHub release; the operator promotes
it manually (you check the assets look right, then click Publish).

### Verify the registry ingested

```sh
curl -s https://registry.terraform.io/v1/providers/openweft/weft/versions \
    | jq '.versions[] | .version' | head -5
```

Should show `"v0.1.0"` within ~5 minutes of the GitHub release going
public. If it doesn't:

- Re-check the GPG signature is detached and points at the right key
- Re-check the public key is registered under the openweft namespace
- Check the `terraform-registry-manifest.json` carries protocol 6.0

## Patch releases

Same flow, bump the tag (`v0.1.1`, `v0.1.2`, …). goreleaser's
`changelog: use: github-native` will pick up commits since the previous
tag automatically.

## Major / breaking releases

When the schema changes incompatibly (an attribute renamed, a
RequiresReplace dropped, a resource removed):

1. Bump the major (`v1.0.0` → `v2.0.0`).
2. Add a `CHANGELOG.md` note under `## Breaking changes` calling out
   the operator-visible diff.
3. Consider `moved` blocks in the schema if a resource has been renamed
   in-place — Terraform's `moved` syntax lets operators upgrade
   without state surgery.
4. Test against a real state file before publishing — `terraform plan`
   against a state created by the previous version should not require
   destroy/recreate unless intentional.

## Yanking a release

If a release ships broken:

1. Edit the GitHub release → check "This is a pre-release" to demote it.
   The registry won't auto-yank but operators visiting the release page
   see it's flagged.
2. Cut a fix release with the next patch version (`v0.1.1`).
3. Optionally, delete the tag from the registry via support ticket
   (HashiCorp doesn't expose a self-service yank API today).

Don't delete the git tag — git tags are immutable contracts in Go
modules, and operators may have already pinned. Yank the release, ship
a fix, move on.

## Troubleshooting

**goreleaser fails with "no GPG keys found"** — secret didn't import.
Check the `Set up GPG` step output; the workflow expects
`GPG_PRIVATE_KEY` to be the full armored block (including the
`-----BEGIN PGP PRIVATE KEY BLOCK-----` headers).

**goreleaser fails with "sign command failed"** — fingerprint mismatch.
The fingerprint stored in `GPG_FINGERPRINT` must match the key being
imported. Re-export both from the same `gpg --list-secret-keys`
session.

**Release builds but registry indexes nothing** — public key not
registered, or the release is still in `draft`. The registry only
indexes published releases.

**`darwin-arm64` build fails with "cgo not supported"** — we set
`CGO_ENABLED=0` in `.goreleaser.yml`'s `builds.env`. If a future
dependency requires cgo, the matrix needs darwin-arm64 build on a
macOS arm64 runner; goreleaser supports this via `goos`/`goarch`
filters + multi-runner setup. Out of scope here, flag it when needed.
