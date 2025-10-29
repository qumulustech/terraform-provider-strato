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
var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

// ClusterResource defines the resource implementation.
type ClusterResource struct {
	client *sdk.ClientWithResponses
}

// ClusterResourceModel describes the resource data model.
type ClusterResourceModel struct {
	Id         types.String `tfsdk:"id"`
	ClusterId  types.String `tfsdk:"cluster_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	Name       types.String `tfsdk:"name"`
	Keypair    types.String `tfsdk:"keypair"`
	NetworkId  types.String `tfsdk:"network_id"`
	FlavorId   types.String `tfsdk:"flavor_id"`
	VolumeSize types.Int64  `tfsdk:"volume_size"`
	NodeCount  types.Int64  `tfsdk:"node_count"`

	// AutoScale      types.Bool  `tfsdk:"auto_scale"`
	// MinNodeCount   types.Int64 `tfsdk:"min_node_count"`
	// MaxNodeCount   types.Int64 `tfsdk:"max_node_count"`
	PrivateKubeAPI types.Bool `tfsdk:"private_kube_api"`
	Tags           types.List `tfsdk:"tags"`

	ControlPlaneName      types.String `tfsdk:"control_plane_name"`
	ControlPlaneNamespace types.String `tfsdk:"control_plane_namespace"`
	Status                types.String `tfsdk:"status"`
	Phase                 types.String `tfsdk:"phase"`
	LastErrorId           types.String `tfsdk:"last_error_id"`
	CreatedAt             types.Int64  `tfsdk:"created_at"`
	UpdatedAt             types.Int64  `tfsdk:"updated_at"`
	Deleted               types.Bool   `tfsdk:"deleted"`
	DeletedAt             types.Int64  `tfsdk:"deleted_at"`
}

func (r *ClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *ClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Cluster resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// required attributes
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack cluster id",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack project id",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name",
				Required:            true,
			},
			"keypair": schema.StringAttribute{
				MarkdownDescription: "OpenStack keypair",
				Required:            true,
			},

			// required attributes but not part of the output
			"network_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack network id",
				Required:            true,
				Computed:            false,
			},
			"flavor_id": schema.StringAttribute{
				MarkdownDescription: "OpenStack flavor id",
				Required:            true,
				Computed:            false,
			},
			"volume_size": schema.Int64Attribute{
				MarkdownDescription: "Node worker volume size in GB",
				Required:            true,
				Computed:            false,
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "Number of node workers",
				Required:            true,
				Computed:            false,
			},

			// optional attributes
			// "auto_scale": schema.BoolAttribute{
			// 	MarkdownDescription: "Cluster auto scale",
			// 	Optional:            true,
			// 	Computed:            false,
			// },
			// "min_node_count": schema.Int64Attribute{
			// 	MarkdownDescription: "Minimum number of node workers",
			// 	Optional:            true,
			// 	Computed:            false,
			// },
			// "max_node_count": schema.Int64Attribute{
			// 	MarkdownDescription: "Maximum number of node workers",
			// 	Optional:            true,
			// 	Computed:            false,
			// },
			"private_kube_api": schema.BoolAttribute{
				MarkdownDescription: "Set to true to disable public access to the kube API",
				Optional:            true,
				Computed:            false,
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Cluster tags",
				Optional:            true,
				Computed:            true,
			},

			// output-only attributes
			"control_plane_name": schema.StringAttribute{
				MarkdownDescription: "Cluster control plane name",
				Computed:            true,
			},
			"control_plane_namespace": schema.StringAttribute{
				MarkdownDescription: "Cluster control plane namespace",
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

func (r *ClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Can skip Authorization header since its handled by client options in provider configuration
	// But we must set X-OS-Cluster-ID and X-OS-Project-ID headers via params
	params := &sdk.CreateClusterParams{
		XOSClusterID: data.ClusterId.ValueString(),
		XOSProjectID: data.ProjectId.ValueString(),
	}
	body := sdk.CreateClusterJSONRequestBody{
		Name:       data.Name.ValueString(),
		NodeCount:  data.NodeCount.ValueInt64(),
		FlavorID:   data.FlavorId.ValueString(),
		NetworkID:  data.NetworkId.ValueString(),
		Keypair:    data.Keypair.ValueString(),
		VolumeSize: data.VolumeSize.ValueInt64(),
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
	if !data.Tags.IsUnknown() && !data.Tags.IsNull() {
		var tags []string
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body.Tags = &tags
	} else {
		body.Tags = &[]string{}
	}
	if !data.PrivateKubeAPI.IsUnknown() && !data.PrivateKubeAPI.IsNull() {
		body.PrivateKubeAPI = &[]bool{data.PrivateKubeAPI.ValueBool()}[0]
	}

	createResult, err := r.client.CreateClusterWithResponse(ctx, params, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create cluster", err.Error())
		return
	}
	if createResult.StatusCode() != 200 {
		// Try to extract error message from response body
		errorMsg := fmt.Sprintf("HTTP %d", createResult.StatusCode())
		if len(createResult.Body) > 0 {
			errorMsg = fmt.Sprintf("HTTP %d: %s", createResult.StatusCode(), string(createResult.Body))
		}
		resp.Diagnostics.AddError("Unable to create cluster", errorMsg)
		return
	}
	if createResult.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to create cluster", "cluster is nil")
		return
	}

	// Calculate timeout based on node count (10-20 minutes)
	attempts := calculateRetryAttempts(data.NodeCount.ValueInt64())

	err = retry.Do(
		func() error {
			if err := r.readCluster(ctx, createResult.JSON200.Id, &data); err != nil {
				return err
			}
			switch data.Status.ValueString() {
			case string(sdk.CLUSTER_STATUS_IN_PROGRESS):
				return fmt.Errorf("cluster is in progress")
			case string(sdk.CLUSTER_STATUS_ERROR):
				return fmt.Errorf("cluster is in error state")
			case string(sdk.CLUSTER_STATUS_DELETING):
				return fmt.Errorf("cluster is in deleting state")
			case string(sdk.CLUSTER_STATUS_READY):
				return nil
			default:
				return fmt.Errorf("cluster is in unknown state")
			}
		},
		retry.Context(ctx),
		retry.Delay(10*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(attempts),
		retry.RetryIf(func(err error) bool {
			return err != nil && err.Error() == "cluster is in progress"
		}),
	)

	if err != nil {
		resp.Diagnostics.AddError("Unable to create cluster", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readCluster(ctx, data.Id.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Unable to read cluster", err.Error())
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	listResult, err := r.client.ListNodePoolsWithResponse(ctx, data.Id.ValueString(), &sdk.ListNodePoolsParams{
		OnlyDefault: &[]bool{true}[0],
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list default node pool", err.Error())
		return
	}
	if listResult.StatusCode() != 200 {
		resp.Diagnostics.AddError("Unable to list default node pool", fmt.Sprintf("http response status code: %d", listResult.StatusCode()))
		return
	}
	if listResult.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to list default node pool", "node pools is nil")
		return
	}
	if len(*listResult.JSON200) == 0 {
		resp.Diagnostics.AddError("Unable to list default node pool", "no node pools found")
		return
	}
	defaultNodePool := (*listResult.JSON200)[0]

	params := &sdk.UpdateClusterParams{}
	body := sdk.UpdateClusterJSONRequestBody{
		NodeCount: data.NodeCount.ValueInt64(),
	}
	updateResult, err := r.client.UpdateClusterWithResponse(ctx, data.Id.ValueString(), params, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update cluster", err.Error())
		return
	}
	if updateResult.StatusCode() != 200 {
		resp.Diagnostics.AddError("Unable to update cluster", fmt.Sprintf("http response status code: %d", updateResult.StatusCode()))
		return
	}
	if updateResult.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to update cluster", "cluster is nil")
		return
	}

	// watch for resizing update if node count is different
	if defaultNodePool.NodeCount != data.NodeCount.ValueInt64() {
		// Calculate timeout based on new node count (10-20 minutes)
		attempts := calculateRetryAttempts(data.NodeCount.ValueInt64())

		err = retry.Do(
			func() error {
				showResult, err := r.client.ShowNodePoolWithResponse(ctx, defaultNodePool.ClusterID, defaultNodePool.Id, &sdk.ShowNodePoolParams{})
				if err != nil {
					return err
				}
				if showResult.StatusCode() != 200 {
					return fmt.Errorf("http response status code: %d", showResult.StatusCode())
				}
				if showResult.JSON200 == nil {
					return fmt.Errorf("node pool is nil")
				}
				switch showResult.JSON200.Status {
				case string(sdk.NODE_POOL_STATUS_RESIZING):
					return fmt.Errorf("node pool is in resizing state")
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
			retry.Delay(10*time.Second),
			retry.DelayType(retry.FixedDelay),
			retry.Attempts(attempts),
			retry.RetryIf(func(err error) bool {
				return err != nil && err.Error() == "node pool is in resizing state"
			}),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError("Unable to update cluster", err.Error())
		return
	}

	if err := r.readCluster(ctx, data.Id.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Unable to update cluster", err.Error())
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteResult, err := r.client.DeleteClusterWithResponse(ctx, data.Id.ValueString(), &sdk.DeleteClusterParams{}, sdk.DeleteClusterRequestBody{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete cluster", err.Error())
		return
	}
	if deleteResult.StatusCode() >= 400 {
		resp.Diagnostics.AddError("Unable to delete cluster", fmt.Sprintf("http response status code: %d", deleteResult.StatusCode()))
		return
	}
	if deleteResult.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to delete cluster", "cluster is nil")
		return
	}

	// Use 10 minute timeout for deletion (independent of node count)
	err = retry.Do(
		func() error {
			showResult, err := r.client.ShowClusterWithResponse(ctx, data.Id.ValueString(), &sdk.ShowClusterParams{})
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
				return fmt.Errorf("cluster is nil")
			}
			if showResult.JSON200.Deleted {
				return nil
			}
			switch showResult.JSON200.Status {
			case string(sdk.CLUSTER_STATUS_IN_PROGRESS):
				return fmt.Errorf("cluster is in progress")
			case string(sdk.CLUSTER_STATUS_ERROR):
				return fmt.Errorf("cluster is in error state")
			case string(sdk.CLUSTER_STATUS_DELETING):
				return fmt.Errorf("cluster is in deleting state")
			case string(sdk.CLUSTER_STATUS_READY):
				return nil
			default:
				return fmt.Errorf("cluster is in unknown state")
			}
		},
		retry.Context(ctx),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(10*time.Second),
		retry.Attempts(60), // 10 minutes
		retry.RetryIf(func(err error) bool {
			return err != nil && err.Error() == "cluster is in deleting state"
		}),
	)

	if err != nil {
		resp.Diagnostics.AddError("Unable to delete cluster", err.Error())
		return
	}
}

func (r *ClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// calculateRetryAttempts calculates the number of retry attempts based on node count.
// Provides 10 minutes for small clusters (≤3 nodes), 20 minutes for larger clusters.
func calculateRetryAttempts(nodeCount int64) uint {
	// Base: 10 minutes = 60 attempts × 10 seconds
	baseAttempts := uint(60)

	// Add 10 more minutes (60 attempts) for clusters with more than 3 nodes
	if nodeCount > 3 {
		return baseAttempts + 60 // 20 minutes total
	}

	return baseAttempts // 10 minutes
}

func (r *ClusterResource) readCluster(ctx context.Context, id string, data *ClusterResourceModel) error {
	params := &sdk.ShowClusterParams{}
	result, err := r.client.ShowClusterWithResponse(ctx, id, params)
	if err != nil {
		return err
	}
	if result.StatusCode() != 200 {
		return fmt.Errorf("http response status code: %d", result.StatusCode())
	}
	if result.JSON200 == nil {
		return fmt.Errorf("cluster is nil")
	}

	data.Id = types.StringValue(result.JSON200.Id)
	data.Name = types.StringValue(result.JSON200.Name)
	data.ClusterId = types.StringValue(result.JSON200.ClusterID)
	data.ProjectId = types.StringValue(result.JSON200.ProjectID)
	data.ControlPlaneName = types.StringValue(result.JSON200.ControlPlaneName)
	data.ControlPlaneNamespace = types.StringValue(result.JSON200.ControlPlaneNamespace)
	data.Keypair = types.StringValue(result.JSON200.Keypair)
	if result.JSON200.Tags != nil {
		listValues, diags := types.ListValueFrom(ctx, types.StringType, *result.JSON200.Tags)
		if diags.HasError() {
			return fmt.Errorf("failed to convert tags to list")
		}
		data.Tags = listValues
	} else {
		data.Tags = types.ListNull(types.StringType)
	}
	data.Status = types.StringValue(result.JSON200.Status)
	data.Phase = types.StringValue(result.JSON200.Phase)
	data.LastErrorId = types.StringValue(result.JSON200.LastErrorID)
	data.CreatedAt = types.Int64Value(result.JSON200.CreatedAt)
	data.UpdatedAt = types.Int64Value(result.JSON200.UpdatedAt)
	data.Deleted = types.BoolValue(result.JSON200.Deleted)
	if result.JSON200.DeletedAt != nil {
		data.DeletedAt = types.Int64Value(*result.JSON200.DeletedAt)
	} else {
		data.DeletedAt = types.Int64Null()
	}

	return nil
}
