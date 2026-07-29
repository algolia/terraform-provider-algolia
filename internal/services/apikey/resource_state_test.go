package apikey

import (
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildApiKeyRequest(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	model := APIKeyResourceModel{
		ACL:                    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("search"), types.StringValue("browse")}),
		Description:            types.StringValue("test key"),
		ExpiresAt:              types.StringValue("2026-04-07T13:00:00Z"),
		Indexes:                types.SetValueMust(types.StringType, []attr.Value{types.StringValue("products")}),
		Referers:               types.SetValueMust(types.StringType, []attr.Value{types.StringValue("https://example.com/*")}),
		MaxHitsPerQuery:        types.Int64Value(100),
		MaxQueriesPerIPPerHour: types.Int64Value(200),
		QueryParameters:        types.StringValue("filters=tenant%3Aacme"),
	}

	apiKey, diags := buildAPIKeyRequest(&model, now)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := apiKey.GetDescription(); got != "test key" {
		t.Fatalf("description = %q, want %q", got, "test key")
	}
	if got := apiKey.GetValidity(); got != 3600 {
		t.Fatalf("validity = %d, want %d", got, 3600)
	}
	if got := apiKey.GetMaxHitsPerQuery(); got != 100 {
		t.Fatalf("max hits = %d, want %d", got, 100)
	}
	if got := apiKey.GetMaxQueriesPerIPPerHour(); got != 200 {
		t.Fatalf("max queries = %d, want %d", got, 200)
	}
	if got := apiKey.GetIndexes(); len(got) != 1 || got[0] != "products" {
		t.Fatalf("indexes = %#v, want [products]", got)
	}
	if got := apiKey.GetReferers(); len(got) != 1 || got[0] != "https://example.com/*" {
		t.Fatalf("referers = %#v, want [https://example.com/*]", got)
	}
	if got := apiKey.GetAcl(); len(got) != 2 {
		t.Fatalf("acl = %#v, want 2 entries", got)
	}
	if got := apiKey.GetQueryParameters(); got != "filters=tenant%3Aacme" {
		t.Fatalf("query parameters = %q, want %q", got, "filters=tenant%3Aacme")
	}
}

// TestBuildAPIKeyRequest_QueryParameters covers the states the attribute can
// reach a request in. UpdateApiKey resets every attribute the request does not
// carry, so what is sent has to track configuration exactly: a configured value
// is sent, a configured "" is sent too because clearing the restriction that way
// has to work, and a null - configuration omitting the attribute - is omitted so
// the reset the operator saw in the plan is what actually happens.
func TestBuildAPIKeyRequest_QueryParameters(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		value   types.String
		want    string
		wantSet bool
	}{
		{
			name:    "configured value is sent",
			value:   types.StringValue("filters=tenant%3Aacme"),
			want:    "filters=tenant%3Aacme",
			wantSet: true,
		},
		{
			name:    "explicit empty string is sent so the restriction is cleared",
			value:   types.StringValue(""),
			want:    "",
			wantSet: true,
		},
		{
			name:  "null is omitted so the planned removal is what happens",
			value: types.StringNull(),
		},
		{
			name:  "unknown is omitted",
			value: types.StringUnknown(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := APIKeyResourceModel{
				ACL:             types.SetValueMust(types.StringType, []attr.Value{types.StringValue("search")}),
				QueryParameters: test.value,
			}

			apiKey, diags := buildAPIKeyRequest(&model, now)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			got, ok := apiKey.GetQueryParametersOk()
			if ok != test.wantSet {
				t.Fatalf("query parameters present = %t, want %t", ok, test.wantSet)
			}
			if test.wantSet && *got != test.want {
				t.Fatalf("query parameters = %q, want %q", *got, test.want)
			}
		})
	}
}

// TestHydrateAPIKeyModel_QueryParameters pins the read-back. What the API
// reports is adopted, so a restriction added or changed out of band lands in
// state and shows up in the next plan. The one exception is a configured "":
// the API omits the field once it is cleared, and emitting null where the plan
// held "" makes Terraform reject the apply as an inconsistent result.
func TestHydrateAPIKeyModel_QueryParameters(t *testing.T) {
	tests := []struct {
		name     string
		prior    types.String
		response []search.GetApiKeyResponseOption
		want     types.String
	}{
		{
			name:     "API value is adopted",
			prior:    types.StringNull(),
			response: []search.GetApiKeyResponseOption{search.WithGetApiKeyResponseQueryParameters("filters=tenant%3Aacme")},
			want:     types.StringValue("filters=tenant%3Aacme"),
		},
		{
			name:     "API value replaces a stale prior",
			prior:    types.StringValue("filters=tenant%3Aold"),
			response: []search.GetApiKeyResponseOption{search.WithGetApiKeyResponseQueryParameters("filters=tenant%3Aacme")},
			want:     types.StringValue("filters=tenant%3Aacme"),
		},
		{
			name:  "absent stays null when nothing was configured",
			prior: types.StringNull(),
			want:  types.StringNull(),
		},
		{
			name:  "absent keeps a configured empty string",
			prior: types.StringValue(""),
			want:  types.StringValue(""),
		},
		{
			name:  "absent resolves an unknown prior to null",
			prior: types.StringUnknown(),
			want:  types.StringNull(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := APIKeyResourceModel{QueryParameters: test.prior}
			resp := search.NewGetApiKeyResponse("key-value", 1785269593, []search.Acl{search.ACL_SEARCH}, test.response...)

			diags := hydrateAPIKeyModel(resp, &model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if !model.QueryParameters.Equal(test.want) {
				t.Errorf("query_parameters = %s, want %s", model.QueryParameters, test.want)
			}
		})
	}
}

// TestAPIKeyResourceSchema_RestrictionsStayVisibleInThePlan pins the schema
// choice that makes the wipe-on-update hazard governable. Every attribute the
// API resets when an update omits it is Optional and not Computed, so its
// planned value is the configuration verbatim and dropping it from
// configuration shows up as a removal the operator has to accept. Marking any
// of them Computed - with or without UseStateForUnknown - would silently
// suppress that removal from the plan, which is the original defect wearing a
// safety label.
func TestAPIKeyResourceSchema_RestrictionsStayVisibleInThePlan(t *testing.T) {
	attributes := apiKeyResourceSchema().Attributes

	for _, name := range []string{"query_parameters", "indexes", "referers", "max_hits_per_query", "max_queries_per_ip_per_hour"} {
		t.Run(name, func(t *testing.T) {
			attribute, ok := attributes[name]
			if !ok {
				t.Fatalf("%s is not in the schema", name)
			}

			if !attribute.IsOptional() {
				t.Errorf("%s is not Optional; it has to be configurable", name)
			}
			if attribute.IsComputed() {
				t.Errorf("%s is Computed, so removing it from configuration would not appear in the plan and the API would reset it unannounced", name)
			}
		})
	}
}

func TestCreatedAtTimestamp(t *testing.T) {
	// Observed live: a key created at 2026-07-28T20:13:13Z reported createdAt
	// 1785269593. Read as milliseconds that is 1970-01-21T15:54:29Z.
	if got, want := createdAtTimestamp(1785269593), "2026-07-28T20:13:13Z"; got != want {
		t.Errorf("createdAtTimestamp() = %q, want %q", got, want)
	}
}

// A key the API reports without a validity does not expire, whatever the
// configuration asked for. This test previously asserted the opposite - that the
// configured expiry was preserved regardless - which is the defect it now guards
// against: state kept claiming an expiry for a key that had already been reset to
// permanent, so the discrepancy never reached a plan. Recording null instead lets
// the next plan show that configuration wants an expiry the key does not have.
func TestHydrateAPIKeyModelRecordsNonExpiringKeyAsNull(t *testing.T) {
	model := APIKeyResourceModel{
		ExpiresAt: types.StringValue("2030-01-01T00:00:00Z"),
	}

	resp := search.NewGetApiKeyResponse(
		"key-value",
		1712486400,
		[]search.Acl{search.ACL_SEARCH},
		search.WithGetApiKeyResponseDescription("desc"),
		search.WithGetApiKeyResponseIndexes([]string{"products"}),
		search.WithGetApiKeyResponseReferers([]string{"https://example.com/*"}),
	)

	diags := hydrateAPIKeyModel(resp, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := model.ID.ValueString(); got != "key-value" {
		t.Fatalf("id = %q, want %q", got, "key-value")
	}
	if got := model.Description.ValueString(); got != "desc" {
		t.Fatalf("description = %q, want %q", got, "desc")
	}
	if !model.ExpiresAt.IsNull() {
		t.Fatalf("expires_at = %q, want null for a key with no validity", model.ExpiresAt.ValueString())
	}
	if got := model.CreatedAt.ValueString(); got == "" {
		t.Fatal("created_at should be set")
	}
}

func TestFlattenExpiresAt(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	original := timeNow
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = original })

	validity := func(seconds int32) *int32 { return &seconds }

	tests := []struct {
		name        string
		prior       types.String
		validity    *int32
		hasValidity bool
		want        types.String
	}{
		{
			name:        "no validity means the key never expires",
			prior:       types.StringValue("2030-01-01T00:00:00Z"),
			hasValidity: false,
			want:        types.StringNull(),
		},
		{
			name:        "a zero validity also means the key never expires",
			prior:       types.StringValue("2030-01-01T00:00:00Z"),
			validity:    validity(0),
			hasValidity: true,
			want:        types.StringNull(),
		},
		{
			// The import case: no prior, so the expiry is derived from now+validity.
			// Leaving this null is what let a later apply silently make the key
			// permanent.
			name:        "an expiring key with no prior derives its expiry",
			prior:       types.StringNull(),
			validity:    validity(3600),
			hasValidity: true,
			want:        types.StringValue("2026-07-28T13:00:00Z"),
		},
		{
			name:        "a configured expiry is kept verbatim when it denotes the same instant",
			prior:       types.StringValue("2026-07-28T13:00:00Z"),
			validity:    validity(3600),
			hasValidity: true,
			want:        types.StringValue("2026-07-28T13:00:00Z"),
		},
		{
			// Jitter between computing a validity and the API recording it must not
			// rewrite the configured string, or every refresh would show a diff.
			name:        "a configured expiry survives jitter inside the tolerance",
			prior:       types.StringValue("2026-07-28T13:00:03Z"),
			validity:    validity(3600),
			hasValidity: true,
			want:        types.StringValue("2026-07-28T13:00:03Z"),
		},
		{
			name:        "a configured expiry in another zone is kept when the instant matches",
			prior:       types.StringValue("2026-07-28T15:00:00+02:00"),
			validity:    validity(3600),
			hasValidity: true,
			want:        types.StringValue("2026-07-28T15:00:00+02:00"),
		},
		{
			// Someone shortening the key's life out of band is real drift.
			name:        "a difference beyond the tolerance is adopted as drift",
			prior:       types.StringValue("2026-07-28T13:00:00Z"),
			validity:    validity(60),
			hasValidity: true,
			want:        types.StringValue("2026-07-28T12:01:00Z"),
		},
		{
			name:        "an unparseable prior is replaced rather than kept",
			prior:       types.StringValue("not a timestamp"),
			validity:    validity(3600),
			hasValidity: true,
			want:        types.StringValue("2026-07-28T13:00:00Z"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flattenExpiresAt(tc.prior, tc.validity, tc.hasValidity)
			if !got.Equal(tc.want) {
				t.Errorf("flattenExpiresAt() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHydrateAPIKeyModel_OptionalSetsPreservePriorEmptiness(t *testing.T) {
	emptySet := types.SetValueMust(types.StringType, []attr.Value{})
	valuedIndexes := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("products")})
	valuedReferers := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("https://example.com/*")})
	apiValues := []search.GetApiKeyResponseOption{
		search.WithGetApiKeyResponseIndexes([]string{"products"}),
		search.WithGetApiKeyResponseReferers([]string{"https://example.com/*"}),
	}

	tests := []struct {
		name         string
		prior        types.Set
		responseOpts []search.GetApiKeyResponseOption
		wantIndexes  types.Set
		wantReferers types.Set
	}{
		{
			name:         "prior null and API empty stays null",
			prior:        types.SetNull(types.StringType),
			wantIndexes:  types.SetNull(types.StringType),
			wantReferers: types.SetNull(types.StringType),
		},
		{
			name:         "prior empty and API empty stays empty",
			prior:        emptySet,
			wantIndexes:  emptySet,
			wantReferers: emptySet,
		},
		{
			name:         "API values replace a null prior",
			prior:        types.SetNull(types.StringType),
			responseOpts: apiValues,
			wantIndexes:  valuedIndexes,
			wantReferers: valuedReferers,
		},
		{
			name:         "API values replace an empty prior",
			prior:        emptySet,
			responseOpts: apiValues,
			wantIndexes:  valuedIndexes,
			wantReferers: valuedReferers,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := APIKeyResourceModel{
				Indexes:  test.prior,
				Referers: test.prior,
			}

			opts := append([]search.GetApiKeyResponseOption{}, test.responseOpts...)
			resp := search.NewGetApiKeyResponse("key-value", 1712486400, []search.Acl{search.ACL_SEARCH}, opts...)

			diags := hydrateAPIKeyModel(resp, &model)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if !model.Indexes.Equal(test.wantIndexes) {
				t.Errorf("indexes = %s, want %s", model.Indexes, test.wantIndexes)
			}
			if !model.Referers.Equal(test.wantReferers) {
				t.Errorf("referers = %s, want %s", model.Referers, test.wantReferers)
			}
		})
	}
}

func TestAPIKeyResponseMatches_WithExpiringKeyTolerance(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(2 * time.Hour)

	expected := search.NewApiKey([]search.Acl{search.ACL_BROWSE, search.ACL_SEARCH})
	expected.SetDescription("updated key")
	expected.SetIndexes([]string{"products_*"})
	expected.SetReferers([]string{"https://example.com/*"})
	expected.SetMaxHitsPerQuery(200)
	expected.SetMaxQueriesPerIPPerHour(1000)
	expected.SetValidity(int32(expiresAt.Sub(now).Seconds()))

	response := search.NewGetApiKeyResponse(
		"key-value",
		now.Unix(),
		[]search.Acl{search.ACL_SEARCH, search.ACL_BROWSE},
		search.WithGetApiKeyResponseDescription("updated key"),
		search.WithGetApiKeyResponseIndexes([]string{"products_*"}),
		search.WithGetApiKeyResponseReferers([]string{"https://example.com/*"}),
		search.WithGetApiKeyResponseMaxHitsPerQuery(200),
		search.WithGetApiKeyResponseMaxQueriesPerIPPerHour(1000),
		search.WithGetApiKeyResponseValidity(int32(expiresAt.Sub(now).Seconds()-3)),
	)

	if !apiKeyResponseMatches(response, expected, &expiresAt, now) {
		t.Fatal("expected API key response to match within validity tolerance")
	}
}

func TestAPIKeyResponseMatches_DetectsMismatch(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	expected := search.NewApiKey([]search.Acl{search.ACL_SEARCH})
	expected.SetDescription("expected")

	response := search.NewGetApiKeyResponse(
		"key-value",
		now.Unix(),
		[]search.Acl{search.ACL_SEARCH},
		search.WithGetApiKeyResponseDescription("actual"),
	)

	if apiKeyResponseMatches(response, expected, nil, now) {
		t.Fatal("expected API key response mismatch to be detected")
	}
}

// TestAPIKeyResponseMatches_DetectsQueryParametersMismatch keeps Update waiting
// until the new restriction is actually readable. Without this comparison the
// read-back could still report the previous value, and hydrating it would write
// stale state that Terraform rejects as an inconsistent apply result.
func TestAPIKeyResponseMatches_DetectsQueryParametersMismatch(t *testing.T) {
	now := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)

	expected := search.NewApiKey([]search.Acl{search.ACL_SEARCH})
	expected.SetQueryParameters("filters=tenant%3Aacme")

	stale := search.NewGetApiKeyResponse(
		"key-value",
		now.Unix(),
		[]search.Acl{search.ACL_SEARCH},
		search.WithGetApiKeyResponseQueryParameters("filters=tenant%3Aold"),
	)
	if apiKeyResponseMatches(stale, expected, nil, now) {
		t.Error("a stale query_parameters value was accepted as a match")
	}

	settled := search.NewGetApiKeyResponse(
		"key-value",
		now.Unix(),
		[]search.Acl{search.ACL_SEARCH},
		search.WithGetApiKeyResponseQueryParameters("filters=tenant%3Aacme"),
	)
	if !apiKeyResponseMatches(settled, expected, nil, now) {
		t.Error("the settled query_parameters value was not accepted as a match")
	}
}
