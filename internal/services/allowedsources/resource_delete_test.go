package allowedsources

import (
	"context"
	"errors"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestManagedSourceValues_NullSet(t *testing.T) {
	// A destroy with nothing in state must not error and must not fall back to
	// enumerating the application's allowlist.
	values, diags := managedSourceValues(context.Background(), types.SetNull(types.ObjectType{AttrTypes: sourceAttrTypes}))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(values) != 0 {
		t.Fatalf("expected no managed values, got %#v", values)
	}
}

func TestManagedSourceValues_OnlyStateEntries(t *testing.T) {
	values, diags := managedSourceValues(context.Background(), sourceSetValue(t, []SourceModel{
		{Source: types.StringValue("10.0.0.0/24"), Description: types.StringValue("b")},
		{Source: types.StringValue("1.2.3.4"), Description: types.StringNull()},
	}))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(values) != 2 || values[0] != "1.2.3.4" || values[1] != "10.0.0.0/24" {
		t.Fatalf("managed values = %#v, want [1.2.3.4 10.0.0.0/24]", values)
	}
}

// TestDeleteSources_DeletesOnlyManagedEntries is the regression test for a
// destroy that used to enumerate the whole application allowlist: only the
// values handed in - which Delete reads from state - may be deleted.
func TestDeleteSources_DeletesOnlyManagedEntries(t *testing.T) {
	var deleted []string
	err := deleteSources([]string{"1.2.3.4", "10.0.0.0/24"}, func(value string) error {
		deleted = append(deleted, value)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deleted) != 2 || deleted[0] != "1.2.3.4" || deleted[1] != "10.0.0.0/24" {
		t.Fatalf("deleted = %#v, want exactly the managed values", deleted)
	}
}

func TestDeleteSources_NoManagedEntries(t *testing.T) {
	called := false
	if err := deleteSources(nil, func(string) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if called {
		t.Fatal("expected no DeleteSource calls when state holds no sources")
	}
}

// TestDeleteSources_IgnoresAlreadyAbsentEntries covers idempotency: an entry
// removed out of band, or by a destroy that failed part-way through, must not
// make the next destroy fail.
func TestDeleteSources_IgnoresAlreadyAbsentEntries(t *testing.T) {
	var deleted []string
	err := deleteSources([]string{"1.2.3.4", "10.0.0.0/24"}, func(value string) error {
		deleted = append(deleted, value)
		if value == "1.2.3.4" {
			return &search.APIError{Status: 404, Message: "source does not exist"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error for an already-absent source: %v", err)
	}

	if len(deleted) != 2 {
		t.Fatalf("deleted = %#v, want the loop to continue past the absent source", deleted)
	}
}

func TestDeleteSources_SurfacesRealFailures(t *testing.T) {
	err := deleteSources([]string{"1.2.3.4"}, func(string) error {
		return &search.APIError{Status: 402, Message: "vault feature not enabled"}
	})
	if err == nil {
		t.Fatal("expected a non-404 API error to be surfaced")
	}

	var apiErr *search.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 402 {
		t.Fatalf("expected the original API error to be wrapped, got %v", err)
	}
}
