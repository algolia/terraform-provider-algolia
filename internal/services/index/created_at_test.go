package index

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/call"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/transport"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Exercise framework planning, including the virtual resource's plan-time relink,
// rather than just checking the schema's list of modifiers.
func TestCreatedAtUpdatePlan(t *testing.T) {
	for _, tc := range []struct {
		name, resourceType string
		relink, unchanged  bool
	}{
		{name: "index settings update", resourceType: "algolia_index"},
		{name: "virtual settings update", resourceType: "algolia_virtual_index"},
		{name: "virtual relink without config change", resourceType: "algolia_virtual_index", relink: true, unchanged: true},
		{name: "unchanged index", resourceType: "algolia_index", unchanged: true},
		{name: "unchanged virtual index", resourceType: "algolia_virtual_index", unchanged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/1/indexes/primary/settings" {
					t.Errorf("unexpected API request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if tc.relink {
					_, _ = fmt.Fprint(w, `{"replicas":["test-index"]}`)
				} else {
					_, _ = fmt.Fprint(w, `{"replicas":["virtual(test-index)"]}`)
				}
			}))
			defer api.Close()
			client, err := search.NewClientWithConfig(search.SearchConfiguration{Configuration: transport.Configuration{
				AppID: "test-app", ApiKey: "test-key",
				Hosts: []transport.StatefulHost{transport.NewStatefulHost("http", api.Listener.Addr().String(), func(call.Kind) bool { return true })},
			}})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(&timestampTestProvider{client: client})()
			ctx := context.Background()
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			typ := schemas.ResourceSchemas[tc.resourceType].ValueType().(tftypes.Object)
			values := func(hits int64, state bool) map[string]tftypes.Value {
				attrs := make(map[string]tftypes.Value, len(typ.AttributeTypes))
				for key, at := range typ.AttributeTypes {
					attrs[key] = tftypes.NewValue(at, nil)
				}
				attrs["name"] = tftypes.NewValue(tftypes.String, "test-index")
				attrs["deletion_protection"] = tftypes.NewValue(tftypes.Bool, false)
				if tc.resourceType == "algolia_virtual_index" {
					attrs["primary_index_name"] = tftypes.NewValue(tftypes.String, "primary")
				}
				paginationType := typ.AttributeTypes["pagination"].(tftypes.Object)
				attrs["pagination"] = tftypes.NewValue(paginationType, map[string]tftypes.Value{
					"hits_per_page":         tftypes.NewValue(tftypes.Number, hits),
					"pagination_limited_to": tftypes.NewValue(tftypes.Number, nil),
				})
				if state {
					attrs["created_at"] = tftypes.NewValue(tftypes.String, "2026-10-28T15:51:48Z")
				}
				return attrs
			}
			encode := func(attrs map[string]tftypes.Value) *tfprotov6.DynamicValue {
				value, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, attrs))
				if err != nil {
					t.Fatal(err)
				}
				return &value
			}
			hits := int64(30)
			if tc.unchanged {
				hits = 20
			}
			plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: tc.resourceType, PriorState: encode(values(20, true)),
				Config: encode(values(hits, false)), ProposedNewState: encode(values(hits, true)),
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range plan.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("planning: %s: %s", d.Summary, d.Detail)
				}
			}
			planned, err := plan.PlannedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			var attrs map[string]tftypes.Value
			if err := planned.As(&attrs); err != nil {
				t.Fatal(err)
			}
			wantKnown := tc.unchanged && !tc.relink
			if attrs["created_at"].IsKnown() != wantKnown {
				t.Fatalf("created_at = %s, want known=%v", attrs["created_at"], wantKnown)
			}
		})
	}
}

type timestampTestProvider struct{ client *search.APIClient }

func (p *timestampTestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "algolia"
}
func (p *timestampTestProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {
}
func (p *timestampTestProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (p *timestampTestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}
func (p *timestampTestProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return &indexResource{client: p.client} },
		func() resource.Resource { return &virtualIndexResource{client: p.client} },
	}
}
