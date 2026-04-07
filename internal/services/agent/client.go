package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents a non-2xx response from the Agent Studio API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// Client is a lightweight HTTP client for the Algolia Agent Studio API.
type Client struct {
	appID  string
	apiKey string
	http   *http.Client
}

// NewClient creates a new Agent Studio API client.
func NewClient(appID, apiKey string) *Client {
	return &Client{
		appID:  appID,
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

func (c *Client) baseURL() string {
	return fmt.Sprintf("https://%s.algolia.net/agent-studio/1", c.appID)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	url := c.baseURL() + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Algolia-API-Key", c.apiKey)
	req.Header.Set("X-Algolia-Application-Id", c.appID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) do(req *http.Request, result any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// AgentRequest is the request body for creating or updating an agent.
type AgentRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Instructions *string `json:"instructions,omitempty"`
	SystemPrompt *string `json:"systemPrompt,omitempty"`
	ProviderID   *string `json:"providerId,omitempty"`
	Model        *string `json:"model,omitempty"`
	TemplateType *string `json:"templateType,omitempty"`
	Config       any     `json:"config,omitempty"`
	Tools        []any   `json:"tools,omitempty"`
}

// AgentResponse is the response from the Agent Studio API.
type AgentResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Status       string  `json:"status"`
	ProviderID   *string `json:"providerId"`
	Model        *string `json:"model"`
	Instructions string  `json:"instructions"`
	SystemPrompt *string `json:"systemPrompt"`
	Config       any     `json:"config"`
	Tools        []any   `json:"tools"`
	TemplateType *string `json:"templateType"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    *string `json:"updatedAt"`
	LastUsedAt   *string `json:"lastUsedAt"`
}

// ProviderRequest is the request body for creating or updating a provider.
type ProviderRequest struct {
	Name         *string        `json:"name,omitempty"`
	ProviderName *string        `json:"providerName,omitempty"`
	Input        map[string]any `json:"input,omitempty"`
}

// ProviderResponse is the response from the Agent Studio providers API.
type ProviderResponse struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	ProviderName string         `json:"providerName"`
	Input        map[string]any `json:"input"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
	LastUsedAt   *string        `json:"lastUsedAt"`
}

type listProvidersResponse struct {
	Data []ProviderResponse `json:"data"`
}

// CreateAgent creates a new agent.
func (c *Client) CreateAgent(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/agents", req)
	if err != nil {
		return nil, err
	}

	var resp AgentResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgent retrieves an agent by ID.
func (c *Client) GetAgent(ctx context.Context, agentID string) (*AgentResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/agents/"+agentID, nil)
	if err != nil {
		return nil, err
	}

	var resp AgentResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateAgent updates an existing agent.
func (c *Client) UpdateAgent(ctx context.Context, agentID string, req *AgentRequest) (*AgentResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPatch, "/agents/"+agentID, req)
	if err != nil {
		return nil, err
	}

	var resp AgentResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAgent deletes an agent by ID.
func (c *Client) DeleteAgent(ctx context.Context, agentID string) error {
	httpReq, err := c.newRequest(ctx, http.MethodDelete, "/agents/"+agentID, nil)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

// PublishAgent publishes an agent.
func (c *Client) PublishAgent(ctx context.Context, agentID string) (*AgentResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/agents/"+agentID+"/publish", nil)
	if err != nil {
		return nil, err
	}

	var resp AgentResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateProvider creates a new provider.
func (c *Client) CreateProvider(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/providers", req)
	if err != nil {
		return nil, err
	}

	var resp ProviderResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetProvider retrieves a provider by ID.
func (c *Client) GetProvider(ctx context.Context, providerID string) (*ProviderResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/providers/"+providerID, nil)
	if err != nil {
		return nil, err
	}

	var resp ProviderResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateProvider updates an existing provider.
func (c *Client) UpdateProvider(ctx context.Context, providerID string, req *ProviderRequest) (*ProviderResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPatch, "/providers/"+providerID, req)
	if err != nil {
		return nil, err
	}

	var resp ProviderResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteProvider deletes a provider by ID.
func (c *Client) DeleteProvider(ctx context.Context, providerID string) error {
	httpReq, err := c.newRequest(ctx, http.MethodDelete, "/providers/"+providerID, nil)
	if err != nil {
		return err
	}

	return c.do(httpReq, nil)
}

// ListProviders lists all providers for the app.
func (c *Client) ListProviders(ctx context.Context) ([]ProviderResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/providers", nil)
	if err != nil {
		return nil, err
	}

	var resp listProvidersResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetProviderModels retrieves the available models for a provider.
func (c *Client) GetProviderModels(ctx context.Context, providerID string) ([]string, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/providers/"+providerID+"/models", nil)
	if err != nil {
		return nil, err
	}

	var payload json.RawMessage
	if err := c.do(httpReq, &payload); err != nil {
		return nil, err
	}

	var models []string
	if err := json.Unmarshal(payload, &models); err == nil {
		return models, nil
	}

	var wrapped struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(payload, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshaling provider models response: %w", err)
	}

	models = make([]string, 0, len(wrapped.Data))
	for _, model := range wrapped.Data {
		if value := firstNonEmptyString(model["id"], model["name"], model["slug"], model["model"]); value != "" {
			models = append(models, value)
		}
	}

	return models, nil
}

func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		if stringValue, ok := value.(string); ok && stringValue != "" {
			return stringValue
		}
	}

	return ""
}
