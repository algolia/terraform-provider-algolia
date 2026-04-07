resource "algolia_agent_provider" "openai" {
  name          = "OpenAI Production"
  provider_name = "openai"

  openai {
    api_key  = var.openai_api_key
    base_url = "https://api.openai.com/v1"
  }
}

data "algolia_agent_provider" "openai" {
  provider_id = algolia_agent_provider.openai.id
}

output "provider_name" {
  value = data.algolia_agent_provider.openai.name
}

output "provider_base_url" {
  value = data.algolia_agent_provider.openai.openai.base_url
}

variable "openai_api_key" {
  description = "OpenAI API key used to create the Agent Studio provider."
  type        = string
  sensitive   = true
}
