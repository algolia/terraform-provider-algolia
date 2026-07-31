package index

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestVirtualReplicaName(t *testing.T) {
	if got, want := virtualReplicaName("products_price_asc"), "virtual(products_price_asc)"; got != want {
		t.Errorf("virtualReplicaName() = %q, want %q", got, want)
	}
}

func TestIsVirtualReplicaName(t *testing.T) {
	cases := []struct {
		name  string
		entry string
		want  bool
	}{
		{name: "virtual form", entry: "virtual(products_price_asc)", want: true},
		{name: "standard replica", entry: "products_price_asc", want: false},
		{name: "name merely containing virtual", entry: "virtual_products", want: false},
		{name: "unterminated virtual form", entry: "virtual(products", want: false},
		{name: "empty", entry: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isVirtualReplicaName(tc.entry); got != tc.want {
				t.Errorf("isVirtualReplicaName(%q) = %v, want %v", tc.entry, got, tc.want)
			}
		})
	}
}

// TestLockPrimaryReplicasSerialises is the regression test for concurrent
// linking: several algolia_virtual_index resources on one primary each
// read-modify-write its replicas list, and Terraform runs them in parallel.
// The counter below stands in for that list - if the lock does not hold, the
// unsynchronised read and write interleave and the total comes out short.
func TestLockPrimaryReplicasSerialises(t *testing.T) {
	const goroutines = 50

	shared := 0
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			unlock := lockPrimaryReplicas("tf-test-primary")
			defer unlock()

			read := shared
			shared = read + 1
		}()
	}

	wg.Wait()

	if shared != goroutines {
		t.Errorf("shared counter = %d after %d locked increments, want %d", shared, goroutines, goroutines)
	}
}

// TestLockPrimaryReplicasIsPerPrimary confirms the lock does not serialise
// unrelated primaries against each other: two virtual replicas on different
// primaries touch different lists and must not wait on one another.
func TestLockPrimaryReplicasIsPerPrimary(t *testing.T) {
	first := lockPrimaryReplicas("tf-test-primary-one")
	defer first()

	acquired := make(chan struct{})
	go func() {
		unlock := lockPrimaryReplicas("tf-test-primary-two")
		defer unlock()

		close(acquired)
	}()

	<-acquired
}

// nullAttrValue builds a typed null for any attribute type, so an advanced block
// object can be assembled with only the attribute under test populated.
func nullAttrValue(ctx context.Context, t *testing.T, typ attr.Type) attr.Value {
	t.Helper()

	value, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
	if err != nil {
		t.Fatalf("building null value for %s: %v", typ, err)
	}

	return value
}

func advancedObjectWithReplicas(ctx context.Context, t *testing.T, replicas types.List) types.Object {
	t.Helper()

	attributes := make(map[string]attr.Value, len(advancedAttrTypes))
	for name, typ := range advancedAttrTypes {
		attributes[name] = nullAttrValue(ctx, t, typ)
	}
	attributes["replicas"] = replicas

	object, diags := types.ObjectValue(advancedAttrTypes, attributes)
	if diags.HasError() {
		t.Fatalf("building advanced object: %v", diags)
	}

	return object
}

// TestPreserveUndeclaredReplicas covers the apply-consistency half of not writing
// an undeclared replicas list: the read-back may legitimately carry an entry the
// plan did not, and Terraform rejects an applied value that gained one.
func TestPreserveUndeclaredReplicas(t *testing.T) {
	ctx := context.Background()

	t.Run("read-back gained an entry the plan did not promise", func(t *testing.T) {
		planned := advancedObjectWithReplicas(ctx, t, stringList(t, "virtual(a)"))
		applied := advancedObjectWithReplicas(ctx, t, stringList(t, "virtual(a)", "virtual(b)"))

		if diags := preserveUndeclaredReplicas(planned, &applied); diags.HasError() {
			t.Fatalf("preserveUndeclaredReplicas() diagnostics = %v", diags)
		}

		got, ok := applied.Attributes()["replicas"].(types.List)
		if !ok {
			t.Fatal("replicas is no longer a list")
		}
		if len(got.Elements()) != 1 {
			t.Errorf("replicas = %v, want the planned single entry", got)
		}
	})

	t.Run("unknown planned value is left alone", func(t *testing.T) {
		// Create plans replicas as unknown; writing an unknown into applied state
		// would be rejected in its own right.
		planned := advancedObjectWithReplicas(ctx, t, types.ListUnknown(types.StringType))
		applied := advancedObjectWithReplicas(ctx, t, stringList(t, "virtual(a)"))

		if diags := preserveUndeclaredReplicas(planned, &applied); diags.HasError() {
			t.Fatalf("preserveUndeclaredReplicas() diagnostics = %v", diags)
		}

		got := applied.Attributes()["replicas"].(types.List)
		if got.IsUnknown() {
			t.Error("preserveUndeclaredReplicas() wrote an unknown value into applied state")
		}
		if len(got.Elements()) != 1 {
			t.Errorf("replicas = %v, want the read-back value untouched", got)
		}
	})

	t.Run("null advanced block is a no-op", func(t *testing.T) {
		applied := advancedObjectWithReplicas(ctx, t, stringList(t, "virtual(a)"))
		before := applied

		if diags := preserveUndeclaredReplicas(types.ObjectNull(advancedAttrTypes), &applied); diags.HasError() {
			t.Fatalf("preserveUndeclaredReplicas() diagnostics = %v", diags)
		}
		if !applied.Equal(before) {
			t.Error("preserveUndeclaredReplicas() changed applied state for a null plan")
		}
	})
}

func TestUnlinkedVirtualIndexDetail(t *testing.T) {
	detail := unlinkedVirtualIndexDetail("products_price_asc", "products")

	for _, want := range []string{
		"products_price_asc",
		"primary index products",
		"virtual(products_price_asc)",
		"advanced.replicas",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail does not mention %q:\n%s", want, detail)
		}
	}
}

func TestUnlinkedVirtualIndexDetailWithoutPrimary(t *testing.T) {
	// State written before the link was known, or an import: there is no primary
	// name to name, and the sentence must still read correctly.
	detail := unlinkedVirtualIndexDetail("products_price_asc", "")

	if strings.Contains(detail, "primary index  ") || strings.Contains(detail, "primary index .") {
		t.Errorf("detail has a gap where the primary index name would go:\n%s", detail)
	}
	if !strings.Contains(detail, "virtual(products_price_asc)") {
		t.Errorf("detail does not name the missing replicas entry:\n%s", detail)
	}
}
