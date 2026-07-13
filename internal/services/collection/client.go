package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ensure the custom unmarshaler is referenced.
var _ json.Unmarshaler = (*CollectionRecord)(nil)

// APIError represents a non-2xx response from the Collections API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// Client is a lightweight HTTP client for the Algolia Collections API.
type Client struct {
	appID  string
	apiKey string
	http   *http.Client
}

// NewClient creates a new Collections API client.
func NewClient(appID, apiKey string) *Client {
	return &Client{
		appID:  appID,
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

// baseURL returns the fixed Collections API base URL.
// Collections is a region-less, shared service — it does NOT use the
// appID-prefixed subdomain that most other Algolia APIs use.
func (c *Client) baseURL() string {
	return "https://experiences.algolia.com"
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

// Conditions mirrors the Conditions schema used in requests and responses.
// facetFilters and numericFilters are free-form (string, []string, [][]string, etc.),
// so they are carried as `any` and the Terraform layer handles JSON (de)serialization.
type Conditions struct {
	FacetFilters   any `json:"facetFilters,omitempty"`
	NumericFilters any `json:"numericFilters,omitempty"`
}

// UpsertRequest is the request body for POST /1/collections.
type UpsertRequest struct {
	ID          *string     `json:"id,omitempty"`
	Name        *string     `json:"name,omitempty"`
	IndexName   *string     `json:"indexName,omitempty"`
	Description *string     `json:"description,omitempty"`
	Add         []string    `json:"add,omitempty"`
	Remove      []string    `json:"remove,omitempty"`
	Conditions  *Conditions `json:"conditions,omitempty"`
	Commit      *bool       `json:"commit,omitempty"`
}

// DeleteRequest is the request body for DELETE /1/collections/{id}.
type DeleteRequest struct {
	Commit *bool `json:"commit,omitempty"`
}

// CollectionRecord is one record entry as returned by the API. Two different
// response shapes exist in the wild:
//
//   - Upsert (POST) returns objects:  [{"objectId": "..."}]
//   - Read   (GET)  returns strings:  ["...", "..."]
//
// The custom UnmarshalJSON below accepts either and normalizes to the struct.
// Any additional fields the server adds to the object form are ignored by
// encoding/json's default behavior.
type CollectionRecord struct {
	ObjectID string `json:"objectId"`
}

// UnmarshalJSON accepts either a bare string objectID or an object with an
// `objectId` field, normalizing both into CollectionRecord.ObjectID.
func (r *CollectionRecord) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return fmt.Errorf("record (string form): %w", err)
		}
		r.ObjectID = s
		return nil
	}
	var obj struct {
		ObjectID string `json:"objectId"`
	}
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return fmt.Errorf("record (object form): %w", err)
	}
	r.ObjectID = obj.ObjectID
	return nil
}

// CollectionResponse mirrors the Collection schema returned by the API.
type CollectionResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	IndexName   string             `json:"indexName"`
	Description *string            `json:"description,omitempty"`
	CreatedAt   string             `json:"createdAt"`
	UpdatedAt   *string            `json:"updatedAt,omitempty"`
	Status      *string            `json:"status,omitempty"`
	Conditions  *Conditions        `json:"conditions,omitempty"`
	Records     []CollectionRecord `json:"records,omitempty"`
}

// RecordIDs extracts just the objectIDs from a response's records, in order.
func (r *CollectionResponse) RecordIDs() []string {
	if len(r.Records) == 0 {
		return nil
	}
	ids := make([]string, len(r.Records))
	for i, rec := range r.Records {
		ids[i] = rec.ObjectID
	}
	return ids
}

// UpsertCollection creates or updates a collection.
func (c *Client) UpsertCollection(ctx context.Context, req *UpsertRequest) (*CollectionResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/1/collections", req)
	if err != nil {
		return nil, err
	}

	var resp CollectionResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCollection retrieves a collection by ID.
func (c *Client) GetCollection(ctx context.Context, id string) (*CollectionResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/1/collections/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp CollectionResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	// Preserve the ID from the URL path if the server doesn't echo it in
	// the body — the GET endpoint is known to omit it.
	if resp.ID == "" {
		resp.ID = id
	}
	return &resp, nil
}

// DeleteCollection deletes a collection by ID.
func (c *Client) DeleteCollection(ctx context.Context, id string, req *DeleteRequest) error {
	httpReq, err := c.newRequest(ctx, http.MethodDelete, "/1/collections/"+id, req)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}
