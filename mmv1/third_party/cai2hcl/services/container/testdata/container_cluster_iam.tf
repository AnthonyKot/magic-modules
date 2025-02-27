resource "google_container_cluster_iam_policy" "my-cluster_iam_policy" {
  location    = "us-central1"
  cluster     = "my-cluster"
  project     = "my-project"
  policy_data = "{\"bindings\":[{\"members\":[\"user:admin@example.com\"],\"role\":\"roles/container.admin\"},{\"members\":[\"user:dev@example.com\",\"group:developers@example.com\"],\"role\":\"roles/container.developer\"}],\"version\":1}"
}