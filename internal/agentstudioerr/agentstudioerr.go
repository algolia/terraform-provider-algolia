// Package agentstudioerr keeps untrusted Agent Studio API errors out of
// Terraform diagnostics.
package agentstudioerr

// Message returns safe diagnostic detail for an Agent Studio API failure.
// The downstream error is deliberately discarded because it can contain
// provider API keys, MCP headers, or other credentials stored by Agent Studio.
func Message(operation string, _ error) string {
	return "Agent Studio could not " + operation + ". Verify the configuration and credentials."
}
