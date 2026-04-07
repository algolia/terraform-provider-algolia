package provider

import (
	"context"
	"testing"

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
