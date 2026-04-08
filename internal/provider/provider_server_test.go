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

func TestProviderSchema_IncludesAnalyticsRegion(t *testing.T) {
	p := &algoliaProvider{}

	var resp frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["analytics_region"].(providerschema.StringAttribute)
	if !ok {
		t.Fatal("expected analytics_region to be a string attribute")
	}

	if !attr.Optional {
		t.Fatal("expected analytics_region to be optional")
	}
}
