# WARNING - lockout risk:
#
# This resource manages the COMPLETE allowlist of source IP addresses/CIDR
# ranges permitted to call the Algolia API for this application. Every
# apply REPLACES THE ENTIRE LIST with exactly the `source` set below.
#
# The list below MUST include the IP address (or CIDR range) of every host
# that runs Terraform against this application - including this one. If you
# omit it, applying this resource will lock the Terraform runner itself out
# of the Algolia API, and this provider cannot be used to undo it (you would
# need to restore access via the Algolia dashboard or support).
resource "algolia_allowed_sources" "example" {
  source = [
    {
      source      = "203.0.113.10/32"
      description = "Terraform CI runner - required, do not remove"
    },
    {
      source      = "198.51.100.0/24"
      description = "Office network"
    },
  ]
}
