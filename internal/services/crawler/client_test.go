package crawler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient returns a Client wired to point at the given httptest.Server instead of
// the real crawler.algolia.com host.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c := NewClient("test-user", "test-key")
	c.baseURL = server.URL
	return c
}

func wantBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()

	got := r.Header.Get("Authorization")
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("test-user:test-key"))
	if got != want {
		t.Errorf("Authorization header = %q, want %q", got, want)
	}
}

func TestCreateCrawler(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody CreateCrawlerRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		wantBasicAuth(t, r)

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"e0f6db8a-24f5-4092-83a4-1b2c6cb6d809"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	resp, err := c.CreateCrawler(context.Background(), &CreateCrawlerRequest{
		Name:   "my-crawler",
		Config: json.RawMessage(`{"appId":"APPID","startUrls":["https://example.com"]}`),
	})
	if err != nil {
		t.Fatalf("CreateCrawler() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/crawlers" {
		t.Errorf("path = %q, want /crawlers", gotPath)
	}
	if gotBody.Name != "my-crawler" {
		t.Errorf("request body name = %q, want my-crawler", gotBody.Name)
	}
	if string(gotBody.Config) != `{"appId":"APPID","startUrls":["https://example.com"]}` {
		t.Errorf("request body config = %s, unexpected", gotBody.Config)
	}
	if resp.ID != "e0f6db8a-24f5-4092-83a4-1b2c6cb6d809" {
		t.Errorf("response ID = %q, unexpected", resp.ID)
	}
}

func TestGetCrawler(t *testing.T) {
	var gotMethod, gotPath, gotWithConfig string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotWithConfig = r.URL.Query().Get("withConfig")
		wantBasicAuth(t, r)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"name": "my-crawler",
			"createdAt": "2024-04-07T09:16:04Z",
			"updatedAt": "2024-04-07T09:16:04Z",
			"running": true,
			"reindexing": false,
			"blocked": false,
			"lastReindexStartAt": null,
			"lastReindexEndedAt": null,
			"config": {"appId":"APPID","startUrls":["https://example.com"]}
		}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	got, err := c.GetCrawler(context.Background(), "crawler-id-123")
	if err != nil {
		t.Fatalf("GetCrawler() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/crawlers/crawler-id-123" {
		t.Errorf("path = %q, want /crawlers/crawler-id-123", gotPath)
	}
	if gotWithConfig != "true" {
		t.Errorf("withConfig query param = %q, want true", gotWithConfig)
	}
	if got.ID != "crawler-id-123" {
		t.Errorf("Crawler.ID = %q, want crawler-id-123 (should be populated from the requested id)", got.ID)
	}
	if got.Name != "my-crawler" {
		t.Errorf("Crawler.Name = %q, want my-crawler", got.Name)
	}
	if !got.Running {
		t.Errorf("Crawler.Running = false, want true")
	}
	if string(got.Config) != `{"appId":"APPID","startUrls":["https://example.com"]}` {
		t.Errorf("Crawler.Config = %s, unexpected", got.Config)
	}
}

func TestUpdateCrawler(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody UpdateCrawlerRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		wantBasicAuth(t, r)

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"taskId":"98458796-b7bb-4703-8b1b-785c1080b110"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	name := "renamed-crawler"
	resp, err := c.UpdateCrawler(context.Background(), "crawler-id-123", &UpdateCrawlerRequest{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("UpdateCrawler() error = %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != "/crawlers/crawler-id-123" {
		t.Errorf("path = %q, want /crawlers/crawler-id-123", gotPath)
	}
	if gotBody.Name == nil || *gotBody.Name != "renamed-crawler" {
		t.Errorf("request body name = %v, want renamed-crawler", gotBody.Name)
	}
	if resp.TaskID != "98458796-b7bb-4703-8b1b-785c1080b110" {
		t.Errorf("response TaskID = %q, unexpected", resp.TaskID)
	}
}

func TestUpdateCrawlerConfig(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody json.RawMessage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		wantBasicAuth(t, r)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		gotBody = json.RawMessage(body)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"taskId":"98458796-b7bb-4703-8b1b-785c1080b110"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	config := json.RawMessage(`{"maxUrls":100}`)
	resp, err := c.UpdateCrawlerConfig(context.Background(), "crawler-id-123", config)
	if err != nil {
		t.Fatalf("UpdateCrawlerConfig() error = %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != "/crawlers/crawler-id-123/config" {
		t.Errorf("path = %q, want /crawlers/crawler-id-123/config", gotPath)
	}
	// The request body must be the raw config object itself, not wrapped in an envelope.
	if string(gotBody) != `{"maxUrls":100}` {
		t.Errorf("request body = %s, want raw config object {\"maxUrls\":100}", gotBody)
	}
	if resp.TaskID != "98458796-b7bb-4703-8b1b-785c1080b110" {
		t.Errorf("response TaskID = %q, unexpected", resp.TaskID)
	}
}

func TestDeleteCrawler(t *testing.T) {
	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		wantBasicAuth(t, r)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"taskId":"98458796-b7bb-4703-8b1b-785c1080b110"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	if err := c.DeleteCrawler(context.Background(), "crawler-id-123"); err != nil {
		t.Fatalf("DeleteCrawler() error = %v", err)
	}

	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/crawlers/crawler-id-123" {
		t.Errorf("path = %q, want /crawlers/crawler-id-123", gotPath)
	}
}

func TestListCrawlers_singlePage(t *testing.T) {
	var gotQueries []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantBasicAuth(t, r)
		gotQueries = append(gotQueries, r.URL.RawQuery)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [{"id":"crawler-1","name":"one"},{"id":"crawler-2","name":"two"}],
			"page": 1,
			"itemsPerPage": 100,
			"total": 2
		}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	items, err := c.ListCrawlers(context.Background())
	if err != nil {
		t.Fatalf("ListCrawlers() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "crawler-1" || items[1].ID != "crawler-2" {
		t.Errorf("items = %+v, unexpected", items)
	}
	if len(gotQueries) != 1 {
		t.Errorf("expected exactly 1 request when the first page covers the total, got %d", len(gotQueries))
	}
}

func TestListCrawlers_multiPage(t *testing.T) {
	var pages int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantBasicAuth(t, r)
		pages++

		page := r.URL.Query().Get("page")
		w.WriteHeader(http.StatusOK)
		switch page {
		case "1":
			_, _ = w.Write([]byte(`{"items":[{"id":"crawler-1","name":"one"}],"page":1,"itemsPerPage":1,"total":2}`))
		case "2":
			_, _ = w.Write([]byte(`{"items":[{"id":"crawler-2","name":"two"}],"page":2,"itemsPerPage":1,"total":2}`))
		default:
			_, _ = w.Write([]byte(`{"items":[],"page":3,"itemsPerPage":1,"total":2}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server)

	items, err := c.ListCrawlers(context.Background())
	if err != nil {
		t.Fatalf("ListCrawlers() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if pages != 2 {
		t.Errorf("expected the client to page through exactly 2 requests, got %d", pages)
	}
}

func TestAPIError_nonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Crawler not found"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	_, err := c.GetCrawler(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("GetCrawler() error = nil, want an *APIError")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
	if apiErr.Body == "" {
		t.Errorf("APIError.Body is empty, want the response body preserved")
	}
}

func TestAPIError_non2xxNon4xxStatus(t *testing.T) {
	// A 3xx that the HTTP client does not auto-follow (e.g. 304 Not Modified) must be
	// treated as an error, not silently unmarshaled as a success. Guards against the
	// >=400 check that would let 2xx-adjacent statuses through.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	c := newTestClient(t, server)

	_, err := c.GetCrawler(context.Background(), "some-id")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotModified {
		t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotModified)
	}
}

func TestAPIError_deleteNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"no rights on this crawler"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	err := c.DeleteCrawler(context.Background(), "some-id")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, http.StatusForbidden)
	}
}
