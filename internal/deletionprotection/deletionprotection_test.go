package deletionprotection

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestEnabled_absentValueProtects is the whole point of the package: the flag lives
// only in Terraform state, so state that predates the attribute - or an import that
// did not seed it - carries no value, and that has to read as protected. Getting
// this direction wrong destroys exactly the resources the attribute exists to guard.
func TestEnabled_absentValueProtects(t *testing.T) {
	cases := []struct {
		name  string
		value types.Bool
		want  bool
	}{
		{name: "explicitly true", value: types.BoolValue(true), want: true},
		{name: "explicitly false", value: types.BoolValue(false), want: false},
		{name: "null, as legacy state and unseeded imports have it", value: types.BoolNull(), want: true},
		{name: "unknown", value: types.BoolUnknown(), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Enabled(tc.value); got != tc.want {
				t.Errorf("Enabled(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

// TestValue_resolvesForState covers the read path: a model rebuilt from an API
// response has the attribute null, because Algolia does not store it. Writing that
// back would drop the operator's setting and leave the next Delete reading an absent
// value, so reads resolve it instead.
func TestValue_resolvesForState(t *testing.T) {
	if got := Value(types.BoolNull()); !got.ValueBool() {
		t.Errorf("Value(null) = %v, want true so a read cannot silently unprotect", got)
	}
	if got := Value(types.BoolUnknown()); !got.ValueBool() {
		t.Errorf("Value(unknown) = %v, want true", got)
	}

	// An explicit false must survive a read, or protection could never be turned off.
	if got := Value(types.BoolValue(false)); got.IsNull() || got.ValueBool() {
		t.Errorf("Value(false) = %v, want a known false", got)
	}
	if got := Value(types.BoolValue(true)); !got.ValueBool() {
		t.Errorf("Value(true) = %v, want true", got)
	}
}

// TestRefuse_namesTheObjectAndTheWayOut keeps the diagnostic actionable: in a
// destroy covering many resources, the message has to say which one stopped and what
// to change. The subject is passed whole so each resource can name itself the way it
// already did, including where the identifier is a secret and must not be echoed.
func TestRefuse_namesTheObjectAndTheWayOut(t *testing.T) {
	d := Refuse(`index "products"`)

	if d.Severity().String() != "Error" {
		t.Errorf("severity = %s, want Error", d.Severity())
	}
	if got := d.Detail(); !strings.HasPrefix(got, `Cannot delete index "products" because`) {
		t.Errorf("detail does not read as a sentence:\n%s", got)
	}
	for _, want := range []string{`index "products"`, "deletion_protection = false"} {
		if !strings.Contains(d.Detail(), want) {
			t.Errorf("detail does not mention %q:\n%s", want, d.Detail())
		}
	}
}

// TestAttribute_defaultsToProtected guards the schema half. Optional+Computed with a
// true default is what makes a configuration that says nothing get the safe value.
func TestAttribute_defaultsToProtected(t *testing.T) {
	attr := Attribute("index")

	if !attr.Optional || !attr.Computed {
		t.Errorf("Optional = %v, Computed = %v; want both true", attr.Optional, attr.Computed)
	}
	if attr.Default == nil {
		t.Fatal("no default set; a configuration omitting the attribute would leave it unprotected")
	}
	if !strings.Contains(attr.Description, "index") {
		t.Errorf("description does not name the resource:\n%s", attr.Description)
	}
}
