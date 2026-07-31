package algoliaerr

import (
	"sort"
	"strings"
)

// Explain renders an Algolia API error for a diagnostic, adding the per-field
// detail the API supplied when it has any.
//
// Some Algolia APIs answer an invalid request with a summary plus a list of what
// exactly was wrong, and the Go client keeps only the summary in the error's
// message. The Ingestion API is the clearest case: it replies
//
//	{"message": "Invalid payload, see error.details",
//	 "error": {"code": "invalid_payload",
//	           "details": [{"label": "policies.criticalThreshold",
//	                        "message": "'criticalThreshold' must be lower or equal to '10'"}]}}
//
// and the message alone - "Invalid payload, see error.details" - points at
// details the operator has no way to see. Naming the field and the constraint
// turns a guessing game into a fix.
//
// Falls back to err.Error() whenever there is nothing structured to add, so this
// is safe to use in place of err.Error() anywhere a client error is reported.
func Explain(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()

	details := detailLines(extras(err))
	if len(details) == 0 {
		return message
	}
	if len(details) == 1 {
		return message + " (" + details[0] + ")"
	}

	return message + "\n\n" + strings.Join(details, "\n")
}

// detailLines pulls the API's per-field messages out of an error's additional
// properties, as "label: message" lines.
//
// Everything here is defensive: these are untyped values decoded from a failure
// response, so any shape that is not the one documented on Explain yields no
// lines rather than a panic or a half-rendered sentence. A detail carrying only
// one of the two fields still contributes what it has - a message without a
// label is worth more than nothing.
func detailLines(properties map[string]any) []string {
	if len(properties) == 0 {
		return nil
	}

	apiError, ok := properties["error"].(map[string]any)
	if !ok {
		return nil
	}

	details, ok := apiError["details"].([]any)
	if !ok {
		return nil
	}

	lines := make([]string, 0, len(details))
	for _, detail := range details {
		entry, ok := detail.(map[string]any)
		if !ok {
			continue
		}

		label, _ := entry["label"].(string)
		message, _ := entry["message"].(string)

		switch {
		case label != "" && message != "":
			lines = append(lines, label+": "+message)
		case message != "":
			lines = append(lines, message)
		case label != "":
			lines = append(lines, label)
		}
	}

	// The API does not promise an order, and an unstable one would make otherwise
	// identical diagnostics differ between runs.
	sort.Strings(lines)

	return lines
}
