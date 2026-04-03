# algolia_agent Example

This example creates an Algolia Agent Studio agent backed by two indexes (`products` and `faq`), with an Algolia search tool and a client-side tool for order status lookups.

`publish = true` requires a real Agent Studio provider and model. The example now takes both as Terraform variables instead of relying on a hard-coded provider ID.

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/downloads) installed
- An Algolia account with a valid App ID and Admin API Key

## Steps

### 1. Build the provider

From the repo root:

```bash
make build
```

### 2. Configure the local dev override

Add the following to `~/.terraformrc` so Terraform resolves the provider from your local build instead of the registry:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/algolia/algolia" = "/path/to/terraform-provider-algolia"
  }
  direct {}
}
```

### 3. Export your credentials

```bash
export ALGOLIA_APP_ID=<your-app-id>
export ALGOLIA_API_KEY=<your-admin-api-key>
export OPENAI_API_KEY=<your-openai-api-key>
```

### 4. Create or locate an Agent Studio provider

You can create a provider in the Agent Studio UI, or directly through the API. Example for OpenAI:

```bash
curl -X POST "https://${ALGOLIA_APP_ID}.algolia.net/agent-studio/1/providers" \
  -H "X-Algolia-API-Key: ${ALGOLIA_API_KEY}" \
  -H "X-Algolia-Application-Id: ${ALGOLIA_APP_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "terraform-example-openai",
    "providerName": "openai",
    "input": {
      "apiKey": "'"${OPENAI_API_KEY}"'"
    }
  }'
```

Then list the models that provider supports and pick one:

```bash
curl "https://${ALGOLIA_APP_ID}.algolia.net/agent-studio/1/providers/<provider-id>/models" \
  -H "X-Algolia-API-Key: ${ALGOLIA_API_KEY}" \
  -H "X-Algolia-Application-Id: ${ALGOLIA_APP_ID}"
```

For OpenAI providers, `gpt-4.1-mini` is a good default when available.

### 5. Import the existing `products` index (if it already exists in Algolia)

```bash
terraform import algolia_index.products products
```

Skip this step if the `products` index doesn't exist yet — Terraform will create it.

### 6. Run the example

```bash
# Preview what Terraform will do
terraform plan \
  -var="agent_provider_id=<provider-id>" \
  -var="agent_model=<model>"

# Apply
terraform apply \
  -var="agent_provider_id=<provider-id>" \
  -var="agent_model=<model>"
```

## Cleanup

```bash
terraform destroy
```

> **Note:** `deletion_protection = true` is set on the `products` index, so it will not be destroyed. Set it to `false` first if you want Terraform to delete it.

> **Note:** Agent Studio currently does not support unpublishing an agent. If you change `publish` from `true` to `false`, Terraform will return an error instead of silently claiming the agent is back in draft state.
