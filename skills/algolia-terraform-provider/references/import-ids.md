# Import IDs

**Import IDs differ per resource, and no single rule covers all of them.** Every resource
has an `## Import` section with a worked example in its page under
[docs/resources](https://github.com/algolia/terraform-provider-algolia/tree/main/docs/resources); read that rather than inferring the form
from a neighbouring resource.

The shapes in use:

```bash
terraform import algolia_index.products products                  # the object's own name
terraform import algolia_rule.promo products/my-rule              # <index>/<object_id>
terraform import algolia_recommend_rule.hide products/related-products/hide  # <index>/<model>/<object_id>
terraform import algolia_agent.support 01234567-89ab-cdef-0123-456789abcdef  # a UUID
terraform import algolia_ab_test.pricing 42                       # a number
terraform import algolia_allowed_sources.main YourApplicationID   # the application id
```

Notes on the less obvious ones:

- The **UUID** form covers `algolia_agent`, `algolia_agent_provider` and all five
  `algolia_ingestion_*` resources. Those ids are only discoverable from the Algolia
  dashboard or API, not from the object's name.
- `algolia_api_key`'s import id is **the key value itself**, which is also why the resource
  treats its `id` as sensitive.
- `algolia_dictionary_entry` takes `<dictionary>/<entry>`, as in `stopwords/my-stopword`,
  where the first segment is a dictionary rather than an index.
- `algolia_composition_rule` takes `<composition>/<rule>`, likewise not an index.
- `algolia_allowed_sources` and `algolia_dictionary_settings` are per-application
  singletons, so the id is the application id.

After importing, run `terraform plan`. An empty plan means the configuration matches what
Algolia holds; anything else means the two disagree and needs reconciling before the next
apply.
