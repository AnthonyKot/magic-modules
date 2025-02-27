package container

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/terraform-google-conversion/v6/caiasset"
	"github.com/GoogleCloudPlatform/terraform-google-conversion/v6/cai2hcl/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/api/container/v1"
)

// ContainerClusterAssetType is the CAI asset type name for container cluster.
const ContainerClusterAssetType string = "container.googleapis.com/Cluster"

// ContainerClusterSchemaName is the TF resource schema name for container cluster.
const ContainerClusterSchemaName string = "google_container_cluster"

// ContainerClusterConverter for container cluster resource.
type ContainerClusterConverter struct {
	name   string
	schema map[string]*schema.Schema
}

// NewContainerClusterConverter returns an HCL converter for container cluster.
func NewContainerClusterConverter(provider *schema.Provider) common.Converter {
	schema := provider.ResourcesMap[ContainerClusterSchemaName].Schema

	return &ContainerClusterConverter{
		name:   ContainerClusterSchemaName,
		schema: schema,
	}
}

// Convert converts asset to HCL resource blocks.
func (c *ContainerClusterConverter) Convert(assets []*caiasset.Asset) ([]*common.HCLResourceBlock, error) {
	var blocks []*common.HCLResourceBlock
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		if asset.IAMPolicy != nil {
			iamBlock, err := c.convertIAM(asset)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, iamBlock)
		}
		if asset.Resource != nil && asset.Resource.Data != nil {
			block, err := c.convertResourceData(asset)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func (c *ContainerClusterConverter) convertIAM(asset *caiasset.Asset) (*common.HCLResourceBlock, error) {
	if asset == nil || asset.IAMPolicy == nil {
		return nil, fmt.Errorf("asset IAM policy is nil")
	}
	location := common.ParseFieldValue(asset.Name, "locations")
	clusterName := common.ParseFieldValue(asset.Name, "clusters")
	project := common.ParseFieldValue(asset.Name, "projects")

	policyData, err := json.Marshal(asset.IAMPolicy)
	if err != nil {
		return nil, err
	}

	return &common.HCLResourceBlock{
		Labels: []string{
			c.name + "_iam_policy",
			clusterName + "_iam_policy",
		},
		Value: cty.ObjectVal(map[string]cty.Value{
			"location":  cty.StringVal(location),
			"cluster":   cty.StringVal(clusterName),
			"project":   cty.StringVal(project),
			"policy_data": cty.StringVal(string(policyData)),
		}),
	}, nil
}

func (c *ContainerClusterConverter) convertResourceData(asset *caiasset.Asset) (*common.HCLResourceBlock, error) {
	if asset == nil || asset.Resource == nil || asset.Resource.Data == nil {
		return nil, fmt.Errorf("asset resource data is nil")
	}

	var cluster *container.Cluster
	if err := common.DecodeJSON(asset.Resource.Data, &cluster); err != nil {
		return nil, err
	}

	hclData := make(map[string]interface{})
	hclData["name"] = cluster.Name
	hclData["description"] = cluster.Description
	
	// Handle location - it could be a zone or a region
	location := common.ParseFieldValue(asset.Name, "locations")
	if location != "" {
		hclData["location"] = location
	} else if cluster.Location != "" {
		hclData["location"] = cluster.Location
	} else {
		// Handle legacy zone field
		hclData["zone"] = common.ParseFieldValue(cluster.Zone, "zones")
	}
	
	if cluster.Network != "" {
		// Extract network name from the full path
		hclData["network"] = lastSegment(cluster.Network)
	}
	
	if cluster.Subnetwork != "" {
		// Extract subnetwork name from the full path
		hclData["subnetwork"] = lastSegment(cluster.Subnetwork)
	}
	
	// Handle initial node count
	if cluster.InitialNodeCount > 0 {
		hclData["initial_node_count"] = cluster.InitialNodeCount
	}
	
	// Extract node locations (zones in the same region as the cluster)
	if len(cluster.Locations) > 0 {
		hclData["node_locations"] = filterOutPrimaryZone(cluster.Locations, cluster.Zone)
	}
	
	// Handle networking configurations
	if cluster.NetworkConfig != nil {
		if cluster.NetworkConfig.EnableIntraNodeVisibility {
			hclData["enable_intranode_visibility"] = cluster.NetworkConfig.EnableIntraNodeVisibility
		}
		
		if cluster.NetworkConfig.DefaultSnatStatus != nil {
			hclData["default_snat_status"] = []map[string]interface{}{
				{
					"disabled": cluster.NetworkConfig.DefaultSnatStatus.Disabled,
				},
			}
		}
	}
	
	// Handle cluster ipv4 cidr
	if cluster.ClusterIpv4Cidr != "" {
		hclData["cluster_ipv4_cidr"] = cluster.ClusterIpv4Cidr
	}
	
	// Handle IP allocation policy
	if cluster.IpAllocationPolicy != nil {
		ipAllocationPolicy := make(map[string]interface{})
		
		if cluster.IpAllocationPolicy.UseIpAliases {
			ipAllocationPolicy["use_ip_aliases"] = cluster.IpAllocationPolicy.UseIpAliases
		}
		
		if cluster.IpAllocationPolicy.ClusterIpv4CidrBlock != "" {
			ipAllocationPolicy["cluster_ipv4_cidr_block"] = cluster.IpAllocationPolicy.ClusterIpv4CidrBlock
		}
		
		if cluster.IpAllocationPolicy.ServicesIpv4CidrBlock != "" {
			ipAllocationPolicy["services_ipv4_cidr_block"] = cluster.IpAllocationPolicy.ServicesIpv4CidrBlock
		}
		
		if cluster.IpAllocationPolicy.ClusterSecondaryRangeName != "" {
			ipAllocationPolicy["cluster_secondary_range_name"] = cluster.IpAllocationPolicy.ClusterSecondaryRangeName
		}
		
		if cluster.IpAllocationPolicy.ServicesSecondaryRangeName != "" {
			ipAllocationPolicy["services_secondary_range_name"] = cluster.IpAllocationPolicy.ServicesSecondaryRangeName
		}
		
		if len(ipAllocationPolicy) > 0 {
			hclData["ip_allocation_policy"] = []map[string]interface{}{ipAllocationPolicy}
		}
	}
	
	// Handle master authorized networks config
	if cluster.MasterAuthorizedNetworksConfig != nil {
		masterAuthorizedNetworksConfig := make(map[string]interface{})
		
		// In Terraform, the existence of the block implies enabled = true
		// so we explicitly set it to avoid diffs
		masterAuthorizedNetworksConfig["enabled"] = cluster.MasterAuthorizedNetworksConfig.Enabled
		
		if len(cluster.MasterAuthorizedNetworksConfig.CidrBlocks) > 0 {
			var cidrBlocks []map[string]interface{}
			for _, cidrBlock := range cluster.MasterAuthorizedNetworksConfig.CidrBlocks {
				cidrBlocks = append(cidrBlocks, map[string]interface{}{
					"cidr_block":   cidrBlock.CidrBlock,
					"display_name": cidrBlock.DisplayName,
				})
			}
			masterAuthorizedNetworksConfig["cidr_blocks"] = cidrBlocks
		}
		
		if len(masterAuthorizedNetworksConfig) > 0 {
			hclData["master_authorized_networks_config"] = []map[string]interface{}{masterAuthorizedNetworksConfig}
		}
	}
	
	// Handle node config
	if cluster.NodeConfig != nil {
		nodeConfig := make(map[string]interface{})
		
		if cluster.NodeConfig.MachineType != "" {
			nodeConfig["machine_type"] = cluster.NodeConfig.MachineType
		}
		
		if cluster.NodeConfig.DiskSizeGb > 0 {
			nodeConfig["disk_size_gb"] = cluster.NodeConfig.DiskSizeGb
		}
		
		if cluster.NodeConfig.DiskType != "" {
			nodeConfig["disk_type"] = cluster.NodeConfig.DiskType
		}
		
		if len(cluster.NodeConfig.OauthScopes) > 0 {
			nodeConfig["oauth_scopes"] = cluster.NodeConfig.OauthScopes
		}
		
		if cluster.NodeConfig.ServiceAccount != "" {
			nodeConfig["service_account"] = cluster.NodeConfig.ServiceAccount
		}
		
		if cluster.NodeConfig.ImageType != "" {
			nodeConfig["image_type"] = cluster.NodeConfig.ImageType
		}
		
		if cluster.NodeConfig.Preemptible {
			nodeConfig["preemptible"] = cluster.NodeConfig.Preemptible
		}
		
		if len(cluster.NodeConfig.Labels) > 0 {
			nodeConfig["labels"] = cluster.NodeConfig.Labels
		}
		
		if len(cluster.NodeConfig.Tags) > 0 {
			nodeConfig["tags"] = cluster.NodeConfig.Tags
		}
		
		if len(cluster.NodeConfig.Metadata) > 0 {
			nodeConfig["metadata"] = cluster.NodeConfig.Metadata
		}
		
		if cluster.NodeConfig.MinCpuPlatform != "" {
			nodeConfig["min_cpu_platform"] = cluster.NodeConfig.MinCpuPlatform
		}
		
		// Handle guest accelerators
		if len(cluster.NodeConfig.Accelerators) > 0 {
			accelerators := make([]map[string]interface{}, 0, len(cluster.NodeConfig.Accelerators))
			for _, accelerator := range cluster.NodeConfig.Accelerators {
				accelerators = append(accelerators, map[string]interface{}{
					"count": accelerator.AcceleratorCount,
					"type":  accelerator.AcceleratorType,
				})
			}
			nodeConfig["guest_accelerator"] = accelerators
		}
		
		// Handle workload metadata config
		if cluster.NodeConfig.WorkloadMetadataConfig != nil {
			workloadMetadataConfig := make(map[string]interface{})
			workloadMetadataConfig["mode"] = cluster.NodeConfig.WorkloadMetadataConfig.Mode
			nodeConfig["workload_metadata_config"] = []map[string]interface{}{workloadMetadataConfig}
		}
		
		// Add shielded instance config if present
		if cluster.NodeConfig.ShieldedInstanceConfig != nil {
			shieldedInstanceConfig := make(map[string]interface{})
			shieldedInstanceConfig["enable_secure_boot"] = cluster.NodeConfig.ShieldedInstanceConfig.EnableSecureBoot
			shieldedInstanceConfig["enable_integrity_monitoring"] = cluster.NodeConfig.ShieldedInstanceConfig.EnableIntegrityMonitoring
			nodeConfig["shielded_instance_config"] = []map[string]interface{}{shieldedInstanceConfig}
		}
		
		// Add the node config if we have at least one item
		if len(nodeConfig) > 0 {
			hclData["node_config"] = []map[string]interface{}{nodeConfig}
		}
	}
	
	// Handle addons config
	if cluster.AddonsConfig != nil {
		addonsConfig := make(map[string]interface{})
		
		if cluster.AddonsConfig.HttpLoadBalancing != nil {
			addonsConfig["http_load_balancing"] = []map[string]interface{}{
				{
					"disabled": cluster.AddonsConfig.HttpLoadBalancing.Disabled,
				},
			}
		}
		
		if cluster.AddonsConfig.HorizontalPodAutoscaling != nil {
			addonsConfig["horizontal_pod_autoscaling"] = []map[string]interface{}{
				{
					"disabled": cluster.AddonsConfig.HorizontalPodAutoscaling.Disabled,
				},
			}
		}
		
		if cluster.AddonsConfig.NetworkPolicyConfig != nil {
			addonsConfig["network_policy_config"] = []map[string]interface{}{
				{
					"disabled": cluster.AddonsConfig.NetworkPolicyConfig.Disabled,
				},
			}
		}
		
		if cluster.AddonsConfig.DnsCacheConfig != nil {
			addonsConfig["dns_cache_config"] = []map[string]interface{}{
				{
					"enabled": cluster.AddonsConfig.DnsCacheConfig.Enabled,
				},
			}
		}
		
		if cluster.AddonsConfig.ConfigConnectorConfig != nil {
			addonsConfig["config_connector_config"] = []map[string]interface{}{
				{
					"enabled": cluster.AddonsConfig.ConfigConnectorConfig.Enabled,
				},
			}
		}
		
		if cluster.AddonsConfig.GcePersistentDiskCsiDriverConfig != nil {
			addonsConfig["gce_persistent_disk_csi_driver_config"] = []map[string]interface{}{
				{
					"enabled": cluster.AddonsConfig.GcePersistentDiskCsiDriverConfig.Enabled,
				},
			}
		}
		
		if len(addonsConfig) > 0 {
			hclData["addons_config"] = []map[string]interface{}{addonsConfig}
		}
	}
	
	// Handle private cluster config
	if cluster.PrivateClusterConfig != nil {
		privateClusterConfig := make(map[string]interface{})
		privateClusterConfig["enable_private_nodes"] = cluster.PrivateClusterConfig.EnablePrivateNodes
		privateClusterConfig["enable_private_endpoint"] = cluster.PrivateClusterConfig.EnablePrivateEndpoint
		privateClusterConfig["master_ipv4_cidr_block"] = cluster.PrivateClusterConfig.MasterIpv4CidrBlock
		
		hclData["private_cluster_config"] = []map[string]interface{}{privateClusterConfig}
	}
	
	// Handle network policy
	if cluster.NetworkPolicy != nil {
		networkPolicy := make(map[string]interface{})
		networkPolicy["enabled"] = cluster.NetworkPolicy.Enabled
		networkPolicy["provider"] = cluster.NetworkPolicy.Provider
		
		hclData["network_policy"] = []map[string]interface{}{networkPolicy}
	}
	
	// Handle maintenance policy
	if cluster.MaintenancePolicy != nil && cluster.MaintenancePolicy.Window != nil {
		maintenancePolicy := make(map[string]interface{})
		
		if cluster.MaintenancePolicy.Window.DailyMaintenanceWindow != nil {
			dailyMaintenanceWindow := make(map[string]interface{})
			dailyMaintenanceWindow["start_time"] = cluster.MaintenancePolicy.Window.DailyMaintenanceWindow.StartTime
			
			maintenancePolicy["daily_maintenance_window"] = []map[string]interface{}{dailyMaintenanceWindow}
		}
		
		if cluster.MaintenancePolicy.Window.RecurringWindow != nil {
			recurringWindow := make(map[string]interface{})
			recurringWindow["start_time"] = cluster.MaintenancePolicy.Window.RecurringWindow.Window.StartTime
			recurringWindow["end_time"] = cluster.MaintenancePolicy.Window.RecurringWindow.Window.EndTime
			
			// Convert recurrence into the format Terraform expects
			recurrence := cluster.MaintenancePolicy.Window.RecurringWindow.Recurrence
			recurringWindow["recurrence"] = recurrence
			
			maintenancePolicy["recurring_window"] = []map[string]interface{}{recurringWindow}
		}
		
		hclData["maintenance_policy"] = []map[string]interface{}{maintenancePolicy}
	}
	
	// Handle workload identity config
	if cluster.WorkloadIdentityConfig != nil && cluster.WorkloadIdentityConfig.WorkloadPool != "" {
		workloadIdentityConfig := make(map[string]interface{})
		workloadIdentityConfig["workload_pool"] = cluster.WorkloadIdentityConfig.WorkloadPool
		
		hclData["workload_identity_config"] = []map[string]interface{}{workloadIdentityConfig}
	}
	
	// Handle database encryption
	if cluster.DatabaseEncryption != nil {
		databaseEncryption := make(map[string]interface{})
		databaseEncryption["state"] = cluster.DatabaseEncryption.State
		databaseEncryption["key_name"] = cluster.DatabaseEncryption.KeyName
		
		hclData["database_encryption"] = []map[string]interface{}{databaseEncryption}
	}
	
	// Convert legacy_abac to Terraform format
	if cluster.LegacyAbac != nil {
		hclData["enable_legacy_abac"] = cluster.LegacyAbac.Enabled
	}
	
	// Handle binary authorization
	if cluster.BinaryAuthorization != nil {
		binaryAuthorization := make(map[string]interface{})
		binaryAuthorization["enabled"] = cluster.BinaryAuthorization.Enabled
		
		hclData["binary_authorization"] = []map[string]interface{}{binaryAuthorization}
	}
	
	// Handle release channel
	if cluster.ReleaseChannel != nil {
		releaseChannel := make(map[string]interface{})
		releaseChannel["channel"] = cluster.ReleaseChannel.Channel
		
		hclData["release_channel"] = []map[string]interface{}{releaseChannel}
	}
	
	// Handle project
	if project := common.ParseFieldValue(asset.Name, "projects"); project != "" {
		hclData["project"] = project
	}
	
	// Convert resource labels
	if len(cluster.ResourceLabels) > 0 {
		hclData["resource_labels"] = cluster.ResourceLabels
	}
	
	// Convert to CTY value with the appropriate schema
	ctyVal, err := common.MapToCtyValWithSchema(hclData, c.schema)
	if err != nil {
		return nil, err
	}
	
	return &common.HCLResourceBlock{
		Labels: []string{c.name, cluster.Name},
		Value:  ctyVal,
	}, nil
}

// Helper to extract the last segment from a GCP resource path
func lastSegment(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// Helper to filter out the primary zone from node locations
func filterOutPrimaryZone(locations []string, primaryZone string) []string {
	var filteredLocations []string
	for _, location := range locations {
		if location != primaryZone {
			filteredLocations = append(filteredLocations, location)
		}
	}
	return filteredLocations
}