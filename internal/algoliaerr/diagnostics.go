package algoliaerr

// Subject identifies the object a resource diagnostic is about: the kind as it
// should read inside a sentence ("rule", "Recommend rule"), the object's ID, and
// optionally the parent object it lives under.
//
// Build one with Object, qualify it with In, then ask it for a message:
//
//	resp.Diagnostics.AddError(algoliaerr.Object("rule", objectID).In("index", indexName).Message(algoliaerr.Read, err))
//
// Message returns the title and detail as two values, which Go passes straight
// into AddError, so the call site still reads as an AddError and greps like one.
type Subject struct {
	kind, id             string
	parentKind, parentID string
}

// Object returns a Subject for an object identified by id on its own, such as an
// index or a rule addressed by its import ID.
func Object(kind, id string) Subject {
	return Subject{kind: kind, id: id}
}

// In returns a copy of s qualified by the parent object it lives under, so the
// detail reads "... rule R on index I: ...". Objects that are not scoped to a
// parent simply skip this.
func (s Subject) In(parentKind, parentID string) Subject {
	s.parentKind = parentKind
	s.parentID = parentID

	return s
}

// Message returns the title and detail reporting that op failed on s.
//
// Every resource in this provider phrases these the same way, and the wording is
// not the point: the point is that a fix made to one resource's error handling
// reaches its siblings. That is what this repository's history keeps getting
// wrong, so the sentence lives here once rather than in each resource.
func (s Subject) Message(op Op, err error) (title, detail string) {
	detail = "Could not " + string(op) + " " + s.kind + " " + s.id
	if s.parentKind != "" {
		detail += " on " + s.parentKind + " " + s.parentID
	}

	return "Error " + op.gerund() + " " + s.kind, detail + ": " + err.Error()
}

// WaitMessage returns the title and detail reporting that the wait for op to be
// applied to an object of the given kind failed, as opposed to op itself failing.
//
// No object ID is involved: this fires after the write was already accepted, so
// the operator is looking at whether the change landed rather than at which
// object it was. It reads naturally only for the mutating operations - Create,
// Update and Delete - since only those are waited on.
func WaitMessage(kind string, op Op, err error) (title, detail string) {
	return "Error waiting for " + kind + " " + op.noun(),
		"Could not confirm " + kind + " " + op.noun() + ": " + err.Error()
}

// Op is the operation a diagnostic reports. Its string value is the imperative
// verb used in a detail ("Could not read ..."); the constants below are the only
// supported values, and the zero value is not one of them.
type Op string

const (
	Create Op = "create"
	Read   Op = "read"
	Update Op = "update"
	Delete Op = "delete"
	Import Op = "import"
)

// gerund returns the form a diagnostic title uses: "Error reading rule".
func (op Op) gerund() string {
	switch op {
	case Create:
		return "creating"
	case Read:
		return "reading"
	case Update:
		return "updating"
	case Delete:
		return "deleting"
	case Import:
		return "importing"
	default:
		return string(op)
	}
}

// noun returns the form a wait diagnostic uses: "Error waiting for rule
// creation". Only the mutating operations have one, since only those are waited
// on; the rest fall back to the verb rather than inventing a word.
func (op Op) noun() string {
	switch op {
	case Create:
		return "creation"
	case Update:
		return "update"
	case Delete:
		return "deletion"
	default:
		return string(op)
	}
}
