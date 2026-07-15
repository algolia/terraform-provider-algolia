// Package crawler is a lightweight net/http client for the Algolia Crawler API.
//
// The Crawler API is not part of the algoliasearch-client-go/v4 Go client, so it is
// hand-rolled here, mirroring the structure of internal/services/agent/client.go.
//
// Endpoints, request/response shapes, and auth were confirmed against two primary
// sources (there is no vendored/typed client to point at instead):
//
//  1. The open-source Algolia CLI's crawler client: github.com/algolia/cli,
//     api/crawler/client.go and api/crawler/types.go.
//  2. The official Crawler OpenAPI spec: github.com/algolia/api-clients-automation,
//     specs/crawler/spec.yml and specs/crawler/paths/{crawlers,crawler,crawlerConfig}.yml
//     (rendered at https://www.algolia.com/doc/rest-api/crawler/).
//
// Both sources agree on the base URL, Basic-auth scheme, and the create/get/delete/list
// shapes. The OpenAPI spec is the only source with an update/rename endpoint (the CLI has
// no "update" command), so the two PATCH methods below are sourced from it exclusively.
package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// defaultBaseURL is the Crawler API's base URL (API version 1), confirmed by both the
// algolia/cli client (DefaultBaseURL = "https://crawler.algolia.com/api/1/") and the
// OpenAPI spec's `servers` entry (https://crawler.algolia.com/api) + `/1/...` paths.
const defaultBaseURL = "https://crawler.algolia.com/api/1"

// APIError represents a non-2xx response from the Crawler API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// Client is a lightweight HTTP client for the Algolia Crawler API.
type Client struct {
	crawlerUserID string
	crawlerAPIKey string
	http          *http.Client

	// baseURL defaults to defaultBaseURL; overridden in tests to point at an httptest.Server.
	baseURL string
}

// NewClient creates a new Crawler API client.
//
// crawlerUserID and crawlerAPIKey are the Crawler-specific credentials from the
// Crawler settings page in the Algolia dashboard (https://dashboard.algolia.com/crawler/settings) —
// they are distinct from the app's regular app_id/api_key.
func NewClient(crawlerUserID, crawlerAPIKey string) *Client {
	return &Client{
		crawlerUserID: crawlerUserID,
		crawlerAPIKey: crawlerAPIKey,
		http:          &http.Client{},
		baseURL:       defaultBaseURL,
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any, query url.Values) (*http.Request, error) {
	reqURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	if len(query) > 0 {
		req.URL.RawQuery = query.Encode()
	}

	// Auth: HTTP Basic auth, base64(crawlerUserID:crawlerAPIKey) — confirmed by both
	// algolia/cli (req.SetBasicAuth(c.UserID, c.APIKey)) and the OpenAPI spec
	// ("Authorization: Basic <base64 of user-id:api-key>").
	req.SetBasicAuth(c.crawlerUserID, c.crawlerAPIKey)

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

// CreateCrawlerRequest is the request body for POST /1/crawlers.
// Source: specs/crawler/paths/crawlers.yml (operationId: createCrawler).
type CreateCrawlerRequest struct {
	Name string `json:"name"`
	// Config is the crawler configuration, modeled as raw JSON rather than a fully
	// typed struct — the crawler config schema is large and highly dynamic (start URLs,
	// sitemaps, actions/record extractors, index settings, etc.), matching this repo's
	// existing JSON-encoded-field convention (see internal/services/index).
	Config json.RawMessage `json:"config"`
}

// CreateCrawlerResponse is the response body for POST /1/crawlers.
type CreateCrawlerResponse struct {
	ID string `json:"id"`
}

// CreateCrawler creates a new crawler and returns its ID.
//
// POST /1/crawlers
// Source: specs/crawler/paths/crawlers.yml (operationId: createCrawler); matches
// algolia/cli's Client.Create.
func (c *Client) CreateCrawler(ctx context.Context, req *CreateCrawlerRequest) (*CreateCrawlerResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/crawlers", req, nil)
	if err != nil {
		return nil, err
	}

	var resp CreateCrawlerResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Crawler is a crawler and (when requested) its configuration.
// Source: specs/crawler/common/schemas/getCrawlerResponse.yml (GetCrawlerResponse /
// BaseResponse / WithConfiguration).
//
// Note: the API response body does not include the crawler's own ID (it's already known
// from the request path); GetCrawler populates ID from the id it was called with.
type Crawler struct {
	ID         string `json:"-"`
	Name       string `json:"name"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	Running    bool   `json:"running"`
	Reindexing bool   `json:"reindexing"`
	Blocked    bool   `json:"blocked"`

	BlockingError  string `json:"blockingError,omitempty"`
	BlockingTaskID string `json:"blockingTaskId,omitempty"`

	LastReindexStartedAt *string `json:"lastReindexStartAt,omitempty"`
	LastReindexEndedAt   *string `json:"lastReindexEndedAt,omitempty"`

	// Config is only present when requested via withConfig=true; raw JSON for the same
	// reason as CreateCrawlerRequest.Config.
	Config json.RawMessage `json:"config,omitempty"`
}

// GetCrawler retrieves a crawler by ID, including its full configuration.
//
// GET /1/crawlers/{id}?withConfig=true
// Source: specs/crawler/paths/crawler.yml (operationId: getCrawler) — the `withConfig`
// query parameter name is confirmed there (bool); matches algolia/cli's Client.Get.
func (c *Client) GetCrawler(ctx context.Context, id string) (*Crawler, error) {
	query := url.Values{"withConfig": {"true"}}

	httpReq, err := c.newRequest(ctx, http.MethodGet, "/crawlers/"+id, nil, query)
	if err != nil {
		return nil, err
	}

	var resp Crawler
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	resp.ID = id
	return &resp, nil
}

// ActionAcknowledgedResponse is returned by operations that queue an async crawler task
// (rename/replace-config, partial-config-update).
// Source: specs/crawler/common/schemas/responses.yml#/ActionAcknowledged.
type ActionAcknowledgedResponse struct {
	TaskID string `json:"taskId"`
}

// UpdateCrawlerRequest is the request body for PATCH /1/crawlers/{id}.
//
// Both fields are optional: send Name alone to rename the crawler, or Config to fully
// replace its configuration (this replacement is NOT versioned, unlike UpdateCrawlerConfig).
// Source: specs/crawler/paths/crawler.yml (operationId: patchCrawler): "If you only want
// to change the crawler's name, you can use this operation. For other configuration
// changes, use the 'Update configuration' endpoint instead, because changes made here
// aren't versioned. When replacing the configuration, you must provide the full
// configuration, including any settings you want to keep."
type UpdateCrawlerRequest struct {
	Name   *string         `json:"name,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// UpdateCrawler renames a crawler and/or fully replaces its configuration (unversioned).
//
// PATCH /1/crawlers/{id}
// Source: specs/crawler/paths/crawler.yml (operationId: patchCrawler). Not present in the
// algolia/cli client (it has no "update" command) — sourced from the OpenAPI spec only.
func (c *Client) UpdateCrawler(ctx context.Context, id string, req *UpdateCrawlerRequest) (*ActionAcknowledgedResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPatch, "/crawlers/"+id, req, nil)
	if err != nil {
		return nil, err
	}

	var resp ActionAcknowledgedResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateCrawlerConfig partially updates a crawler's configuration. Unlike UpdateCrawler,
// this creates a new, versioned configuration revision, and only the provided fields are
// changed (the request body is the raw partial config object itself, not wrapped in a
// {"config": ...} envelope).
//
// PATCH /1/crawlers/{id}/config
// Source: specs/crawler/paths/crawlerConfig.yml (operationId: patchConfig), request body
// schema `configuration.yml#/PartialConfig`. Not present in the algolia/cli client —
// sourced from the OpenAPI spec only.
func (c *Client) UpdateCrawlerConfig(ctx context.Context, id string, config json.RawMessage) (*ActionAcknowledgedResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPatch, "/crawlers/"+id+"/config", config, nil)
	if err != nil {
		return nil, err
	}

	var resp ActionAcknowledgedResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteCrawler deletes a crawler by ID.
//
// DELETE /1/crawlers/{id}
// Source: specs/crawler/paths/crawler.yml (operationId: deleteCrawler); matches the
// endpoint algolia/cli's `pkg/cmd/auth/crawler` etc. do not expose but the OpenAPI spec
// documents directly.
func (c *Client) DeleteCrawler(ctx context.Context, id string) error {
	httpReq, err := c.newRequest(ctx, http.MethodDelete, "/crawlers/"+id, nil, nil)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

// CrawlerListItem is a single entry in the ListCrawlers response — only id/name are
// returned per item, not the full Crawler shape.
// Source: specs/crawler/common/schemas/crawlersResponse.yml; matches algolia/cli's
// CrawlerListItem.
type CrawlerListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listCrawlersResponse struct {
	Items        []CrawlerListItem `json:"items"`
	Page         int               `json:"page"`
	ItemsPerPage int               `json:"itemsPerPage"`
	Total        int               `json:"total"`
}

// listCrawlersMaxItemsPerPage is the API's documented maximum for itemsPerPage
// (specs/crawler/common/parameters.yml#/itemsPerPage: maximum 100), used here to
// minimize the number of requests needed to page through all crawlers.
const listCrawlersMaxItemsPerPage = 100

// ListCrawlers lists all crawlers for the account, paging through every page of results.
//
// GET /1/crawlers
// Source: specs/crawler/paths/crawlers.yml (operationId: listCrawlers); matches
// algolia/cli's Client.ListAll (which pages through Client.List the same way).
func (c *Client) ListCrawlers(ctx context.Context) ([]CrawlerListItem, error) {
	var items []CrawlerListItem

	for page := 1; ; page++ {
		query := url.Values{
			"itemsPerPage": {fmt.Sprintf("%d", listCrawlersMaxItemsPerPage)},
			"page":         {fmt.Sprintf("%d", page)},
		}

		httpReq, err := c.newRequest(ctx, http.MethodGet, "/crawlers", nil, query)
		if err != nil {
			return nil, err
		}

		var resp listCrawlersResponse
		if err := c.do(httpReq, &resp); err != nil {
			return nil, err
		}

		items = append(items, resp.Items...)

		if len(resp.Items) == 0 || len(items) >= resp.Total {
			break
		}
	}

	return items, nil
}
