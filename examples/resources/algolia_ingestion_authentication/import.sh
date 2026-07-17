# Import an Ingestion authentication by its UUID. Secret values in `input`
# are redacted by the API and cannot be recovered on import; set them in
# configuration and apply to reconcile.
terraform import algolia_ingestion_authentication.algolia_destination 6c02aeb1-775e-418e-870b-1faccd4b2c0f
