package collection

import (
	"encoding/json"
	"testing"
)

func TestCollectionRecord_UnmarshalJSON_StringForm(t *testing.T) {
	// Shape returned by GET /1/collections/{id}
	var rec CollectionRecord
	if err := json.Unmarshal([]byte(`"sku-123"`), &rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ObjectID != "sku-123" {
		t.Errorf("got %q, want sku-123", rec.ObjectID)
	}
}

func TestCollectionRecord_UnmarshalJSON_ObjectForm(t *testing.T) {
	// Shape returned by POST /1/collections upsert
	var rec CollectionRecord
	if err := json.Unmarshal([]byte(`{"objectId": "sku-456", "extra": "ignored"}`), &rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ObjectID != "sku-456" {
		t.Errorf("got %q, want sku-456", rec.ObjectID)
	}
}

func TestCollectionResponse_MixedRecordsDecoding(t *testing.T) {
	// Real-world: upsert response (object form).
	upsertBody := `{
      "id": "abc", "name": "n", "indexName": "i", "createdAt": "t",
      "records": [{"objectId":"a"},{"objectId":"b"}]
    }`
	var upsert CollectionResponse
	if err := json.Unmarshal([]byte(upsertBody), &upsert); err != nil {
		t.Fatalf("upsert decode: %v", err)
	}
	if ids := upsert.RecordIDs(); len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("upsert ids: %v", ids)
	}

	// Real-world: GET response (string form).
	getBody := `{
      "id": "abc", "name": "n", "indexName": "i", "createdAt": "t",
      "records": ["x","y","z"]
    }`
	var got CollectionResponse
	if err := json.Unmarshal([]byte(getBody), &got); err != nil {
		t.Fatalf("get decode: %v", err)
	}
	if ids := got.RecordIDs(); len(ids) != 3 || ids[0] != "x" || ids[2] != "z" {
		t.Errorf("get ids: %v", ids)
	}
}
