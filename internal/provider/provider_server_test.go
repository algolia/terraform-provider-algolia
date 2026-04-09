package provider

import (
	"context"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProviderServerGetSchema(t *testing.T) {
	factory := providerserver.NewProtocol6WithError(New("test")())
	server, err := factory()
	if err != nil {
		t.Fatalf("factory returned error: %v", err)
	}

	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema returned error: %v", err)
	}
	if response == nil {
		t.Fatal("expected non-nil schema response")
	}
}

func TestProviderSchema_IncludesServiceSpecificRegions(t *testing.T) {
	p := &algoliaProvider{}

	var resp frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &resp)

	querySuggestionsAttr, ok := resp.Schema.Attributes["query_suggestions_region"].(providerschema.StringAttribute)
	if !ok {
		t.Fatal("expected query_suggestions_region to be a string attribute")
	}

	if !querySuggestionsAttr.Optional {
		t.Fatal("expected query_suggestions_region to be optional")
	}

	personalizationAttr, ok := resp.Schema.Attributes["personalization_region"].(providerschema.StringAttribute)
	if !ok {
		t.Fatal("expected personalization_region to be a string attribute")
	}

	if !personalizationAttr.Optional {
		t.Fatal("expected personalization_region to be optional")
	}

	if _, ok := resp.Schema.Attributes["analytics_region"]; ok {
		t.Fatal("expected analytics_region to be removed from the provider schema")
	}
}
