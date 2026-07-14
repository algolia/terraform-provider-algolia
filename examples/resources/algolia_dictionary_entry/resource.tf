resource "algolia_dictionary_entry" "stopword" {
  dictionary = "stopwords"
  language   = "en"
  word       = "the"
}

resource "algolia_dictionary_entry" "plural" {
  dictionary = "plurals"
  language   = "fr"
  words      = ["cheval", "chevaux"]
}

resource "algolia_dictionary_entry" "compound" {
  dictionary    = "compounds"
  language      = "de"
  word          = "kopfschmerzen"
  decomposition = ["kopf", "schmerzen"]
}
