terraform {
  required_providers {
    algolia = {
      source = "registry.terraform.io/algolia/algolia"
    }
  }
}

provider "algolia" {}

variable "agent_provider_id" {
  description = "Deprecated override for an existing Agent Studio provider UUID."
  type        = string
  default     = null
}

variable "agent_model" {
  description = "Model identifier supported by the selected Agent Studio provider."
  type        = string
}

variable "openai_api_key" {
  description = "OpenAI API key used to create the Agent Studio provider in Terraform."
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

# The products index already exists — import it first:
#   terraform import algolia_index.products products
# Then `terraform apply` won't re-apply settings (no diff).
resource "algolia_index" "products" {
  name                = "products"
  deletion_protection = true
}

# FAQ index — created fresh by Terraform
resource "algolia_index" "faq" {
  name                = "faq"
  deletion_protection = false

  attributes {
    searchable_attributes  = ["question", "answer", "category"]
    attributes_to_retrieve = ["question", "answer", "category", "url"]
  }

  ranking {
    ranking        = ["typo", "geo", "words", "filters", "proximity", "attribute", "exact", "custom"]
    custom_ranking = ["desc(popularity)"]
  }

  faceting {
    attributes_for_faceting = ["searchable(category)"]
    max_values_per_facet    = 50
    sort_facet_values_by    = "count"
  }

  highlighting {
    highlight_pre_tag  = "<em>"
    highlight_post_tag = "</em>"
  }

  pagination {
    hits_per_page = 10
  }

  typos {
    typo_tolerance = "true"
  }

  languages {
    query_languages = ["en"]
  }

  query_strategy {
    query_type = "prefixLast"
  }
}

# Agent that searches both indexes
resource "algolia_agent" "example" {
  name         = "Support Bot"
  description  = "Customer support agent powered by Algolia search"
  instructions = "You are a helpful support agent. Answer questions using the search tools provided."

  system_prompt = "Always be polite and concise."
  provider_id   = coalesce(var.agent_provider_id, algolia_agent_provider.openai.id)
  model         = var.agent_model

  publish             = true
  deletion_protection = false

  config = jsonencode({
    temperature = 0.7
    max_tokens  = 1024
  })

  tool_algolia_search {
    name = "search_products"

    index {
      name        = algolia_index.products.name
      description = "Product catalog — search by label, brand, category, color, or type"
      search_parameters = jsonencode({
        attributesToRetrieve = ["label", "brand", "categories", "price", "type"]
        hitsPerPage          = 5
      })
    }

    # index {
    #   name        = algolia_index.faq.name
    #   description = "Frequently asked questions — search by question, answer, or category"
    #   search_parameters = jsonencode({
    #     attributesToRetrieve = ["question", "answer", "category", "url"]
    #     hitsPerPage          = 5
    #   })
    # }
  }

  tool_client_side {
    name        = "get_order_status"
    description = "Retrieve order status from the backend"
    input_schema = jsonencode({
      type = "object"
      properties = {
        order_id = { type = "string", description = "The order ID" }
      }
      required = ["order_id"]
    })
  }
}
