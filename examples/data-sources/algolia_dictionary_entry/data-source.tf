resource "algolia_dictionary_entry" "example" {
  dictionary = "stopwords"
  language   = "en"
  word       = "the"
}

data "algolia_dictionary_entry" "example" {
  dictionary = algolia_dictionary_entry.example.dictionary
  object_id  = algolia_dictionary_entry.example.object_id
}
