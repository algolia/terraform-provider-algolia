resource "algolia_dictionary_settings" "example" {
  disable_standard_entries = {
    stopwords = {
      en = true
    }
  }
}
