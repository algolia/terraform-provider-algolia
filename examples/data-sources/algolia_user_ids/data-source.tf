data "algolia_user_ids" "example" {}

output "user_id_cluster_assignments" {
  value = { for user in data.algolia_user_ids.example.user_ids : user.user_id => user.cluster_name }
}
