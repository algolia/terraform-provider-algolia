data "algolia_clusters" "example" {}

output "cluster_names" {
  value = [for cluster in data.algolia_clusters.example.clusters : cluster.cluster_name]
}
