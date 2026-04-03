package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
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
