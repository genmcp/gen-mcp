# Release Process

This project uses [GoReleaser](https://goreleaser.com/) with tag-based releases. Pushing a semver tag triggers the full release pipeline automatically.

## Creating a Release

1. Tag the commit you want to release:

   ```bash
   git tag v1.2.0
   git push origin v1.2.0
   ```

2. The [Release workflow](.github/workflows/release.yaml) will automatically:
   - Run all tests
   - Build binaries for all platforms (Linux, macOS, Windows on amd64 and arm64)
   - Sign all archives with cosign (keyless via Sigstore)
   - Generate checksums
   - Create the GitHub release with auto-generated release notes
   - Publish deb/rpm packages
   - Update the [Homebrew tap](https://github.com/genmcp/homebrew-genmcp)

## Pre-releases

Tags with a pre-release suffix (e.g., `-rc.1`, `-beta.1`) are automatically marked as pre-releases on GitHub. Homebrew is not updated for pre-releases.

```bash
git tag v1.2.0-rc.1
git push origin v1.2.0-rc.1
```

## Nightly Releases

The [Nightly workflow](.github/workflows/nightly.yaml) runs automatically Monday through Friday at 02:00 UTC. It creates a rolling `nightly` release with snapshot builds. You can also trigger it manually from the Actions tab.

Nightly builds skip Homebrew updates and use GoReleaser's snapshot mode.

## Manual Dispatch

Both the release and nightly workflows support manual triggering via `workflow_dispatch` in the GitHub Actions UI.

## Installing via Homebrew

```bash
brew tap genmcp/homebrew-genmcp
brew install --cask genmcp
brew install --cask genmcp-server
```

## Verifying Signed Binaries

All release archives are signed with [cosign](https://docs.sigstore.dev/cosign/overview/) using keyless signing. To verify:

1. Download the archive and its `.bundle` file from the release page.

2. Verify with cosign:

   ```bash
   cosign verify-blob \
     --bundle genmcp-linux-amd64.zip.bundle \
     --certificate-identity-regexp "https://github.com/genmcp/gen-mcp/.*" \
     --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
     genmcp-linux-amd64.zip
   ```

## Repository Secrets

The release workflow requires the following secrets:

- `HOMEBREW_TAP_GITHUB_TOKEN` — A GitHub PAT with `repo` scope for the `genmcp/homebrew-genmcp` repository, used to push Homebrew cask updates.
