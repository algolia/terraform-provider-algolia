package analyticsregion

import "testing"

func TestNewIngestionClient(t *testing.T) {
	t.Run("valid region returns a client", func(t *testing.T) {
		for _, region := range []string{"us", "eu", "US", "  EU  "} {
			client, err := NewIngestionClient("appID", "apiKey", region)
			if err != nil {
				t.Fatalf("region %q: unexpected error: %v", region, err)
			}
			if client == nil {
				t.Fatalf("region %q: expected a non-nil client", region)
			}
		}
	})

	t.Run("empty region returns an error", func(t *testing.T) {
		client, err := NewIngestionClient("appID", "apiKey", "")
		if err == nil {
			t.Fatal("expected an error for an empty region, got nil")
		}
		if client != nil {
			t.Fatalf("expected a nil client, got %#v", client)
		}
	})

	t.Run("invalid region returns an error", func(t *testing.T) {
		client, err := NewIngestionClient("appID", "apiKey", "not-a-region")
		if err == nil {
			t.Fatal("expected an error for an invalid region, got nil")
		}
		if client != nil {
			t.Fatalf("expected a nil client, got %#v", client)
		}
	})
}
