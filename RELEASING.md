# Release Process

This document outlines the release process for this project, which uses automated workflows to handle versioning, building, and publishing releases.

## Release Cadence

gen-mcp minor versions (that is, 0.**5**.0 versus 0.**4**.x) are typically released every 6 weeks. In order to be practical and flexible, we will consider a more rapid minor release if any of the following conditions are true:

- A significant set of well-tested features introduced since the last release.
- User demand for rapid adoption of a specific feature.

Additionally, we will consider delaying a minor release if no significant features have landed during the normal 6-week release cycle.

gen-mcp patch versions (for example, 0.5.**2** versus 0.5.**1**) are released as often as weekly. Maintainers decide whether a patch release is called for based on community input. A patch release may bypass this cadence if circumstances warrant.

## Overview

The release process involves three main workflows:
1. **Pre-release creation** - Automatically triggered when pushing to release branches
2. **Release publishing** - Manually triggered to publish the final release
3. **Nightly releases** - Automatically created daily if there are unreleased commits

## Release Steps

### 1. Ensure CHANGELOG is Up to Date

Before cutting a release, ensure that `CHANGELOG.md` contains a section for your release version in the following format:

```markdown
## [vX.Y.Z]
### Added
- New feature descriptions
### Changed  
- Changes to existing functionality
### Fixed
- Bug fixes
```

The CHANGELOG must contain a section matching the exact version you plan to release (e.g., `## [v1.2.3]`).

### 2. Cut a New Release Branch

Create and push a new release branch following the naming convention `release/vX.Y`:

```bash
git checkout -b release/v1.2
git push -u origin release/v1.2
```

**Important**: The branch name must match the pattern `release/vX.Y` (e.g., `release/v1.2`, not `release/v1.2.0`).

### 3. Automated Pre-release Creation

When you push to a release branch, the **Create Pre-Release** workflow automatically:
- Determines the next patch version (Z) by checking existing releases for that X.Y version
- Creates a draft pre-release with the version `vX.Y.Z-prerelease`
- Builds and uploads binaries for multiple platforms (Linux, Windows, macOS on amd64 and arm64)
- Extracts changelog content for the release notes

If there are no new commits since the last release on that branch, no pre-release will be created.

### 4. Verify the Pre-release

Check the GitHub Releases page to verify:
- The pre-release was created successfully
- All expected binary assets were uploaded
- The release notes contain the correct changelog content
- The version number is correct

### 5. Publish the Final Release

When ready to publish the final release, manually trigger the **Publish Release** workflow:

1. Go to Actions → Publish Release
2. Enter the release branch name (e.g., `release/v1.2`)
3. Run the workflow

The workflow will:
- Validate the release branch format
- Determine the final release version (vX.Y.Z)
- Validate that CHANGELOG.md contains the required version section
- Run all tests
- Convert the pre-release to a final release (or create a new one if no pre-release exists). This involves creating a git tag.
- Build and upload fresh binaries
- Mark the release as the latest

## Z-Stream Releases (Patch Releases)

For patch releases on existing release branches:

1. **Backport commits** to the release branch:
   ```bash
   git checkout release/v1.2
   git cherry-pick <commit-hash>
   git push
   ```

2. **Automatic pre-release**: Pushing new commits to a release branch automatically creates a new pre-release with an incremented Z version (e.g., if v1.2.0 exists, the next will be v1.2.1-prerelease).

3. **Publish the patch release**: Re-run the **Publish Release** workflow with the same release branch to publish the new patch version.

## Nightly Releases

The **Nightly Release** workflow runs automatically at 02:00 UTC daily and:
- Checks if there are unreleased commits since the last stable release
- Creates a nightly release with format `nightly-YYYYMMDD-<short-commit>`
- Uses the "Unreleased" section from CHANGELOG.md for release notes
- Skips creation if no new commits exist or a nightly already exists for the current commit

## Version Numbering

- **Release branches**: `release/vX.Y` (e.g., `release/v1.2`)
- **Pre-releases**: `vX.Y.Z-prerelease` (e.g., `v1.2.0-prerelease`)
- **Final releases**: `vX.Y.Z` (e.g., `v1.2.0`)
- **Nightly releases**: `nightly-YYYYMMDD-<commit>` (e.g., `nightly-20240315-abc123f`)

The Z version (patch number) is automatically determined by checking existing releases for the X.Y version and incrementing accordingly.

## Verify the signed binary

You can cryptographically verify that the downloaded binaries (`.zip` files) are authentic and have not been tampered with. This process uses `cosign` to check the signature and certificate, which were generated securely during our automated build process.

### Step 1: Install Cosign

You'll need the `cosign` command-line tool. Please see the [Official Cosign Installation Guide](https://docs.sigstore.dev/cosign/installation/).

### Step 2: Verify the Binary

1.  From the release page, download three files for your platform:
    * The binary archive (e.g., `genmcp-linux-amd64.zip`)
    * The certificate (e.g., `genmcp-linux-amd64.zip.pem`)
    * The signature (e.g., `genmcp-linux-amd64.zip.sig`)

2.  Run the `cosign verify-blob` command in your terminal.

    **Example (for the Linux amd64 CLI):**
    ```bash
      cosign verify-blob \
         --certificate genmcp-linux-amd64.zip.pem \
         --signature genmcp-linux-amd64.zip.sig \
         --certificate-identity-regexp "https://github.com/genmcp/gen-mcp/.*" \
         --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
         genmcp-linux-amd64.zip
   ```

3.  If the signature is valid, `cosign` will contact the public Sigstore transparency log and print:
    ```
    Verified OK
    ```
