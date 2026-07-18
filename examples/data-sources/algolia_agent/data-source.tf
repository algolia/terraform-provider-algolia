# Look up an existing Agent Studio agent by its UUID.
data "algolia_agent" "example" {
  id = "01234567-89ab-cdef-0123-456789abcdef"
}

output "agent_name" {
  value = data.algolia_agent.example.name
}
