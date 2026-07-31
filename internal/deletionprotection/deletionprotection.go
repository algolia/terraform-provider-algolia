// Package deletionprotection holds the `deletion_protection` attribute shared by
// resources whose deletion cannot be undone.
//
// Algolia does not store this flag, so it exists only in Terraform state. That is
// what makes its default direction matter: an absent value has to read as
// protected. State written before the attribute existed, or by an import that did
// not seed it, carries no value at all, and treating that as "unprotected" would
// destroy the very resources the attribute was added to guard. Requiring an
// explicit `false` costs one apply; guessing wrong costs an API key or a data
// pipeline.
//
// The rules live here rather than in each service package because they had already
// drifted into three shapes across the resources that implemented them first, and a
// fail-safe that differs between copies is one edit away from not being one.
package deletionprotection

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Attribute returns the `deletion_protection` attribute for a resource schema.
// noun names the thing being protected in the description, as it should read in
// "prevents accidental deletion of the <noun>".
//
// It is Optional+Computed with a `true` default so that a configuration saying
// nothing gets the safe value, and so that turning protection off is an ordinary
// configuration change rather than a special gesture.
func Attribute(noun string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: "When true, prevents accidental deletion of the " + noun +
			". Must be set to false and applied before destroying.",
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(true),
	}
}

// Enabled reports whether a stored value blocks deletion. An absent or unknown
// value blocks it, for the reason given in the package comment.
func Enabled(value types.Bool) bool {
	return Value(value).ValueBool()
}

// Value resolves a stored value for writing back to state, replacing an absent one
// with the protected default.
//
// Every read path needs this. The API returns nothing for the attribute, so a model
// rebuilt from an API response has it null; writing that back would both drop the
// user's setting and leave the next Delete reading an absent value.
func Value(value types.Bool) types.Bool {
	if value.IsNull() || value.IsUnknown() {
		return types.BoolValue(true)
	}

	return value
}

// Refuse builds the diagnostic for a delete that protection blocks.
//
// subject is the complete phrase naming what could not be deleted, and reads
// directly after "Cannot delete": `fmt.Sprintf("index %q", name)` for most
// resources, or something more careful where the identifier is a secret, as it is
// for an API key. Naming it matters because a destroy covering many resources has
// to say which one stopped.
func Refuse(subject string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Deletion Protection Enabled",
		"Cannot delete "+subject+" because deletion_protection is enabled. "+
			"Set deletion_protection = false and apply before destroying.",
	)
}
