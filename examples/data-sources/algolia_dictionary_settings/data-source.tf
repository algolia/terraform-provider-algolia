resource "algolia_dictionary_settings" "example" {
  disable_standard_entries = {
    stopwords = {
      en = true
    }
  }
}

data "algolia_dictionary_settings" "example" {
  depends_on = [algolia_dictionary_settings.example]
}

# Which standard entries are disabled, per dictionary and language.
output "disabled_standard_entries" {
  value = data.algolia_dictionary_settings.example.disable_standard_entries
}
