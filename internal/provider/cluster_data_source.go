// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/QumulusTechnology/strato-project/sdk"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ClusterDataSource{}

func NewClusterDataSource() datasource.DataSource {
	return &ClusterDataSource{}
}

// ClusterDataSource defines the data source implementation.
type ClusterDataSource struct {
	client *sdk.ClientWithResponses
}

// ClusterDataSourceModel describes the data source data model.
type ClusterDataSourceModel struct {
	Id                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	ClusterId             types.String `tfsdk:"cluster_id"`
	ProjectId             types.String `tfsdk:"project_id"`
	ControlPlaneName      types.String `tfsdk:"control_plane_name"`
	ControlPlaneNamespace types.String `tfsdk:"control_plane_namespace"`
	Keypair               types.String `tfsdk:"keypair"`
	Tags                  types.List   `tfsdk:"tags"`
	Status                types.String `tfsdk:"status"`
	Phase                 types.String `tfsdk:"phase"`
	LastErrorId           types.String `tfsdk:"last_error_id"`
	CreatedAt             types.Int64  `tfsdk:"created_at"`
	UpdatedAt             types.Int64  `tfsdk:"updated_at"`
	Deleted               types.Bool   `tfsdk:"deleted"`
	DeletedAt             types.Int64  `tfsdk:"deleted_at"`
}

func (d *ClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *ClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Cluster data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Cluster identifier",
				Required:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name",
				Computed:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack cluster id",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack project id",
				Computed:            true,
			},
			"control_plane_name": schema.StringAttribute{
				MarkdownDescription: "Cluster control plane name",
				Computed:            true,
			},
			"control_plane_namespace": schema.StringAttribute{
				MarkdownDescription: "Cluster control plane namespace",
				Computed:            true,
			},
			"keypair": schema.StringAttribute{
				MarkdownDescription: "OpenStack keypair",
				Computed:            true,
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Cluster tags",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Cluster status",
				Computed:            true,
			},
			"phase": schema.StringAttribute{
				MarkdownDescription: "Cluster phase",
				Computed:            true,
			},
			"last_error_id": schema.StringAttribute{
				MarkdownDescription: "Cluster last error id",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				MarkdownDescription: "Cluster created at",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				MarkdownDescription: "Cluster updated at",
				Computed:            true,
			},
			"deleted": schema.BoolAttribute{
				MarkdownDescription: "Cluster deleted",
				Computed:            true,
			},
			"deleted_at": schema.Int64Attribute{
				MarkdownDescription: "Cluster deleted at",
				Computed:            true,
				Optional:            true,
			},
		},
	}
}

func (d *ClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	showResult, err := d.client.ShowClusterWithResponse(ctx, data.Id.ValueString(), &sdk.ShowClusterParams{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read cluster", err.Error())
		return
	}
	if showResult.StatusCode() != 200 {
		resp.Diagnostics.AddError("Unable to read cluster", fmt.Sprintf("http response status code: %d", showResult.StatusCode()))
		return
	}
	cluster := showResult.JSON200
	if cluster == nil {
		resp.Diagnostics.AddError("Unable to read cluster", "cluster is nil")
		return
	}

	data.Id = types.StringValue(cluster.Id)
	data.Name = types.StringValue(cluster.Name)
	data.ClusterId = types.StringValue(cluster.ClusterID)
	data.ProjectId = types.StringValue(cluster.ProjectID)
	data.ControlPlaneName = types.StringValue(cluster.ControlPlaneName)
	data.ControlPlaneNamespace = types.StringValue(cluster.ControlPlaneNamespace)
	data.Keypair = types.StringValue(cluster.Keypair)
	if cluster.Tags != nil {
		listValues, diags := types.ListValueFrom(ctx, types.StringType, *cluster.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Tags = listValues
	} else {
		data.Tags = types.ListNull(types.StringType)
	}
	data.Status = types.StringValue(cluster.Status)
	data.Phase = types.StringValue(cluster.Phase)
	data.LastErrorId = types.StringValue(cluster.LastErrorID)
	data.CreatedAt = types.Int64Value(cluster.CreatedAt)
	data.UpdatedAt = types.Int64Value(cluster.UpdatedAt)
	data.Deleted = types.BoolValue(cluster.Deleted)
	if cluster.DeletedAt != nil {
		data.DeletedAt = types.Int64Value(*cluster.DeletedAt)
	} else {
		data.DeletedAt = types.Int64Null()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
