# Vault-Based Release Automation - Setup & Testing Guide

This document explains the Vault-integrated release automation setup and how to test it.

## What Was Changed

### 1. Updated Release Workflow ([.github/workflows/release.yml](.github/workflows/release.yml))

**Key Changes:**
- ✅ Added `id-token: write` permission for Vault authentication
- ✅ Changed runner to use `group: prod` (your self-hosted runner group)
- ✅ Integrated `QumulusTechnology/vault-setup-action@v2` to fetch GPG keys from Vault
- ✅ Fetches both `private-key` and `public-key` from `secret/data/qcp/global/automation-user-gpg-key`
- ✅ Removed dependency on GitHub Secrets for GPG keys (now uses Vault)

**Vault Path:**
```
secret/data/qcp/global/automation-user-gpg-key
├── private-key  → GPG_PRIVATE_KEY
└── public-key   → GPG_PUBLIC_KEY
```

### 2. Created Test Workflow ([.github/workflows/test-release.yml](.github/workflows/test-release.yml))

A manual workflow for testing the release process without creating actual releases:

**Features:**
- ✅ Manual trigger via GitHub Actions UI
- ✅ Dry-run mode (default) - builds everything but doesn't create a release
- ✅ Fetches GPG keys from Vault and verifies them
- ✅ Imports GPG key and validates fingerprint
- ✅ Runs GoReleaser in snapshot mode
- ✅ Verifies GPG signatures on checksums
- ✅ Uploads artifacts for inspection
- ✅ Creates detailed test summary

## Prerequisites

Before testing, ensure these GitHub repository secrets are configured:

1. **AWS_ACCOUNT_DATA** - AWS credentials for Vault access
2. **VAULT_ADDR** - Vault server address (e.g., `https://vault.cloudportal.app`)

These should already be configured if you're using the `QumulusTechnology/vault-setup-action` elsewhere.

## Testing the Setup

### Step 1: Push Changes to GitHub

```bash
# The changes are already committed locally
git push origin main
```

### Step 2: Run the Test Workflow

1. Go to your repository on GitHub:
   ```
   https://github.com/dniasoff/terraform-provider-strato-danny/actions
   ```

2. Click on the **"Test Release"** workflow in the left sidebar

3. Click the **"Run workflow"** button (top right)

4. Select:
   - Branch: `main`
   - Dry run mode: `true` (default)

5. Click **"Run workflow"**

### Step 3: Monitor the Test Run

The workflow will:

1. **Checkout** the code
2. **Set up Go** from go.mod
3. **Fetch GPG keys from Vault** and verify they're not empty
4. **Import GPG key** and display fingerprint/key ID
5. **Create a temporary test tag** (e.g., `v0.0.0-test-1234567890`)
6. **Run GoReleaser** in snapshot mode (no actual release)
7. **List generated artifacts** (binaries, checksums, signatures)
8. **Verify GPG signature** on the checksums file
9. **Upload artifacts** as GitHub Action artifacts

### Step 4: Review Test Results

After the workflow completes, check:

1. **Workflow Summary** - Shows GPG fingerprint and artifacts
2. **Artifacts** - Download `test-release-artifacts.zip` to inspect binaries
3. **Logs** - Review each step for any errors

**Expected Output:**
```
✅ GPG_PRIVATE_KEY fetched successfully
✅ GPG_PUBLIC_KEY fetched successfully
✅ GPG key imported successfully
Fingerprint: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
✅ Signature verification successful!
```

## Troubleshooting

### Issue: Vault Authentication Fails

**Error:** `Error authenticating to Vault`

**Solutions:**
- Verify `AWS_ACCOUNT_DATA` secret is correct
- Verify `VAULT_ADDR` secret points to correct Vault server
- Check that the runner has access to AWS credentials
- Ensure the Vault path is correct: `secret/data/qcp/global/automation-user-gpg-key`

### Issue: GPG Key Import Fails

**Error:** `gpg: invalid armor: line 1: no header`

**Solutions:**
- Verify the Vault keys are properly formatted (ASCII-armored)
- Check that `private-key` contains the full GPG private key block
- Ensure no extra whitespace or line breaks were added

### Issue: Runner Not Found

**Error:** `No runner matching label 'group: prod' found`

**Solutions:**
- Verify the self-hosted runner group name is correct
- Check if runners in the 'prod' group are online
- Alternative: Change to `runs-on: ubuntu-latest` for testing (but won't have Vault access)

### Issue: Signature Verification Fails

**Error:** `gpg: BAD signature`

**Solutions:**
- Ensure the GPG key in Vault matches the signing key
- Verify the key hasn't expired
- Check that passphrase is empty (or correct if set)

## Creating an Actual Release

Once testing is successful, create a real release:

### Option 1: Using the Test Workflow (Non-Dry-Run)

1. Go to **Actions** → **Test Release**
2. Click **Run workflow**
3. Set **dry_run** to `false`
4. This will create a real GitHub release

### Option 2: Push a Version Tag (Production Method)

```bash
# Create and push a version tag
git tag v0.1.0
git push origin v0.1.0
```

This will trigger the main release workflow automatically.

## Verifying the Release

After a release is created:

1. **Check GitHub Releases**:
   ```
   https://github.com/dniasoff/terraform-provider-strato-danny/releases
   ```

2. **Verify Release Assets**:
   - ✅ Binaries for all platforms (`.zip` files)
   - ✅ `terraform-provider-strato_{VERSION}_SHA256SUMS`
   - ✅ `terraform-provider-strato_{VERSION}_SHA256SUMS.sig`
   - ✅ `terraform-provider-strato_{VERSION}_manifest.json`

3. **Verify GPG Signature Locally**:
   ```bash
   # Download the public key from Vault and import
   # Then verify the signature
   gpg --verify terraform-provider-strato_*_SHA256SUMS.sig \
                terraform-provider-strato_*_SHA256SUMS
   ```

## Next Steps After Successful Release

1. **Register GPG Public Key on Terraform Registry**:
   - Get the public key from Vault: `https://vault.cloudportal.app/ui/vault/secrets/secret/show/qcp/global/automation-user-gpg-key`
   - Go to https://registry.terraform.io → User Settings → Signing Keys
   - Add the public key

2. **Publish Provider to Terraform Registry**:
   - Follow steps in [TERRAFORM_REGISTRY_SETUP.md](TERRAFORM_REGISTRY_SETUP.md)
   - Section: "Step 6: Publish Your Provider to Terraform Registry"

3. **Monitor Future Releases**:
   - All future releases are automatic when you push version tags
   - Terraform Registry webhook will auto-ingest new versions

## Workflow Files Summary

### [.github/workflows/release.yml](.github/workflows/release.yml)
- **Trigger**: Push of tags matching `v*`
- **Purpose**: Production releases to GitHub and Terraform Registry
- **GPG Source**: Vault
- **Runner**: `group: prod`

### [.github/workflows/test-release.yml](.github/workflows/test-release.yml)
- **Trigger**: Manual (workflow_dispatch)
- **Purpose**: Testing release process without publishing
- **GPG Source**: Vault
- **Runner**: `group: prod`
- **Default Mode**: Dry-run (snapshot build)

## Security Notes

- ✅ GPG keys are stored securely in Vault (not in GitHub Secrets)
- ✅ Vault authentication uses AWS OIDC (no long-lived credentials)
- ✅ GPG private key is never exposed in logs
- ✅ Artifacts are signed to ensure integrity
- ✅ Only runners in the 'prod' group can access Vault secrets

## Support

If you encounter issues:

1. Check the workflow logs for specific error messages
2. Review Vault permissions for the automation user
3. Verify runner group configuration
4. Test Vault connectivity from the runner manually

## Documentation References

- [TERRAFORM_REGISTRY_SETUP.md](TERRAFORM_REGISTRY_SETUP.md) - Complete Terraform Registry publishing guide
- [CLAUDE.md](CLAUDE.md) - Provider architecture and development guide
- [Terraform Registry Publishing Docs](https://developer.hashicorp.com/terraform/registry/providers/publishing)
