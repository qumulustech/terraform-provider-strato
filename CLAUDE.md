# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Terraform provider for Strato cloud services (OpenStack-based infrastructure) built using the **Terraform Plugin Framework** (not the deprecated SDK). The provider enables infrastructure-as-code management of Kubernetes clusters and node pools via the Strato API.

**Key Details:**
- Provider Type: `strato`
- API Endpoint: `https://api.cloudportal.run/strato/`
- External SDK: `github.com/QumulusTechnology/strato-project` (v1.0.11+)
- Authentication: Bearer token (sensitive)
- Go Version: 1.24.4+
- Terraform Version Support: >= 1.0

## Development Commands

### Building and Installing
```bash
make install          # Build and install provider to $GOPATH/bin
go install            # Alternative: direct install
make build            # Compile without installing
```

### Testing
```bash
make test             # Run unit tests (10 parallel, 120s timeout)
make testacc          # Run acceptance tests (requires TF_ACC=1, creates real resources)
```

### Code Quality
```bash
make fmt              # Format code with gofmt
make lint             # Run golangci-lint
make generate         # Generate documentation and run code generation
```

### Full Development Cycle
```bash
make                  # Runs: fmt, lint, install, generate (default target)
```

## Architecture

### Provider Structure

**Core Provider** ([internal/provider/provider.go:187](internal/provider/provider.go))
- Type: `stratoProvider`
- Configuration: Single `bearer_token` (sensitive, required)
- Client: Generated SDK client (`sdk.ClientWithResponses`)
- Custom request editors for:
  - **Authorization**: Injects Bearer token into all API requests
  - **Debug logging**: Logs HTTP request details (method, URL, headers, body)

**Resources:**
- `strato_cluster` - Kubernetes cluster management
- `strato_node_pool` - Node pool management within clusters

**Data Sources:**
- `strato_cluster` - Read cluster by ID
- `strato_node_pool` - Read node pool by cluster ID and pool ID

### Resource Implementation Pattern

All resources follow the Terraform Plugin Framework pattern:

1. **Metadata()** - Defines resource type name
2. **Schema()** - Defines attributes (required, optional, computed, sensitive)
3. **Configure()** - Receives provider client from provider.Configure()
4. **Create()** - Creates resource via API, polls until ready
5. **Read()** - Fetches current state from API
6. **Update()** - Updates existing resource, polls until ready
7. **Delete()** - Deletes resource, polls until deleted
8. **ImportState()** - Enables `terraform import` by ID

### Asynchronous Operation Handling

Both cluster and node pool resources use **status-based polling** with `avast/retry-go`:

```go
retry.Do(
    func() error {
        // Fetch current state from API
        // Check status field
        // Return error if not READY
    },
    retry.Delay(10*time.Second),
    retry.DelayType(retry.FixedDelay),
    retry.Attempts(30),  // 5 minutes total timeout
    retry.RetryIf(func(err error) bool {
        return err != nil && err.Error() == "resource is in progress"
    }),
)
```

This pattern is used in:
- Cluster creation (wait for status: READY)
- Node pool creation (wait for status: READY)
- Cluster deletion (wait for 404 or deleted status)
- Node pool updates (wait for resize to complete)

### Key Code Conventions

1. **Interface compliance checks** at package level:
   ```go
   var _ resource.Resource = &ClusterResource{}
   var _ resource.ResourceWithImportState = &ClusterResource{}
   ```

2. **Error handling** via diagnostics accumulation:
   ```go
   resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
   if resp.Diagnostics.HasError() {
       return
   }
   ```

3. **Type safety** with framework types:
   - Use `types.String`, `types.Int64`, `types.List`, etc.
   - Handle null/unknown values explicitly with `.IsNull()` / `.IsUnknown()`

4. **API interaction**:
   - Always check HTTP status codes explicitly
   - Validate response is not nil before dereferencing
   - Handle 404 as "resource deleted" in Read operations

5. **Null value handling** for optional attributes:
   ```go
   if !data.PrivateKubeAPI.IsNull() {
       requestData.PrivateKubeAPI = data.PrivateKubeAPI.ValueBoolPointer()
   }
   ```

## Resource Details

### Cluster Resource ([internal/provider/cluster_resource.go:543](internal/provider/cluster_resource.go))

**Required Attributes:**
- `cluster_id` - OpenStack cluster ID (string)
- `project_id` - OpenStack project ID (string)
- `name` - Cluster name (string)
- `keypair` - OpenStack keypair name (string)
- `network_id` - OpenStack network ID (string)
- `flavor_id` - OpenStack flavor ID for workers (string)
- `volume_size` - Worker node volume size in GB (int64)
- `node_count` - Number of worker nodes (int64)

**Optional Attributes:**
- `private_kube_api` - Disable public Kubernetes API access (bool)
- `tags` - List of cluster tags (list of strings)

**Computed Attributes:**
- `status`, `phase`, `control_plane_name`, `control_plane_namespace`
- `created_at`, `updated_at`, `deleted`, `deleted_at`
- `last_error_id`

**Update Behavior:** Only `node_count` can be updated (triggers node pool resize)

### Node Pool Resource ([internal/provider/node_pool_resource.go:511](internal/provider/node_pool_resource.go))

**Required Attributes:**
- `cluster_id` - Parent cluster ID (string)
- `name` - Node pool name (string, normalized by API)
- `flavor_id`, `network_id`, `key_pair` - OpenStack resource IDs
- `volume_size` - Node volume size in GB (int64)
- `node_count` - Number of nodes (int64)

**Computed Attributes:**
- `full_name` - API-normalized name (e.g., "my-pool" → "my-pool-abc123")
- `server_group_id`, `is_default`
- `status`, `created_at`, `updated_at`, etc.

**Update Behavior:** Only `node_count` can be updated

## Testing

### Test Files
- `internal/provider/*_test.go` - Resource/data source tests
- `internal/provider/provider_test.go` - Test infrastructure setup

### Running Tests
```bash
go test -v -cover -timeout=120s -parallel=10 ./...          # Unit tests
TF_ACC=1 go test -v -cover -timeout=120m ./...             # Acceptance tests
```

### Test Configuration
- Uses `terraform-plugin-testing` framework
- Protocol: protov6 (Plugin Protocol version 6)
- Provider factories configured in `provider_test.go`
- Acceptance tests require `TF_ACC=1` environment variable

## Code Generation

Run via `make generate` or `cd tools && go generate ./...`:

1. **Copyright headers** - `hashicorp/copywrite` tool
2. **Format Terraform examples** - `terraform fmt` on examples/
3. **Generate documentation** - `terraform-plugin-docs` tool
   - Reads from Schema definitions
   - Outputs to `docs/` directory
   - Uses examples from `examples/` directory

## Release Process

**Automated via GitHub Actions:**

1. Create and push a version tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions (`.github/workflows/release.yml`) triggers:
   - GoReleaser builds for multiple platforms
   - Generates checksums and GPG signatures
   - Creates GitHub release with binaries

**Supported Platforms:**
- Linux (amd64, 386, arm, arm64)
- macOS (amd64, arm64, excluding 386)
- Windows (amd64, 386, arm, arm64)
- FreeBSD (amd64, 386, arm, arm64)

**Release Requirements:**
- `GPG_FINGERPRINT` environment variable set for signing
- `terraform-registry-manifest.json` included in release

## CI/CD

### Test Workflow (`.github/workflows/test.yml`)
Runs on PR and push to main:
1. **Build** - Compile provider, run golangci-lint
2. **Generate** - Verify documentation is up-to-date
3. **Test** - Matrix test across Terraform 1.0-1.4

### Release Workflow (`.github/workflows/release.yml`)
Triggers on `v*` tags, uses GoReleaser for multi-platform builds

## Dependencies

**Critical Direct Dependencies:**
- `github.com/QumulusTechnology/strato-project` - Generated API client SDK
- `github.com/hashicorp/terraform-plugin-framework` - Provider framework
- `github.com/hashicorp/terraform-plugin-testing` - Test framework
- `github.com/avast/retry-go/v4` - Retry/polling logic

**Adding New Dependencies:**
```bash
go get github.com/author/package
go mod tidy
git commit go.mod go.sum
```

## Linting Configuration

**Enabled Linters** (`.golangci.yml`):
- copyloopvar, durationcheck, errcheck, forcetypeassert
- godot, ineffassign, makezero, misspell, nilerr
- predeclared, staticcheck, unconvert, unparam, unused

**Excluded Paths:**
- Generated code, examples, third-party dependencies

## Common Development Patterns

### Adding a New Resource

1. Create `internal/provider/my_resource.go`:
   ```go
   type MyResource struct {
       client *sdk.ClientWithResponses
   }

   func (r *MyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
       resp.TypeName = req.ProviderTypeName + "_my_resource"
   }

   func (r *MyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
       resp.Schema = schema.Schema{
           Attributes: map[string]schema.Attribute{
               "id": schema.StringAttribute{
                   Computed: true,
                   // ...
               },
           },
       }
   }

   // Implement Configure, Create, Read, Update, Delete, ImportState
   ```

2. Register in `internal/provider/provider.go`:
   ```go
   func (p *stratoProvider) Resources(ctx context.Context) []func() resource.Resource {
       return []func() resource.Resource{
           NewClusterResource,
           NewNodePoolResource,
           NewMyResource,  // Add this
       }
   }
   ```

3. Add tests in `internal/provider/my_resource_test.go`
4. Run `make generate` to update documentation
5. Add examples to `examples/resources/strato_my_resource/`

### Handling API Responses

```go
resp, err := r.client.GetSomethingWithResponse(ctx, id)
if err != nil {
    response.Diagnostics.AddError("API Error", fmt.Sprintf("Error: %s", err.Error()))
    return
}

if resp.StatusCode() == 404 {
    // Resource deleted, remove from state
    response.State.RemoveResource(ctx)
    return
}

if resp.StatusCode() != 200 || resp.JSON200 == nil {
    response.Diagnostics.AddError("Unexpected Response",
        fmt.Sprintf("Status: %d", resp.StatusCode()))
    return
}

// Use resp.JSON200.Field
```

### Polling for Resource Readiness

```go
err := retry.Do(
    func() error {
        resp, err := r.client.GetResourceWithResponse(ctx, data.ID.ValueString())
        if err != nil {
            return err
        }
        if resp.JSON200.Status != "READY" {
            return fmt.Errorf("resource not ready: %s", resp.JSON200.Status)
        }
        return nil
    },
    retry.Context(ctx),
    retry.Delay(10*time.Second),
    retry.DelayType(retry.FixedDelay),
    retry.Attempts(30),  // 5 minutes
    retry.RetryIf(func(err error) bool {
        return err != nil && strings.Contains(err.Error(), "not ready")
    }),
)
```

## Debugging

### Enable Debug Logging

The provider includes debug logging for HTTP requests. View logs with:

```bash
TF_LOG=DEBUG terraform apply
```

Debug output includes:
- HTTP method, URL, headers
- Request body (if present)
- Response status and body

### Local Development

1. Build and install locally:
   ```bash
   make install
   ```

2. Create a Terraform configuration with local provider:
   ```hcl
   terraform {
     required_providers {
       strato = {
         source = "hashicorp.com/local/strato"
       }
     }
   }
   ```

3. Run Terraform commands:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## API Client Updates

The SDK is generated externally in `github.com/QumulusTechnology/strato-project`. To update:

1. Update the SDK version in `go.mod`:
   ```bash
   go get github.com/QumulusTechnology/strato-project@v1.0.12
   go mod tidy
   ```

2. Update resource implementations if API changes
3. Run tests to verify compatibility
4. Update documentation with `make generate`
