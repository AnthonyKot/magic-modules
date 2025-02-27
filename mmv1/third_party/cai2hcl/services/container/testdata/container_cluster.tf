resource "google_container_cluster" "my-cluster" {
  name                     = "my-cluster"
  description              = "A GKE cluster for testing"
  location                 = "us-central1"
  network                  = "default"
  subnetwork               = "default"
  initial_node_count       = 1
  enable_intranode_visibility = true
  node_locations           = ["us-central1-a", "us-central1-b", "us-central1-c"]
  project                  = "my-project"
  
  node_config {
    machine_type           = "e2-medium"
    disk_size_gb           = 100
    disk_type              = "pd-standard"
    image_type             = "COS_CONTAINERD"
    oauth_scopes           = ["https://www.googleapis.com/auth/cloud-platform"]
    service_account        = "default"
    metadata = {
      disable-legacy-endpoints = "true"
    }
  }
  
  addons_config {
    http_load_balancing {
      disabled = false
    }
    horizontal_pod_autoscaling {
      disabled = false
    }
    network_policy_config {
      disabled = true
    }
  }
  
  ip_allocation_policy {
    use_ip_aliases          = true
    cluster_ipv4_cidr_block = "10.52.0.0/14"
    services_ipv4_cidr_block = "10.56.0.0/20"
  }
  
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.0.16/28"
  }
  
  network_policy {
    enabled  = false
    provider = "PROVIDER_UNSPECIFIED"
  }
  
  database_encryption {
    state    = "DECRYPTED"
  }
  
  workload_identity_config {
    workload_pool = "my-project.svc.id.goog"
  }
  
  release_channel {
    channel = "REGULAR"
  }
  
  resource_labels = {
    environment = "test"
  }
}