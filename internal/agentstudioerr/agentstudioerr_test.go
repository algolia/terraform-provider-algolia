package agentstudioerr

import (
	"errors"
	"strings"
	"testing"
)

func TestMessageDiscardsSecretBearingDownstreamErrors(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		secret    string
	}{
		{name: "provider API key", operation: "create the provider", secret: "d0n0tl0gth1sproviderkey"},
		{name: "MCP header", operation: "create the agent", secret: "Bearer d0n0tl0gth1smcpsecret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := Message(tt.operation, errors.New("rejected credential: "+tt.secret))
			if strings.Contains(message, tt.secret) {
				t.Fatalf("Message() leaked the downstream error: %q", message)
			}
			if !strings.Contains(message, tt.operation) {
				t.Fatalf("Message() = %q, want safe operation context", message)
			}
		})
	}
}
