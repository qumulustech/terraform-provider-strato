# Private Module Access Fix

## Problem

The workflow was failing with this error:
```
go: github.com/QumulusTechnology/strato-project@v1.0.11-0.20250902111707-9c4ce024546f:
invalid version: git ls-remote -q origin in /home/runner/go/pkg/mod/cache/vcs/...:
exit status 128:
	fatal: could not read Username for 'https://github.com': terminal prompts disabled
```

**Root Cause:** The `strato-project` dependency is a **private GitHub repository**, and Go was unable to authenticate when downloading the module.

## Solution

### 1. Fetch GitHub Token from Vault

Both workflows now fetch a GitHub access token from Vault:

```yaml
- name: Fetch secrets from Vault
  uses: QumulusTechnology/vault-setup-action@v2
  with:
    aws_account_data: ${{ secrets.AWS_ACCOUNT_DATA }}
    vault_addr: ${{ secrets.VAULT_ADDR }}
    platform: qcp
    secrets: |
      secret/data/qcp/global/automation-user-gpg-key private-key | GPG_PRIVATE_KEY;
      secret/data/qcp/global/automation-user-gpg-key public-key | GPG_PUBLIC_KEY;
      secret/data/qcp/global/automation-user-github-token token | GITHUB_ACCESS_TOKEN;
```

### 2. Configure Git for Private Modules

Added a step to configure Git to use the token for HTTPS authentication:

```yaml
- name: Configure Git for private modules
  run: |
    git config --global url."https://${GITHUB_ACCESS_TOKEN}@github.com/".insteadOf "https://github.com/"
```

This tells Git (and Go) to automatically use the token when accessing any GitHub repository over HTTPS.

### 3. Updated Test Workflow

Added verification to ensure the GitHub token is fetched:

```yaml
- name: Verify secrets were fetched
  run: |
    if [ -z "$GITHUB_ACCESS_TOKEN" ]; then
      echo "❌ ERROR: GITHUB_ACCESS_TOKEN is empty"
      exit 1
    else
      echo "✅ GITHUB_ACCESS_TOKEN fetched successfully"
      echo "Token length: ${#GITHUB_ACCESS_TOKEN} characters"
    fi
```

## How It Works

1. **Vault stores the GitHub token** at `secret/data/qcp/global/automation-user-github-token`
2. **Workflow fetches the token** using vault-setup-action
3. **Git is configured** to use the token for all GitHub HTTPS URLs
4. **Go module download** uses Git under the hood, so it automatically authenticates
5. **Private repositories** are now accessible during build

## Vault Configuration Required

The automation user's GitHub token must be stored in Vault:

**Path:** `secret/data/qcp/global/automation-user-github-token`

**Key:** `token`

**Value:** A GitHub Personal Access Token (PAT) or GitHub App token with:
- `repo` scope (for private repository access)
- Access to the `QumulusTechnology` organization
- Read access to the `strato-project` repository

## Testing

To verify the fix works:

1. **Push changes to GitHub:**
   ```bash
   git push origin main
   ```

2. **Run the test workflow:**
   - Go to: https://github.com/dniasoff/terraform-provider-strato-danny/actions
   - Select "Test Release" workflow
   - Click "Run workflow" → Keep dry_run as `true`

3. **Check the logs:**
   Look for these success indicators:
   ```
   ✅ GITHUB_ACCESS_TOKEN fetched successfully
   ✅ Go module download succeeds
   ✅ Build completes successfully
   ```

## Alternative Solutions (Not Used)

We chose the Git configuration approach, but here are alternatives:

### Option A: GOPRIVATE Environment Variable
```yaml
env:
  GOPRIVATE: github.com/QumulusTechnology/*
```
This tells Go the module is private, but you still need authentication.

### Option B: .netrc File
```yaml
- name: Configure .netrc
  run: |
    echo "machine github.com login ${GITHUB_ACCESS_TOKEN}" >> ~/.netrc
```
This also works but Git URL rewriting is cleaner.

### Option C: SSH Authentication
```yaml
- name: Configure SSH
  run: |
    git config --global url."git@github.com:".insteadOf "https://github.com/"
```
Would require SSH key management instead of token.

## Security Considerations

- ✅ Token is stored securely in Vault (not in repository)
- ✅ Token is fetched at runtime using OIDC authentication
- ✅ Token is never logged or exposed in output
- ✅ Git configuration is temporary (only exists during workflow run)
- ✅ Token has minimal required permissions (read access to private repos)

## Files Modified

1. **[.github/workflows/release.yml](.github/workflows/release.yml)**
   - Added GitHub token fetch
   - Added Git configuration step

2. **[.github/workflows/test-release.yml](.github/workflows/test-release.yml)**
   - Added GitHub token fetch
   - Added Git configuration step
   - Added token verification

3. **[VAULT_SETUP_TESTING.md](VAULT_SETUP_TESTING.md)**
   - Documented the additional Vault path
   - Added note about private module access

## Related Documentation

- [Go Modules - Private Repositories](https://golang.org/doc/faq#git_https)
- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [Git Configuration](https://git-scm.com/docs/git-config)
