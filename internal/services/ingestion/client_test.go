package ingestion

import (
	"testing"

	providertypes "github.com/algolia/terraform-provider-algolia/internal/types"
)

func TestBaseConfigureAndClient(t *testing.T) {
	t.Run("configure populates fields from ProviderData", func(t *testing.T) {
		var b base
		diags := b.configure(&providertypes.ProviderData{
			AppID:           "appID",
			APIKey:          "apiKey",
			AnalyticsRegion: "us",
		})
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}

		client, clientDiags := b.client()
		if clientDiags.HasError() {
			t.Fatalf("unexpected diagnostics building client: %v", clientDiags)
		}
		if client == nil {
			t.Fatal("expected a non-nil client")
		}
	})

	t.Run("configure is a no-op for nil ProviderData", func(t *testing.T) {
		var b base
		diags := b.configure(nil)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
	})

	t.Run("configure errors on unexpected ProviderData type", func(t *testing.T) {
		var b base
		diags := b.configure("not-provider-data")
		if !diags.HasError() {
			t.Fatal("expected an error diagnostic for an unexpected ProviderData type")
		}
	})

	t.Run("client errors without a configured region", func(t *testing.T) {
		var b base
		b.appID = "appID"
		b.apiKey = "apiKey"

		client, diags := b.client()
		if !diags.HasError() {
			t.Fatal("expected an error diagnostic for a missing analytics region")
		}
		if client != nil {
			t.Fatalf("expected a nil client, got %#v", client)
		}
	})
}
