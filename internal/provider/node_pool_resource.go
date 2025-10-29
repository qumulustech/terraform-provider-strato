// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/QumulusTechnology/strato-project/sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NodePoolResource{}
var _ resource.ResourceWithImportState = &NodePoolResource{}

func NewNodePoolResource() resource.Resource {
	return &NodePoolResource{}
}

// NodePoolResource defines the resource implementation.
type NodePoolResource struct {
	client *sdk.ClientWithResponses
}

// NodePoolResourceModel describes the resource data model.
type NodePoolResourceModel struct {
	Id         types.String `tfsdk:"id"`
	ClusterId  types.String `tfsdk:"cluster_id"`
	Name       types.String `tfsdk:"name"`
	FullName   types.String `tfsdk:"full_name"`
	FlavorId   types.String `tfsdk:"flavor_id"`
	NetworkId  types.String `tfsdk:"network_id"`
	KeyPair    types.String `tfsdk:"key_pair"`
	VolumeSize types.Int64  `tfsdk:"volume_size"`
	NodeCount  types.Int64  `tfsdk:"node_count"`

	// optional attributes
	// AutoScale    types.Bool  `tfsdk:"auto_scale"`
	// MinNodeCount types.Int64 `tfsdk:"min_node_count"`
	// MaxNodeCount types.Int64 `tfsdk:"max_node_count"`

	// computed attributes
	ServerGroupId types.String `tfsdk:"server_group_id"`
	IsDefault     types.Bool   `tfsdk:"is_default"`
	Status        types.String `tfsdk:"status"`
	LastErrorId   types.String `tfsdk:"last_error_id"`
	CreatedAt     types.Int64  `tfsdk:"created_at"`
	UpdatedAt     types.Int64  `tfsdk:"updated_at"`
	Deleted       types.Bool   `tfsdk:"deleted"`
	DeletedAt     types.Int64  `tfsdk:"deleted_at"`
}

func (r *NodePoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_pool"
}

func (r *NodePoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Node pool resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Node pool identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// required attributes
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "Cluster identifier",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Node pool name (NOTE: will be normalized by the API, use the `full_name` attribute to see the actual name)",
				Required:            true,
			},
			"flavor_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack flavor id",
				Required:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack network id",
				Required:            true,
			},
			"key_pair": schema.StringAttribute{
				MarkdownDescription: "OpenStack keypair",
				Required:            true,
			},
			"volume_size": schema.Int64Attribute{
				MarkdownDescription: "Node worker volume size in GB",
				Required:            true,
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "Number of node workers",
				Required:            true,
			},

			// optional attributes
			// "auto_scale": schema.BoolAttribute{
			// 	MarkdownDescription: "Node pool auto scale",
			// 	Optional:            true,
			// 	Computed:            true,
			// },
			// "min_node_count": schema.Int64Attribute{
			// 	MarkdownDescription: "Minimum number of node workers",
			// 	Optional:            true,
			// 	Computed:            true,
			// },
			// "max_node_count": schema.Int64Attribute{
			// 	MarkdownDescription: "Maximum number of node workers",
			// 	Optional:            true,
			// 	Computed:            true,
			// },

			// computed attributes
			"full_name": schema.StringAttribute{
				MarkdownDescription: "Node pool full name as normalized by the API (includes prefix)",
				Computed:            true,
			},
			"server_group_id": schema.StringAttribute{
				MarkdownDescription: "Server group identifier",
				Computed:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Is default node pool",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Node pool status",
				Computed:            true,
			},
			"last_error_id": schema.StringAttribute{
				MarkdownDescription: "Node pool last error id",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				MarkdownDescription: "Node pool created at",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				MarkdownDescription: "Node pool updated at",
				Computed:            true,
			},
			"deleted": schema.BoolAttribute{
				MarkdownDescription: "Node pool deleted",
				Computed:            true,
			},
			"deleted_at": schema.Int64Attribute{
				MarkdownDescription: "Node pool deleted at",
				Computed:            true,
				Optional:            true,
			},
		},
	}
}

func (r *NodePoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *NodePoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NodePoolResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsUnknown() || data.Name.IsNull() || data.Name.ValueString() == "" {
		resp.Diagnostics.AddError("Missing Required Field", "The 'name' field is required")
		return
	}

	// Build request body
	body := sdk.CreateNodepoolJSONRequestBody{
		Name:       data.Name.ValueString(),
		FlavorID:   data.FlavorId.ValueString(),
		NetworkID:  data.NetworkId.ValueString(),
		Keypair:    data.KeyPair.ValueString(),
		VolumeSize: data.VolumeSize.ValueInt64(),
		NodeCount:  data.NodeCount.ValueInt64(),
	}

	// if !data.AutoScale.IsUnknown() && !data.AutoScale.IsNull() {
	// 	body.AutoScale = &[]bool{data.AutoScale.ValueBool()}[0]
	// }
	// if !data.MinNodeCount.IsUnknown() && !data.MinNodeCount.IsNull() {
	// 	body.MinNodeCount = &[]int64{data.MinNodeCount.ValueInt64()}[0]
	// }
	// if !data.MaxNodeCount.IsUnknown() && !data.MaxNodeCount.IsNull() {
	// 	body.MaxNodeCount = &[]int64{data.MaxNodeCount.ValueInt64()}[0]
	// }
	// Note: Labels are not supported in CreateNodePoolRequestBody

	createResult, err := r.client.CreateNodepoolWithResponse(ctx, data.ClusterId.ValueString(), &sdk.CreateNodepoolParams{}, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create node pool", err.Error())
		return
	}
	if createResult.StatusCode() != 200 {
		resp.Diagnostics.AddError("Unable to create node pool", fmt.Sprintf("http response status code: %d", createResult.StatusCode()))
		return
	}
	if createResult.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to create node pool", "node pool is nil")
		return
	}

	// Wait for node pool to be ready - calculate timeout based on node count (10-20 minutes)
	attempts := calculateRetryAttempts(data.NodeCount.ValueInt64())

	err = retry.Do(
		func() error {
			if err := r.readNodePool(ctx, data.ClusterId.ValueString(), createResult.JSON200.Id, &data); err != nil {
				return err
			}
			switch data.Status.ValueString() {
			case string(sdk.NODE_POOL_STATUS_CREATING):
				return fmt.Errorf("node pool is creating")
			case string(sdk.NODE_POOL_STATUS_RESIZING):
				return fmt.Errorf("node pool is resizing")
			case string(sdk.NODE_POOL_STATUS_ERROR):
				return fmt.Errorf("node pool is in error state")
			case string(sdk.NODE_POOL_STATUS_DELETING):
				return fmt.Errorf("node pool is in deleting state")
			case string(sdk.NODE_POOL_STATUS_READY):
				return nil
			default:
				return fmt.Errorf("node pool is in unknown state")
			}
		},
		retry.Context(ctx),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(10*time.Second),
		retry.Attempts(attempts),
		retry.RetryIf(func(err error) bool {
			return err != nil && err.Error() == "node pool is creating"
		}),
	)

	if err != nil {
		resp.Diagnostics.AddError("Unable to create node pool", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NodePoolResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readNodePool(ctx, data.ClusterId.ValueString(), data.Id.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Unable to read node pool", err.Error())
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NodePoolResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsUnknown() || data.Name.IsNull() || data.Name.ValueString() == "" {
		resp.Diagnostics.AddError("Missing Required Field", "The 'name' field is required")
		return
	}

	// Build update request body
	body := sdk.UpdateNodepoolJSONRequestBody{
		NodeCount: data.NodeCount.ValueInt64(),
	}

	// if !data.FlavorId.IsUnknown() && !data.FlavorId.IsNull() {
	// 	body.FlavorID = &[]string{data.FlavorId.ValueString()}[0]
	// }
	// if !data.VolumeSize.IsUnknown() && !data.VolumeSize.IsNull() {
	// 	body.VolumeSize = &[]int64{data.VolumeSize.ValueInt64()}[0]
	// }

	// if !data.AutoScale.IsUnknown() && !data.AutoScale.IsNull() {
	// 	body.AutoScale = &[]bool{data.AutoScale.ValueBool()}[0]
	// }
	// if !data.MinNodeCount.IsUnknown() && !data.MinNodeCount.IsNull() {
	// 	body.MinNodeCount = &[]int64{data.MinNodeCount.ValueInt64()}[0]
	// }
	// if !data.MaxNodeCount.IsUnknown() && !data.MaxNodeCount.IsNull() {
	// 	body.MaxNodeCount = &[]int64{data.MaxNodeCount.ValueInt64()}[0]
	// }

	updateResult, err := r.client.UpdateNodepoolWithResponse(ctx, data.ClusterId.ValueString(), data.Id.ValueString(), &sdk.UpdateNodepoolParams{}, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update node pool", err.Error())
		return
	}
	if updateResult.StatusCode() != 200 {
		resp.Diagnostics.AddError("Unable to update node pool", fmt.Sprintf("http response status code: %d", updateResult.StatusCode()))
		return
	}
	if updateResult.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to update node pool", "node pool is nil")
		return
	}

	// Calculate timeout based on new node count (10-20 minutes)
	attempts := calculateRetryAttempts(data.NodeCount.ValueInt64())

	err = retry.Do(
		func() error {
			if err := r.readNodePool(ctx, data.ClusterId.ValueString(), data.Id.ValueString(), &data); err != nil {
				return err
			}
			switch data.Status.ValueString() {
			case string(sdk.NODE_POOL_STATUS_CREATING):
				return fmt.Errorf("node pool is creating")
			case string(sdk.NODE_POOL_STATUS_RESIZING):
				return fmt.Errorf("node pool is resizing")
			case string(sdk.NODE_POOL_STATUS_ERROR):
				return fmt.Errorf("node pool is in error state")
			case string(sdk.NODE_POOL_STATUS_DELETING):
				return fmt.Errorf("node pool is in deleting state")
			case string(sdk.NODE_POOL_STATUS_READY):
				return nil
			default:
				return fmt.Errorf("node pool is in unknown state")
			}
		},
		retry.Context(ctx),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(10*time.Second),
		retry.Attempts(attempts),
		retry.RetryIf(func(err error) bool {
			return err != nil && err.Error() == "node pool is resizing"
		}),
	)

	if err != nil {
		resp.Diagnostics.AddError("Unable to update node pool", err.Error())
		return
	}

	if err := r.readNodePool(ctx, data.ClusterId.ValueString(), data.Id.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Unable to update node pool", err.Error())
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NodePoolResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteResult, err := r.client.DeleteNodepoolWithResponse(ctx, data.ClusterId.ValueString(), data.Id.ValueString(), &sdk.DeleteNodepoolParams{}, sdk.DeleteNodepoolJSONRequestBody{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete node pool", err.Error())
		return
	}
	if deleteResult.StatusCode() >= 400 {
		resp.Diagnostics.AddError("Unable to delete node pool", fmt.Sprintf("http response status code: %d", deleteResult.StatusCode()))
		return
	}
	if deleteResult.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to delete node pool", "node pool is nil")
		return
	}

	// Wait for node pool to be deleted - use 10 minute timeout (independent of node count)
	err = retry.Do(
		func() error {
			showResult, err := r.client.ShowNodePoolWithResponse(ctx, data.ClusterId.ValueString(), data.Id.ValueString(), &sdk.ShowNodePoolParams{})
			if err != nil {
				return err
			}
			if showResult.StatusCode() == 404 {
				return nil
			}
			if showResult.StatusCode() != 200 {
				return fmt.Errorf("http response status code: %d", showResult.StatusCode())
			}
			if showResult.JSON200 == nil {
				return fmt.Errorf("node pool is nil")
			}
			if showResult.JSON200.Deleted {
				return nil
			}
			switch showResult.JSON200.Status {
			case string(sdk.NODE_POOL_STATUS_CREATING):
				return fmt.Errorf("node pool is creating")
			case string(sdk.NODE_POOL_STATUS_RESIZING):
				return fmt.Errorf("node pool is resizing")
			case string(sdk.NODE_POOL_STATUS_ERROR):
				return fmt.Errorf("node pool is in error state")
			case string(sdk.NODE_POOL_STATUS_DELETING):
				return fmt.Errorf("node pool is in deleting state")
			case string(sdk.NODE_POOL_STATUS_READY):
				return nil
			default:
				return fmt.Errorf("node pool is in unknown state")
			}
		},
		retry.Context(ctx),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(10*time.Second),
		retry.Attempts(60), // 10 minutes
		retry.RetryIf(func(err error) bool {
			return err != nil && err.Error() == "node pool is in deleting state"
		}),
	)

	if err != nil {
		resp.Diagnostics.AddError("Unable to delete node pool", err.Error())
		return
	}
}

func (r *NodePoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NodePoolResource) readNodePool(ctx context.Context, clusterId, nodePoolId string, data *NodePoolResourceModel) error {
	params := &sdk.ShowNodePoolParams{}
	result, err := r.client.ShowNodePoolWithResponse(ctx, clusterId, nodePoolId, params)
	if err != nil {
		return err
	}
	if result.StatusCode() != 200 {
		return fmt.Errorf("http response status code: %d", result.StatusCode())
	}
	if result.JSON200 == nil {
		return fmt.Errorf("node pool is nil")
	}

	nodePool := result.JSON200
	data.Id = types.StringValue(nodePool.Id)
	// data.Name = types.StringValue(nodePool.Name)
	data.FullName = types.StringValue(nodePool.Name)
	data.ServerGroupId = types.StringValue(nodePool.ServerGroupID)
	data.FlavorId = types.StringValue(nodePool.FlavorID)
	data.NetworkId = types.StringValue(nodePool.NetworkID)
	data.KeyPair = types.StringValue(nodePool.KeyPair)
	data.VolumeSize = types.Int64Value(nodePool.VolumeSize)
	data.IsDefault = types.BoolValue(nodePool.IsDefault)
	data.NodeCount = types.Int64Value(nodePool.NodeCount)

	// data.MaxNodeCount = types.Int64Value(nodePool.MaxNodeCount)
	// data.MinNodeCount = types.Int64Value(nodePool.MinNodeCount)
	// data.AutoScale = types.BoolValue(nodePool.AutoScale)

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

	return nil
}
