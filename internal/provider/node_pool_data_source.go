// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/QumulusTechnology/strato-project/sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &NodePoolDataSource{}

func NewNodePoolDataSource() datasource.DataSource {
	return &NodePoolDataSource{}
}

// NodePoolDataSource defines the data source implementation.
type NodePoolDataSource struct {
	client *sdk.ClientWithResponses
}

// ClusterDataSourceModel describes the data source data model.
type NodePoolDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	ServerGroupId types.String `tfsdk:"server_group_id"`
	ClusterId     types.String `tfsdk:"cluster_id"`
	Name          types.String `tfsdk:"name"`
	FlavorId      types.String `tfsdk:"flavor_id"`
	NetworkId     types.String `tfsdk:"network_id"`
	KeyPair       types.String `tfsdk:"key_pair"`
	VolumeSize    types.Int64  `tfsdk:"volume_size"`
	IsDefault     types.Bool   `tfsdk:"is_default"`
	NodeCount     types.Int64  `tfsdk:"node_count"`
	MaxNodeCount  types.Int64  `tfsdk:"max_node_count"`
	MinNodeCount  types.Int64  `tfsdk:"min_node_count"`
	AutoScale     types.Bool   `tfsdk:"auto_scale"`
	Status        types.String `tfsdk:"status"`
	LastErrorId   types.String `tfsdk:"last_error_id"`
	CreatedAt     types.Int64  `tfsdk:"created_at"`
	UpdatedAt     types.Int64  `tfsdk:"updated_at"`
	Deleted       types.Bool   `tfsdk:"deleted"`
	DeletedAt     types.Int64  `tfsdk:"deleted_at"`
}

func (d *NodePoolDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_pool"
}

func (d *NodePoolDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Node pool data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Node pool identifier",
				Required:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "Cluster identifier",
				Required:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "Node pool name",
				Computed:            true,
			},
			"server_group_id": schema.StringAttribute{
				MarkdownDescription: "Server group identifier",
				Computed:            true,
			},
			"flavor_id": schema.StringAttribute{
				MarkdownDescription: "Flavor identifier",
				Computed:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "Network identifier",
				Computed:            true,
			},
			"key_pair": schema.StringAttribute{
				MarkdownDescription: "Key pair identifier",
				Computed:            true,
			},
			"volume_size": schema.Int64Attribute{
				MarkdownDescription: "Volume size",
				Computed:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Is default",
				Computed:            true,
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "Node count",
				Computed:            true,
			},
			"max_node_count": schema.Int64Attribute{
				MarkdownDescription: "Max node count",
				Computed:            true,
			},
			"min_node_count": schema.Int64Attribute{
				MarkdownDescription: "Min node count",
				Computed:            true,
			},
			"auto_scale": schema.BoolAttribute{
				MarkdownDescription: "Auto scale",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status",
				Computed:            true,
			},
			"last_error_id": schema.StringAttribute{
				MarkdownDescription: "Last error identifier",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				MarkdownDescription: "Created at",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				MarkdownDescription: "Updated at",
				Computed:            true,
			},
			"deleted": schema.BoolAttribute{
				MarkdownDescription: "Deleted",
				Computed:            true,
			},
			"deleted_at": schema.Int64Attribute{
				MarkdownDescription: "Deleted at",
				Computed:            true,
				Optional:            true,
			},
		},
	}
}

func (d *NodePoolDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NodePoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NodePoolDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	showResult, err := d.client.ShowNodePoolWithResponse(ctx, data.ClusterId.ValueString(), data.Id.ValueString(), &sdk.ShowNodePoolParams{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read node pool", err.Error())
		return
	}
	if showResult.StatusCode() != 200 {
		resp.Diagnostics.AddError("Unable to read node pool", fmt.Sprintf("http response status code: %d", showResult.StatusCode()))
		return
	}
	nodePool := showResult.JSON200
	if nodePool == nil {
		resp.Diagnostics.AddError("Unable to read node pool", "node pool is nil")
		return
	}

	data.Id = types.StringValue(nodePool.Id)
	data.Name = types.StringValue(nodePool.Name)
	data.ServerGroupId = types.StringValue(nodePool.ServerGroupID)
	data.FlavorId = types.StringValue(nodePool.FlavorID)
	data.NetworkId = types.StringValue(nodePool.NetworkID)
	data.KeyPair = types.StringValue(nodePool.KeyPair)
	data.VolumeSize = types.Int64Value(nodePool.VolumeSize)
	data.IsDefault = types.BoolValue(nodePool.IsDefault)
	data.NodeCount = types.Int64Value(nodePool.NodeCount)
	data.MaxNodeCount = types.Int64Value(nodePool.MaxNodeCount)
	data.MinNodeCount = types.Int64Value(nodePool.MinNodeCount)
	data.AutoScale = types.BoolValue(nodePool.AutoScale)
	data.Status = types.StringValue(nodePool.Status)
	data.LastErrorId = types.StringValue(nodePool.LastErrorID)
	data.CreatedAt = types.Int64Value(nodePool.CreatedAt)
	data.UpdatedAt = types.Int64Value(nodePool.UpdatedAt)
	data.Deleted = types.BoolValue(nodePool.Deleted)
	if nodePool.DeletedAt != nil {
		data.DeletedAt = types.Int64Value(*nodePool.DeletedAt)
	} else {
		data.DeletedAt = types.Int64Null()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
