terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {}

variable "openai_api_key" {
  description = "OpenAI API key used to create the Agent Studio provider."
  type        = string
  sensitive   = true
}

resource "algolia_agent_provider" "openai" {
  name          = "terraform-example-openai"
  provider_name = "openai"

  openai {
    api_key = var.openai_api_key
  }
}
