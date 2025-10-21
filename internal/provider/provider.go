// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QumulusTechnology/strato-project/sdk"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure stratoProvider satisfies various provider interfaces.
var _ provider.Provider = &stratoProvider{}
var _ provider.ProviderWithFunctions = &stratoProvider{}
var _ provider.ProviderWithEphemeralResources = &stratoProvider{}

// stratoProvider defines the provider implementation.
type stratoProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// stratoProviderModel describes the provider data model.
type stratoProviderModel struct {
	BearerToken types.String `tfsdk:"bearer_token"`
}

func (p *stratoProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "strato"
	resp.Version = p.version
}

func (p *stratoProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"bearer_token": schema.StringAttribute{
				MarkdownDescription: "Bearer token for the Strato API",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *stratoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data stratoProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.BearerToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("bearer_token"),
			"Unknown bearer token",
			"The provider cannot create the Strato API client as there is an unknown configuration value for the bearer token.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	debugOption := sdk.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		var msg strings.Builder
		msg.WriteString("HTTP Request:\n")
		msg.WriteString("  Method: " + req.Method + "\n")
		msg.WriteString("  URL: " + req.URL.String() + "\n")
		msg.WriteString("  Host: " + req.Host + "\n")
		msg.WriteString("  Path: " + req.URL.Path + "\n")
		msg.WriteString("  Remote Addr: " + req.RemoteAddr + "\n")

		if req.URL.RawQuery != "" {
			msg.WriteString("  Query: " + req.URL.RawQuery + "\n")
		}

		if req.ContentLength > 0 {
			msg.WriteString("  Content Length: " + strconv.FormatInt(req.ContentLength, 10) + "\n")
		}

		if userAgent := req.Header.Get("User-Agent"); userAgent != "" {
			msg.WriteString("  User-Agent: " + userAgent + "\n")
		}

		if contentType := req.Header.Get("Content-Type"); contentType != "" {
			msg.WriteString("  Content-Type: " + contentType + "\n")
		}

		if _, _, ok := req.BasicAuth(); ok {
			msg.WriteString("  Has Auth: true\n")
		}

		if authHeader := req.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer") {
				msg.WriteString("  Auth Type: bearer\n")
			} else if strings.HasPrefix(strings.ToLower(authHeader), "basic") {
				msg.WriteString("  Auth Type: basic\n")
			}
		}

		if req.Body != nil && req.ContentLength != 0 {
			body, err := io.ReadAll(req.Body)
			if err == nil && len(body) > 0 {
				// Reset the body for subsequent reads
				req.Body = io.NopCloser(strings.NewReader(string(body)))

				bodyStr := string(body)
				if len(bodyStr) > 1000 {
					bodyStr = bodyStr[:1000] + "... [truncated]"
				}
				msg.WriteString("  Body: " + bodyStr + "\n")
			}
		}

		tflog.Debug(ctx, msg.String())

		return nil
	})
	authClientOption := sdk.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+data.BearerToken.ValueString())
		return nil
	})
	client, err := sdk.NewClientWithResponses("https://api.cloudportal.run/strato/", authClientOption, debugOption)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Strato client",
			"An unexpected error occurred when creating the Strato client: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *stratoProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
		NewNodePoolResource,
	}
}

func (p *stratoProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return nil
	// return []func() ephemeral.EphemeralResource{
	// 	NewExampleEphemeralResource,
	// }
}

func (p *stratoProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewClusterDataSource,
		NewNodePoolDataSource,
	}
}

func (p *stratoProvider) Functions(ctx context.Context) []func() function.Function {
	return nil
	// return []func() function.Function{
	// 	NewExampleFunction,
	// }
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &stratoProvider{
			version: version,
		}
	}
}
