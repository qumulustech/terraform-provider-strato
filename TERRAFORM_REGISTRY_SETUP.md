# Terraform Registry Publishing Setup Guide

This guide walks you through the process of publishing the `terraform-provider-strato` to the Terraform Registry.

## Prerequisites Checklist

- [ ] Public GitHub repository (current: `dniasoff/terraform-provider-strato-danny`)
- [ ] Repository follows naming convention: `terraform-provider-{NAME}` ✅
- [ ] GitHub account with owner/admin access to the repository
- [ ] GPG keypair for signing releases
- [ ] Terraform Registry account (sign in at https://registry.terraform.io)

## Step 1: Generate GPG Key (If Needed)

If you don't already have a GPG key for signing releases:

```bash
# Generate a new GPG key (RSA or DSA, NOT ECC)
gpg --full-generate-key

# Follow the prompts:
# - Select: (1) RSA and RSA
# - Key size: 4096 bits
# - Expiration: 0 (does not expire) or set an appropriate expiration
# - Enter your name and email
# - Set a passphrase (you'll need this for GitHub Secrets)

# List your keys to get the Key ID
gpg --list-secret-keys --keyid-format=long

# Export your PUBLIC key (ASCII-armored format)
gpg --armor --export your-email@example.com > public-key.asc

# Export your PRIVATE key (for GitHub Secrets)
gpg --armor --export-secret-keys your-email@example.com > private-key.asc
```

**Important Notes:**
- Keep your private key (`private-key.asc`) secure - you'll upload it to GitHub Secrets
- The public key will be uploaded to the Terraform Registry
- Use RSA or DSA keys, NOT the default ECC type (not supported by Terraform Registry)

## Step 2: Configure GitHub Secrets

Add these secrets to your repository at `Settings > Secrets and variables > Actions`:

1. **GPG_PRIVATE_KEY**
   - Content: Your ASCII-armored private GPG key (contents of `private-key.asc`)
   - Copy the entire key including the `-----BEGIN PGP PRIVATE KEY BLOCK-----` and `-----END PGP PRIVATE KEY BLOCK-----` lines

2. **PASSPHRASE**
   - Content: The passphrase you set when creating the GPG key
   - If you didn't set a passphrase, you can leave this as an empty secret

To add secrets:
```
1. Go to: https://github.com/dniasoff/terraform-provider-strato-danny/settings/secrets/actions
2. Click "New repository secret"
3. Add GPG_PRIVATE_KEY with your private key content
4. Add PASSPHRASE with your GPG key passphrase
```

## Step 3: Register Your GPG Key with Terraform Registry

1. Sign in to the Terraform Registry at https://registry.terraform.io
2. Click your profile icon (top right) → **User Settings**
3. Select **Signing Keys** from the left sidebar
4. Click **Add a key**
5. Paste the contents of your `public-key.asc` (ASCII-armored public key)
6. Click **Add key**

## Step 4: Verify Repository Configuration

Your repository is already configured with the necessary files:

### ✅ `.goreleaser.yml`
- Configured to build for multiple platforms (Linux, macOS, Windows, FreeBSD)
- Includes the `terraform-registry-manifest.json` in releases
- Signs releases with GPG
- Creates proper archive format: `terraform-provider-strato_{VERSION}_{OS}_{ARCH}.zip`

### ✅ `terraform-registry-manifest.json`
- Specifies protocol version 6.0 (for Plugin Framework providers)
- Will be included in each release

### ✅ `.github/workflows/release.yml`
- Triggers on version tags (v*)
- Imports GPG key from secrets
- Runs GoReleaser to build and sign releases
- Creates GitHub releases automatically

## Step 5: Prepare Documentation (If Not Already Done)

The Terraform Registry automatically generates documentation from your code, but you should verify:

1. **Provider Documentation**: Check that `internal/provider/provider.go` has good descriptions
2. **Resource Documentation**: Verify each resource has clear `MarkdownDescription` fields
3. **Examples**: Ensure `examples/` directory has working examples for each resource

To generate and preview documentation locally:
```bash
make generate
# Documentation will be in the docs/ directory
```

You can preview how it will look on the Terraform Registry using:
https://registry.terraform.io/tools/doc-preview

## Step 6: Publish Your Provider to Terraform Registry

### 6.1: Create Your First Release

Create and push a version tag to trigger the release workflow:

```bash
# Ensure your code is committed
git add .
git commit -m "Prepare for initial release"

# Create a version tag (use semantic versioning with 'v' prefix)
git tag v1.0.0

# Push the tag to GitHub (this triggers the release workflow)
git push origin v1.0.0
```

The GitHub Action will:
1. Build binaries for all supported platforms
2. Sign the checksums with your GPG key
3. Create a GitHub release with all artifacts
4. Upload the `terraform-registry-manifest.json`

### 6.2: Register the Provider on Terraform Registry

After your first release is created on GitHub:

1. Go to https://registry.terraform.io
2. Click **Publish** (top right) → **Provider**
3. Select your GitHub organization: **dniasoff**
4. Select your repository: **terraform-provider-strato-danny**
5. Click **Publish Provider**

The Terraform Registry will:
- Verify the repository structure
- Download and verify the v1.0.0 release
- Check the GPG signature
- Install a webhook on your repository for future releases
- Generate documentation from your code

### 6.3: Verify Publication

After publishing, your provider will be available at:
```
https://registry.terraform.io/providers/dniasoff/strato/latest
```

Users can then use it in their Terraform configurations:

```hcl
terraform {
  required_providers {
    strato = {
      source  = "dniasoff/strato"
      version = "~> 1.0"
    }
  }
}

provider "strato" {
  bearer_token = var.strato_bearer_token
}
```

## Step 7: Future Releases

Once the provider is registered, releasing new versions is automatic:

```bash
# Make your changes and commit them
git add .
git commit -m "Add new feature"

# Create a new version tag
git tag v1.1.0

# Push the tag
git push origin v1.1.0
```

The GitHub Action will automatically:
1. Build and sign the release
2. Create a GitHub release
3. The Terraform Registry webhook will detect the new release and ingest it automatically (usually within minutes)

## Troubleshooting

### Release Action Fails

**GPG Key Issues:**
- Verify `GPG_PRIVATE_KEY` secret contains the complete private key with headers
- Verify `PASSPHRASE` secret matches your GPG key passphrase
- Ensure you used RSA/DSA key type (not ECC)

**Build Failures:**
- Check that tests pass: `make test`
- Verify Go version compatibility: `go version` should be >= 1.23

### Provider Not Appearing on Registry

**Webhook Issues:**
- Check that the webhook was created: Go to your repository → Settings → Webhooks
- Look for a webhook pointing to `registry.terraform.io`
- If missing, use the "Resync" button on your provider settings page in the Terraform Registry

**Version Not Detected:**
- Verify tag format: Must be `v{MAJOR}.{MINOR}.{PATCH}` (e.g., `v1.0.0`)
- Check that GitHub release was created successfully
- Verify all required assets are in the release:
  - Zip files for each platform
  - `terraform-provider-strato_{VERSION}_SHA256SUMS`
  - `terraform-provider-strato_{VERSION}_SHA256SUMS.sig`
  - `terraform-provider-strato_{VERSION}_manifest.json`

### Signature Verification Fails

- Ensure your public GPG key is registered in Terraform Registry
- Verify the key ID in the Registry matches your signing key
- Check that GoReleaser is using the correct GPG key (via `GPG_FINGERPRINT`)

## Repository Naming Consideration

**Current Repository Name:** `terraform-provider-strato-danny`

The Terraform Registry will use the repository name to determine the provider name. In your case:
- Provider name will be: `strato-danny` (not ideal)

**Recommended Actions:**

1. **Option A: Use as-is**
   - Provider will be accessible as `dniasoff/strato-danny`
   - Users will reference it: `source = "dniasoff/strato-danny"`

2. **Option B: Rename repository (Recommended)**
   - Rename to: `terraform-provider-strato`
   - Provider will be accessible as `dniasoff/strato`
   - More professional and cleaner naming
   - To rename: Go to repository Settings → General → Repository name

If you choose Option B, update references in:
- `go.mod` (module path)
- Documentation
- Examples

## Additional Resources

- [Terraform Registry Provider Publishing](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [GoReleaser Documentation](https://goreleaser.com/)
- [terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework) - Reference implementation
- Terraform Registry Support: terraform-registry@hashicorp.com

## Security Notes

- Never commit your GPG private key to the repository
- Keep your GPG passphrase secure
- GitHub Secrets are encrypted and only available to workflows
- Consider key rotation: If your GPG key is compromised, generate a new one and update both GitHub Secrets and Terraform Registry

## Monitoring

After publication, monitor:
- **Downloads**: Check usage statistics in the Terraform Registry dashboard
- **Issues**: GitHub Issues for bug reports and feature requests
- **Releases**: Ensure the webhook triggers properly for each new tag
- **Documentation**: Verify docs render correctly on the Registry

## Next Steps After Publication

1. Add a badge to your README:
   ```markdown
   [![Terraform Registry](https://img.shields.io/badge/terraform-registry-623CE4)](https://registry.terraform.io/providers/dniasoff/strato)
   ```

2. Update your [CLAUDE.md](CLAUDE.md) with the published provider information

3. Consider adding:
   - CHANGELOG.md for tracking version changes
   - More comprehensive examples
   - Integration tests with real Strato API (if possible)
   - Contributing guidelines for community contributions
