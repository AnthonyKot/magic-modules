package container

import (
	"testing"

	"github.com/GoogleCloudPlatform/terraform-google-conversion/v6/cai2hcl/testing"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tpg_provider "github.com/hashicorp/terraform-provider-google-beta/google-beta/provider"
)

func TestContainerClusterConverter(t *testing.T) {
	var provider *schema.Provider = tpg_provider.Provider()
	converter := NewContainerClusterConverter(provider)

	testing.AssertTestFiles(t, converter, "testdata/container_cluster.json", "testdata/container_cluster.tf")
	testing.AssertTestFiles(t, converter, "testdata/container_cluster_iam.json", "testdata/container_cluster_iam.tf")
}