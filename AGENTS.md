# AGENTS.md

This file provides guidance to AI coding agents working with code in this repository.

## Build & Test Commands

```bash
make build                    # compile the provider
make test                     # no credentials needed; acceptance tests skip themselves without TF_ACC
make testacc                  # acceptance tests (needs ALGOLIA_APP_ID & ALGOLIA_API_KEY; some suites also need ALGOLIA_ANALYTICS_REGION)
make e2e                      # end-to-end tests against a real application (build tag `e2e`, see below)
make lint                     # run golangci-lint
make generate                 # regenerate docs via tfplugindocs

# run a single test
go test ./internal/services/index/ -run TestExpandTypoTolerance -v

# run all unit tests (skip acceptance)
go test ./... -v -timeout 120m
```

There are three test tiers, and knowing which one covers a change decides how much
you can conclude from a green run:

- **Unit tests** run everywhere with no credentials. Where an HTTP fake is needed they
  build one with `httptest` plus `search.NewClientWithConfig`; see
  `internal/services/index/delete_unlink_test.go` for the pattern.
- **Acceptance tests** (`TestAcc*`, `TF_ACC=1`) drive real Terraform plans against a
  real application through the plugin-testing framework.
- **End-to-end tests** (`internal/e2e/`, build tag `e2e`, `make e2e`) drive whole
  configurations through create, drift, reconcile, update and destroy, asserting
  Algolia's own view at each stage rather than only Terraform state.

**In CI, the acceptance and e2e jobs run on a push to `main`, and on a pull request only
when a maintainer applies the `acceptance` or `e2e` label.** They execute repository code
with the admin `ALGOLIA_API_KEY`, so they must never run unprompted on a pull request,
which would hand that key to anyone who can open one. The label changes who can start the
run, not whose code runs, so two things in the guard are load-bearing: the run is
restricted to pull requests from this repository, because GitHub withholds secrets from
fork pull requests and such a run would otherwise go green having skipped everything; and
the trigger stays `pull_request`, never `pull_request_target`, which *would* expose the
key to a fork.

Removing the label stops it again, since `unlabeled` is in the trigger's `types`. Note
that `synchronize` is too, so every new commit on a labelled pull request re-runs the live
suites: remove the label once you have the answer.

Without a label a pull request goes green on build, lint and unit tests alone. If a change
touches behaviour the unit tests cannot reach, either label the PR or run the relevant
suite locally, and say which in the PR.

### Debugging with delve

Install it if you need it (`go install github.com/go-delve/delve/cmd/dlv@latest`); it is
deliberately not pinned in `.tool-versions`, since CI has no use for it.

`main.go` accepts `-debug`, the flag providers use to be driven under a debugger, for
when you are running `terraform` by hand. For an acceptance test there is a shorter path:
the framework serves the provider from the *test* process and points the `terraform` CLI
at it by reattach, so `TF_ACC=1 dlv test ./internal/services/index/ --
-test.run TestAccIndexResource_basic` breaks straight into `Create`, `Read` or
`ModifyPlan` with no reattach details to pass by hand.

Driving it from a script rather than a terminal needs
`--allow-non-terminal-interactive=true`, or delve refuses with "Stdin is not a terminal".

Worth it for one kind of question: what the framework actually handed the provider, where
Config, Plan and State differ in ways `tflog` makes tedious to compare. Printing a value
shows the distinction directly, as in
`basetypes.BoolValue {state: ValueStateKnown (2), value: false}`. When the question is
what Algolia did instead, a `curl` probe against the API answers it faster.

### Why an acceptance suite skipped

Acceptance tests need `TF_ACC=1` plus `ALGOLIA_APP_ID` and `ALGOLIA_API_KEY`. Beyond
that, several suites skip **silently** without something extra, so a green run does not
by itself mean the code was exercised. Check this list before concluding anything from
one:

- **Region-routed services** (Query Suggestions, Personalization, A/B Testing, Ingestion)
  need `ALGOLIA_ANALYTICS_REGION` set to `us` or `eu`.
- **Personalization** also needs `ALGOLIA_RUN_PERSONALIZATION_ACC=1`, because the API
  enforces a daily strategy-save quota.
- **A/B Testing** also needs `ALGOLIA_RUN_ABTESTING_ACC=1`, because creating a test has
  cost and quota implications for the target application.
- **MCM** also needs `ALGOLIA_RUN_MCM_ACC=1`, because `algolia_clusters` and
  `algolia_user_ids` error on applications that are not on a multi-cluster plan, which is
  most of them.
- **Allowed Sources** also needs `ALGOLIA_RUN_ALLOWEDSOURCES_ACC=1`, because the endpoint
  requires the Vault feature (HTTP 402 without it).
- **Compositions** also needs `ALGOLIA_RUN_COMPOSITION_ACC=1`, because the API is not
  enabled on most applications (HTTP 404 without it).
- **Agent** needs `OPENAI_API_KEY` for its published-agent tests, and **Agent Provider**
  needs `OPENAI_API_KEY` and `GOOGLE_GENAI_API_KEY` for the OpenAI and Google suites.
  The CI acceptance job supplies neither, so those suites skip on every main-branch run.
  A key is also not sufficient on its own: these tests create a real provider, which
  Algolia validates by calling the vendor, so an account with an exhausted credit
  balance makes them fail rather than skip (`429 credit_balance_exhausted`).

Compositions, Allowed Sources and MCM cannot be made to pass by configuration alone;
they need an application provisioned for those features.

**Recommend** needs nothing beyond `TF_ACC` and credentials. It was once gated behind
`ALGOLIA_RUN_RECOMMEND_ACC=1` for an upstream bug, `algoliasearch-client-go` v4 failing to
decode the API's numeric `_metadata.lastUpdate` into its `*string` field. That is worked
around by `getRecommendRule` in `internal/services/recommend/get_rule.go`, which strips
`_metadata` before decoding, and the gate is gone.

### Lint and toolchain pinning

Lint configuration lives in `.golangci.yml`. `gofmt`/`goimports` are registered there as
**formatters**, which is what makes `golangci-lint run` fail on unformatted files (in
golangci-lint v2 formatters are inert unless listed). The linter version is pinned in both
`.tool-versions` and `.github/workflows/test.yml`; keep the two in sync so a lint failure
reproduces locally. The Go toolchain version is likewise pinned in both `.tool-versions` and
the `go` directive in `go.mod` - a mismatch is a hard failure under `GOTOOLCHAIN=local`.

## Architecture

This is a Terraform provider built on the **Terraform Plugin Framework** (not the older SDK v2). It uses the **Algolia Go client v4** (`algoliasearch-client-go/v4`).

**Registry address:** `registry.terraform.io/algolia/algolia`

### Package Layout

- `main.go` - entry point, serves the provider via `providerserver.Serve`
- `internal/provider/` - provider definition (schema, configure, resource and data source registration)
- `internal/services/` - one package per API surface, sixteen of them today. Fifteen
  expose Terraform resources or data sources; `crawler` holds only an HTTP client,
  because the crawler was descoped on 2026-07-18 and no crawler resource is planned. The
  provider's `crawler_user_id` and `crawler_api_key` attributes are deprecated and
  configure nothing, which `internal/provider/provider.go` says at the point of
  declaration. Do not build on them.

Five shared packages carry conventions rather than API surface, and a new resource
should reach for them instead of re-implementing:

- `internal/types/` - the `ProviderData` struct every resource reads its client from,
  extracted to break an import cycle between provider and services
- `internal/algoliaerr/` - status inspection (`IsNotFound`, `IsRetryable`), the
  house diagnostic wording via `Object(kind, id).Message(op, err)`, and `Explain`,
  which appends the field-level detail Algolia hides inside an error's extra properties
- `internal/algoliawait/` - the bounded, cancellable poll loop for Algolia's queued
  writes. Its own comment records that this loop was hand-copied into six packages
  before extraction, one of them with an uncancellable `time.Sleep`; do not make it seven
- `internal/deletionprotection/` - the `deletion_protection` attribute and its fail-safe
- `internal/analyticsregion/` - routing for region-routed APIs (Query Suggestions,
  Personalization, A/B Testing, Ingestion)

### File naming inside a service package

Files split along concerns rather than a fixed list of names: the model structs, the
schema, the two mapping directions (**expand** is Terraform model to Algolia request,
**flatten** is Algolia response to Terraform model), and the resource and data source
implementations.

Which of those exist, and what they are called, varies by package, so **read the package
you are working in rather than assuming a rule**. Reasons it varies: a read-only package
has no expand and no resource at all; some packages map inline or through a
`resource_state.go` hydrate function instead of a separate expand or flatten file; a
package covering several resource concepts prefixes each set with the concept, while a
package covering one usually does not; and `crawler` has none of these files, being a
client with no Terraform surface. `index` combines several of those, since it holds the
index, the virtual index and the indices data source.

This is one place not to generalise from a single example. Three earlier attempts to write
a single naming rule here were each wrong in a different way, because the variation is
real. Copy the layout of the package you are extending, or of the nearest existing package
with a comparable shape.

Three files in `index` hold rules rather than plumbing, and are worth reading before
touching replicas or deletes: `replica_link.go` (ownership of the primary's `replicas`
list, and the delete path), `settings_write.go` (why a settings write is re-sent), and
`delete_confirm.go` (why a published delete task is not proof of deletion).

### Key Design Patterns

**Union types:** Algolia's API has union types (`TypoTolerance`, `Distinct`, `IgnorePlurals`, `RemoveStopWords`, `OptionalWords`, `ReRankingApplyFilter`). These are mapped to simpler Terraform types:

- `TypoTolerance` (bool|enum) → `types.String` with values `"true"/"false"/"min"/"strict"`
- `Distinct` (bool|int32) → `types.Int64` where 0=false, 1=true, 2+=group sizes
- `IgnorePlurals`/`RemoveStopWords` (bool|[]language) → split into two fields: a `types.Bool` and a `types.List` for languages

**JSON-encoded fields:** Complex nested types (`decompounded_attributes`, `custom_normalization`, `user_data`, `semantic_search`, `re_ranking_apply_filter`) are stored as JSON-encoded `types.String` in Terraform state.

**Deletion protection:** nine resources carry `deletion_protection`, defaulting to true.
See the dedicated section below for which resources get it and why the others do not.

**Settings-as-index:** Algolia auto-creates an index on the first `SetSettings` call;
there is no separate "create index" API. Two consequences worth holding on to. Writing
settings to an index that has just been deleted **recreates it**, which is how an
overeager unlink resurrected a primary that the same destroy had removed. And the wait
after a write is `waitForSettingsWrite`, not a plain task wait: Algolia restarts an
index's task queue when another write turns that index into a replica, which voids the
task ID even though the write landed, so a wait that stops progressing re-sends the
write rather than burning its whole budget. Both behaviours look like complexity worth
tidying away. They are not.

**Deleting an index:** goes through `deleteIndexWithUnlink` plus `confirmIndexDeleted`.
Algolia refuses `deleteIndex` while a primary still lists the index as a replica, and it
stops refusing the moment that primary is gone, so the delete is retried *before*
anything is written to the primary. A published delete task is also not proof the index
went away, which is what `confirmIndexDeleted` re-checks.

### Replica ownership

This is the subtlest invariant in the codebase and the one most easily broken by a
change that looks local. A primary index's `replicas` setting holds **both** kinds of
replica, told apart only by a `virtual(...)` marker, and **two resources write that one
field**: `algolia_index` through `advanced.replicas`, and every `algolia_virtual_index`
through its own entry. Before the split, whichever applied last unlinked the other's
replicas.

Ownership is divided by the kind of entry, and all three halves have to hold together:

- **Plan time:** `advanced.replicas` rejects a `virtual(...)` entry. A virtual replica is
  declared by its own resource, never by naming it here.
- **Write:** `mergeStandardReplicas` re-reads the live list and keeps the virtual entries
  Algolia reports, because the API takes this field as the complete set. Writing only the
  configured names would unlink every virtual replica.
- **Read:** `standardReplicasOf` filters the virtual entries out, so state holds only what
  the attribute can declare. Surfacing them would put values in state that no
  configuration can produce, and every refresh would plan a change no apply could settle.

There is a fourth rule about the field being written at all. `advanced.replicas` is
Optional+Computed, so when a configuration says nothing about it the plan value falls
back to the last refreshed state rather than staying null. Expanding that plan yields a
list for an index whose configuration never mentioned replicas, and writing it makes
Terraform an authoritative writer of a field nobody declared, silently unlinking any
virtual replica added since that refresh. So the list is written **only when the
configuration actually declares it** (`replicasDeclared`, which inspects config rather
than plan), and when it does not, the planned value is kept in the applied state
(`preserveUndeclaredReplicas`) because the read-back would otherwise return more entries
than the plan promised and Terraform rejects that as an inconsistent apply result.

Two smaller rules complete it. Concurrent writes to one primary are serialised by
`lockPrimaryReplicas`, since Terraform applies resources in parallel and a
read-modify-write of a shared field otherwise loses entries. And a replica's own settings
report a `primary` whichever kind it is, so **classifying a replica requires reading the
primary's list** (`classifyReplicaLinkage`); the difference matters because Algolia copies
records into a standard replica and a virtual one holds none.

### Adding a New Resource

1. Create a new package under `internal/services/`
2. Implement model, schema, expand/flatten, resource, and data source files
3. Register in `internal/provider/provider.go` (Resources/DataSources methods)
4. Use `*providertypes.ProviderData` from `internal/types/` to access the Algolia client
5. If the API is region-routed, use `internal/analyticsregion` plus `ProviderData.AnalyticsRegion` instead of embedding per-resource region config
6. Raise diagnostics through `internal/algoliaerr` and wait for queued writes through
   `internal/algoliawait`, rather than formatting errors or writing a poll loop by hand
7. If destroying the resource loses something a re-apply cannot restore, add
   `deletion_protection` from `internal/deletionprotection` - see below
8. Add an example under `examples/resources/algolia_<name>/resource.tf`; it is published
   to the registry, so it has to be a configuration that actually applies
9. Run `make generate` to render the docs, and `go test ./internal/provider/ -update` to
   accept the new schema into the snapshot

### Deletion protection

`internal/deletionprotection` provides the `deletion_protection` attribute, and the
rule that an **absent value means protected**. Algolia does not store the flag, so
state written before the attribute existed carries no value, and reading that as
"unprotected" would destroy exactly what the attribute was added to guard.

Nine resources carry it today, for three different reasons:

- `algolia_api_key`, because its id *is* the credential, so a replacement is a different
  secret and everything holding the old one breaks at once.
- The five `algolia_ingestion_*` resources, because they carry live pipelines and a
  destroyed task stops moving data.
- `algolia_index`, `algolia_virtual_index` and `algolia_agent`, because the records or
  configuration behind them are not reconstructible from the Terraform configuration
  alone.

Do **not** apply it to resources the configuration fully describes, such as rules,
synonyms or dictionary entries: re-applying restores those, so a guard there only adds
friction to every configuration while protecting nothing.

Two things are easy to miss. Every path that rebuilds the model from an API response
must run the stored value through `deletionprotection.Value`, or a read turns it null
and silently unprotects the resource; putting that call in the resource's single
flatten or hydrate function covers create, read, update and import at once. And every
acceptance fixture for a guarded resource needs `deletion_protection = false`, or the
framework's own destroy step fails - which only a live acceptance run reveals.

### Secrets in state, diagnostics and logs

Mark any attribute holding a credential `Sensitive`, which is what governs how Terraform
renders it in plan output and state. It does **not** govern diagnostics or `TF_LOG`
output, and that gap is a real one: `algolia_api_key`'s `id` *is* the key, so a
diagnostic that interpolates it leaks a live credential into a terminal and a CI log.

The `apikey` package carries the pattern for that case, and any resource whose
identifier is itself a secret should follow it: `keyLabel` builds a non-secret way to
refer to the key (its operator-supplied description), `redactKey` strips the value out of
an API error before it reaches a diagnostic, and `maskKeyValue` puts it in the logging
context so `tflog` cannot print it. `resource_secret_test.go` exists to keep that honest.

The other side is CI, covered above: the acceptance and e2e jobs hold the admin API key
and therefore run only on a push to `main`. Do not widen those triggers to pull requests.

### Commits, pull requests and the CHANGELOG

- Commits follow Conventional Commits with a scope naming the service, for example
  `fix(index): confirm a deleted index is actually gone`. Commits are signed; do not
  work around a signing failure with `--no-gpg-sign`.
- Every user-visible change gets a `CHANGELOG.md` entry under `FEATURES:`,
  `BREAKING CHANGES:`, `BUG FIXES:` or `NOTES:`. Breaking entries say what an operator
  has to do differently, not just what changed. Read the state-compatibility section below
  before breaking anything state-shaped, since whether that is free depends on who is
  holding state at the time.
- Docs under `docs/` are generated by `tfplugindocs` and committed, so a schema or
  description edit needs `make generate` in the same commit.
- Keep commit messages and PR descriptions short: what was wrong, what now happens, and
  what an operator has to change. Leave reasoning, alternatives considered and how it was
  verified to the diff, the code comments and the review thread. This repository is
  intended to become public, so its history is written for a stranger reading it later,
  not for the person who reviewed it.

### Cutting a release

The version string is not stored in one place, and a missed copy leaves a snippet that
someone will paste and watch fail. `scripts/install.sh` installs the **latest** release by
default, so a document still naming the previous version sends a reader straight into
`no available releases match the given constraints`. Before tagging:

1. Date the `CHANGELOG.md` heading, replacing `(Unreleased)`.
2. Update every version reference: `README.md`, `INSTALL.md` (the `--tag` example, the two
   archive filenames, and the pin), `examples/provider/provider.tf`, the `versions.tf` of
   **every** directory under `examples/` that has one (`ecommerce-search`,
   `ingestion-pipeline`, `media-search`),
   `skills/algolia-terraform-provider/SKILL.md` plus the schema note at the top of its
   `references/attribute-shapes.md`, and this file's state-compatibility section. Find them
   with `grep -rn '<previous-version>' --include='*.md' --include='*.tf' .`

   That grep is necessary but not sufficient: a range constraint such as `~> 0.1` does not
   contain the previous version, so it never appears in the results. `media-search` was missed
   by two releases that way. Also run
   `grep -rn 'version *= *"' examples --include='*.tf'` and read every pin.
3. Run `make generate`. `docs/index.md` takes its snippet from
   `examples/provider/provider.tf`, so it follows rather than being edited.
4. Confirm the Tests workflow is green for the commit you are about to tag: the release
   workflow refuses to build otherwise.
5. Tag and push. Then install from the published release into a scratch mirror and run
   `terraform init` plus an `apply` against a disposable application, because the release
   artifact is the one thing no test in this repo exercises.

GoReleaser signs each release's `SHA256SUMS`, not the individual archives, with the key in the
`GPG_PRIVATE_KEY` and `PASSPHRASE` secrets; `INSTALL.md` records its fingerprint. When rotating
it, publish the new public half to a keyserver *before* the release that uses it, or the
signature it ships cannot be checked by anyone.

### Schema changes and state compatibility

Terraform reads stored state against the schema version that wrote it. Removing an
attribute, renaming one, or changing its type therefore breaks anyone holding older
state unless the resource's schema `Version` is raised and `UpgradeState` is
implemented. No resource declares a version yet, and every schema is at version 0.

**`0.1.1` is published**, superseding `0.1.0` and `v0.1.0-beta.1`. Because these carry no
pre-release marker, range constraints such as `~> 0.1` resolve against them where they could
never match a `-beta` build, so a configuration can now float within `0.1.x`.

Whether a state-shaped change is free therefore depends on whether any stored state was
written by a published version. That is not something this file can settle, because it
changes without the code changing: establish it before deciding, rather than inferring it
from the version number. Where nothing depends on the old shape, prefer the clean design
over a compatibility layer and record it under `BREAKING CHANGES:`. Otherwise the change
needs a schema `Version` bump with an `UpgradeState`, or a CHANGELOG entry giving the
recovery procedure, which means `terraform state rm` followed by `terraform import`.

Default to a schema `Version` bump plus an `UpgradeState`. Note that a bare re-apply is
*not* a recovery path for a removed, renamed or retyped attribute: the provider decodes
stored state against the current schema, so it can fail before an apply gets the chance
to fix anything. If you break beta state deliberately, the CHANGELOG entry has to give
the actual procedure, which means `terraform state rm` followed by `terraform import`, or
recreating the resource.

`TestSchemaSnapshot` in `internal/provider` exists to make the moment visible: it pins the shape of every
schema in `internal/provider/testdata/schema.txt` and fails on any change, so
accepting one is a deliberate act rather than something noticed after release. Accept
an intended change with `go test ./internal/provider/ -update` and read the resulting
diff as carefully as the code. Descriptions are excluded from the snapshot on purpose,
so prose edits do not train anyone to re-approve it without looking.
