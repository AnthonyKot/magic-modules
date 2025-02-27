# Container CAI to HCL Converters

This package provides converters for GKE (Google Kubernetes Engine) resources from Cloud Asset Inventory (CAI) format to HashiCorp Configuration Language (HCL) format, which is used by Terraform.

## Supported Resources

The following resource types are supported:

- `google_container_cluster`: Converts GKE cluster configurations from CAI to HCL format.

## Usage

These converters are automatically used by the CAI to HCL conversion process when processing CAI assets of type `container.googleapis.com/Cluster`.

The converter handles both the resource configuration and IAM policies associated with GKE clusters.

## Resource Mapping

The converter handles mapping of the following key components:

- Basic cluster configuration (name, description, location)
- Node configuration (machine type, disk size, OAuth scopes, etc.)
- Networking configuration (VPC, subnetwork, IP allocation policy)
- Private cluster configuration
- Master authorized networks
- Node pool configuration
- Maintenance windows
- Workload identity configuration
- Database encryption
- Network policy settings
- Add-ons configuration (HTTP load balancing, horizontal pod autoscaling, etc.)
- Resource labels

## IAM Policy Conversion

For IAM policies attached to GKE clusters, the converter creates a corresponding `google_container_cluster_iam_policy` resource in the Terraform configuration.