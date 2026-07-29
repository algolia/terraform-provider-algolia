package algoliaerr

import (
	"errors"
	"testing"
)

// errFixture stands in for whatever the Algolia client returned. The builder must
// append its Error() verbatim, since that text is the only thing telling the
// operator what actually went wrong.
var errFixture = errors.New("index does not exist")

// TestSubjectMessage pins the exact title and detail for each operation, in both
// the with-parent and without-parent forms. These strings are user-visible in
// `terraform apply` output, so a change here is a change to the provider's
// interface and must be deliberate.
func TestSubjectMessage(t *testing.T) {
	tests := []struct {
		name       string
		subject    Subject
		op         Op
		wantTitle  string
		wantDetail string
	}{
		{
			name:       "create with parent",
			subject:    Object("rule", "my-rule").In("index", "products"),
			op:         Create,
			wantTitle:  "Error creating rule",
			wantDetail: "Could not create rule my-rule on index products: index does not exist",
		},
		{
			name:       "read with parent",
			subject:    Object("synonym", "syn-1").In("index", "products"),
			op:         Read,
			wantTitle:  "Error reading synonym",
			wantDetail: "Could not read synonym syn-1 on index products: index does not exist",
		},
		{
			name:       "update with parent",
			subject:    Object("Recommend rule", "rec-1").In("index", "products"),
			op:         Update,
			wantTitle:  "Error updating Recommend rule",
			wantDetail: "Could not update Recommend rule rec-1 on index products: index does not exist",
		},
		{
			name:       "delete with a parent that is not an index",
			subject:    Object("composition rule", "cr-1").In("composition", "my-composition"),
			op:         Delete,
			wantTitle:  "Error deleting composition rule",
			wantDetail: "Could not delete composition rule cr-1 on composition my-composition: index does not exist",
		},
		{
			name:       "create without parent",
			subject:    Object("index", "products"),
			op:         Create,
			wantTitle:  "Error creating index",
			wantDetail: "Could not create index products: index does not exist",
		},
		{
			name:       "read without parent",
			subject:    Object("index", "products"),
			op:         Read,
			wantTitle:  "Error reading index",
			wantDetail: "Could not read index products: index does not exist",
		},
		{
			name:       "update without parent",
			subject:    Object("virtual index", "products_virtual"),
			op:         Update,
			wantTitle:  "Error updating virtual index",
			wantDetail: "Could not update virtual index products_virtual: index does not exist",
		},
		{
			name:       "delete without parent",
			subject:    Object("virtual index", "products_virtual"),
			op:         Delete,
			wantTitle:  "Error deleting virtual index",
			wantDetail: "Could not delete virtual index products_virtual: index does not exist",
		},
		{
			// An import ID is composite ("products/my-rule"), so it names the parent
			// itself and must not be qualified with one.
			name:       "import of a composite ID",
			subject:    Object("rule", "products/my-rule"),
			op:         Import,
			wantTitle:  "Error importing rule",
			wantDetail: "Could not import rule products/my-rule: index does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, detail := tt.subject.Message(tt.op, errFixture)
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", detail, tt.wantDetail)
			}
		})
	}
}

func TestWaitMessage(t *testing.T) {
	tests := []struct {
		name       string
		kind       string
		op         Op
		wantTitle  string
		wantDetail string
	}{
		{
			name:       "creation",
			kind:       "rule",
			op:         Create,
			wantTitle:  "Error waiting for rule creation",
			wantDetail: "Could not confirm rule creation: index does not exist",
		},
		{
			name:       "update",
			kind:       "synonym",
			op:         Update,
			wantTitle:  "Error waiting for synonym update",
			wantDetail: "Could not confirm synonym update: index does not exist",
		},
		{
			name:       "deletion",
			kind:       "composition rule",
			op:         Delete,
			wantTitle:  "Error waiting for composition rule deletion",
			wantDetail: "Could not confirm composition rule deletion: index does not exist",
		},
		{
			name:       "multi-word kind",
			kind:       "Recommend rule",
			op:         Create,
			wantTitle:  "Error waiting for Recommend rule creation",
			wantDetail: "Could not confirm Recommend rule creation: index does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, detail := WaitMessage(tt.kind, tt.op, errFixture)
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", detail, tt.wantDetail)
			}
		})
	}
}

// TestInDoesNotMutateTheReceiver guards the value semantics that let a caller
// keep one unqualified Subject and qualify it differently per call.
func TestInDoesNotMutateTheReceiver(t *testing.T) {
	bare := Object("rule", "my-rule")
	qualified := bare.In("index", "products")

	if _, detail := bare.Message(Read, errFixture); detail != "Could not read rule my-rule: index does not exist" {
		t.Errorf("In() mutated its receiver: detail = %q", detail)
	}
	if _, detail := qualified.Message(Read, errFixture); detail != "Could not read rule my-rule on index products: index does not exist" {
		t.Errorf("detail = %q, want the parent qualifier", detail)
	}
}
