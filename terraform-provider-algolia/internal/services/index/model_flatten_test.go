package index

import (
	"context"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// flattenTypoTolerance
// ---------------------------------------------------------------------------

func TestFlattenTypoTolerance(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenTypoTolerance(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("bool true returns 'true'", func(t *testing.T) {
		tt := search.BoolAsTypoTolerance(true)
		got := flattenTypoTolerance(tt)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != "true" {
			t.Errorf("expected 'true', got %q", got.ValueString())
		}
	})

	t.Run("bool false returns 'false'", func(t *testing.T) {
		tt := search.BoolAsTypoTolerance(false)
		got := flattenTypoTolerance(tt)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != "false" {
			t.Errorf("expected 'false', got %q", got.ValueString())
		}
	})

	t.Run("enum min returns 'min'", func(t *testing.T) {
		tt := search.TypoToleranceEnumAsTypoTolerance(search.TYPO_TOLERANCE_ENUM_MIN)
		got := flattenTypoTolerance(tt)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != "min" {
			t.Errorf("expected 'min', got %q", got.ValueString())
		}
	})

	t.Run("enum strict returns 'strict'", func(t *testing.T) {
		tt := search.TypoToleranceEnumAsTypoTolerance(search.TYPO_TOLERANCE_ENUM_STRICT)
		got := flattenTypoTolerance(tt)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != "strict" {
			t.Errorf("expected 'strict', got %q", got.ValueString())
		}
	})
}

// ---------------------------------------------------------------------------
// flattenDistinct
// ---------------------------------------------------------------------------

func TestFlattenDistinct(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenDistinct(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %d", got.ValueInt64())
		}
	})

	t.Run("bool true returns 1", func(t *testing.T) {
		d := search.BoolAsDistinct(true)
		got := flattenDistinct(d)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueInt64() != 1 {
			t.Errorf("expected 1, got %d", got.ValueInt64())
		}
	})

	t.Run("bool false returns 0", func(t *testing.T) {
		d := search.BoolAsDistinct(false)
		got := flattenDistinct(d)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueInt64() != 0 {
			t.Errorf("expected 0, got %d", got.ValueInt64())
		}
	})

	t.Run("int32 0", func(t *testing.T) {
		d := search.Int32AsDistinct(int32(0))
		got := flattenDistinct(d)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueInt64() != 0 {
			t.Errorf("expected 0, got %d", got.ValueInt64())
		}
	})

	t.Run("int32 1", func(t *testing.T) {
		d := search.Int32AsDistinct(int32(1))
		got := flattenDistinct(d)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueInt64() != 1 {
			t.Errorf("expected 1, got %d", got.ValueInt64())
		}
	})

	t.Run("int32 2", func(t *testing.T) {
		d := search.Int32AsDistinct(int32(2))
		got := flattenDistinct(d)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueInt64() != 2 {
			t.Errorf("expected 2, got %d", got.ValueInt64())
		}
	})
}

// ---------------------------------------------------------------------------
// flattenNullableString
// ---------------------------------------------------------------------------

func TestFlattenNullableString(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenNullableString(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		s := "hello"
		got := flattenNullableString(&s)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != "hello" {
			t.Errorf("expected 'hello', got %q", got.ValueString())
		}
	})

	t.Run("empty string returns empty string value", func(t *testing.T) {
		s := ""
		got := flattenNullableString(&s)
		if got.IsNull() {
			t.Fatal("expected non-null for empty string pointer")
		}
		if got.ValueString() != "" {
			t.Errorf("expected empty string, got %q", got.ValueString())
		}
	})
}

// ---------------------------------------------------------------------------
// flattenNullableBool
// ---------------------------------------------------------------------------

func TestFlattenNullableBool(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenNullableBool(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %v", got.ValueBool())
		}
	})

	t.Run("non-nil true", func(t *testing.T) {
		b := true
		got := flattenNullableBool(&b)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if !got.ValueBool() {
			t.Error("expected true, got false")
		}
	})

	t.Run("non-nil false", func(t *testing.T) {
		b := false
		got := flattenNullableBool(&b)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueBool() {
			t.Error("expected false, got true")
		}
	})
}

// ---------------------------------------------------------------------------
// flattenNullableInt32
// ---------------------------------------------------------------------------

func TestFlattenNullableInt32(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenNullableInt32(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %d", got.ValueInt64())
		}
	})

	t.Run("non-nil returns int64 value", func(t *testing.T) {
		v := int32(42)
		got := flattenNullableInt32(&v)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueInt64() != 42 {
			t.Errorf("expected 42, got %d", got.ValueInt64())
		}
	})

	t.Run("zero value", func(t *testing.T) {
		v := int32(0)
		got := flattenNullableInt32(&v)
		if got.IsNull() {
			t.Fatal("expected non-null for zero value")
		}
		if got.ValueInt64() != 0 {
			t.Errorf("expected 0, got %d", got.ValueInt64())
		}
	})
}

// ---------------------------------------------------------------------------
// flattenStringList
// ---------------------------------------------------------------------------

func TestFlattenStringList(t *testing.T) {
	ctx := context.Background()

	t.Run("nil returns null list", func(t *testing.T) {
		got := flattenStringList(ctx, nil)
		if !got.IsNull() {
			t.Fatal("expected null list")
		}
	})

	t.Run("empty slice returns empty list", func(t *testing.T) {
		got := flattenStringList(ctx, []string{})
		if got.IsNull() {
			t.Fatal("expected non-null list for empty slice")
		}
		if len(got.Elements()) != 0 {
			t.Fatalf("expected 0 elements, got %d", len(got.Elements()))
		}
	})

	t.Run("non-nil returns list with values", func(t *testing.T) {
		input := []string{"alpha", "beta", "gamma"}
		got := flattenStringList(ctx, input)
		if got.IsNull() {
			t.Fatal("expected non-null list")
		}
		elems := got.Elements()
		if len(elems) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(elems))
		}
		expected := []string{"alpha", "beta", "gamma"}
		for i, want := range expected {
			sv, ok := elems[i].(types.String)
			if !ok {
				t.Fatalf("element %d: expected types.String, got %T", i, elems[i])
			}
			if sv.ValueString() != want {
				t.Errorf("element %d: expected %q, got %q", i, want, sv.ValueString())
			}
		}
	})
}

// ---------------------------------------------------------------------------
// flattenSupportedLanguageList
// ---------------------------------------------------------------------------

func TestFlattenSupportedLanguageList(t *testing.T) {
	ctx := context.Background()

	t.Run("nil returns null list", func(t *testing.T) {
		got := flattenSupportedLanguageList(ctx, nil)
		if !got.IsNull() {
			t.Fatal("expected null list")
		}
	})

	t.Run("non-nil returns list with language strings", func(t *testing.T) {
		input := []search.SupportedLanguage{
			search.SupportedLanguage("en"),
			search.SupportedLanguage("de"),
		}
		got := flattenSupportedLanguageList(ctx, input)
		if got.IsNull() {
			t.Fatal("expected non-null list")
		}
		elems := got.Elements()
		if len(elems) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(elems))
		}
		expected := []string{"en", "de"}
		for i, want := range expected {
			sv, ok := elems[i].(types.String)
			if !ok {
				t.Fatalf("element %d: expected types.String, got %T", i, elems[i])
			}
			if sv.ValueString() != want {
				t.Errorf("element %d: expected %q, got %q", i, want, sv.ValueString())
			}
		}
	})
}

// ---------------------------------------------------------------------------
// flattenIgnorePlurals
// ---------------------------------------------------------------------------

func TestFlattenIgnorePlurals(t *testing.T) {
	ctx := context.Background()

	t.Run("nil sets both to null", func(t *testing.T) {
		var block LanguagesModel
		flattenIgnorePlurals(ctx, nil, &block)
		if !block.IgnorePlurals.IsNull() {
			t.Error("expected IgnorePlurals to be null")
		}
		if !block.IgnorePluralsLanguages.IsNull() {
			t.Error("expected IgnorePluralsLanguages to be null")
		}
	})

	t.Run("bool true sets IgnorePlurals=true, languages=null", func(t *testing.T) {
		ip := search.BoolAsIgnorePlurals(true)
		var block LanguagesModel
		flattenIgnorePlurals(ctx, ip, &block)
		if block.IgnorePlurals.IsNull() {
			t.Fatal("expected IgnorePlurals to be non-null")
		}
		if !block.IgnorePlurals.ValueBool() {
			t.Error("expected IgnorePlurals=true")
		}
		if !block.IgnorePluralsLanguages.IsNull() {
			t.Error("expected IgnorePluralsLanguages to be null")
		}
	})

	t.Run("bool false sets IgnorePlurals=false, languages=null", func(t *testing.T) {
		ip := search.BoolAsIgnorePlurals(false)
		var block LanguagesModel
		flattenIgnorePlurals(ctx, ip, &block)
		if block.IgnorePlurals.IsNull() {
			t.Fatal("expected IgnorePlurals to be non-null")
		}
		if block.IgnorePlurals.ValueBool() {
			t.Error("expected IgnorePlurals=false")
		}
		if !block.IgnorePluralsLanguages.IsNull() {
			t.Error("expected IgnorePluralsLanguages to be null")
		}
	})

	t.Run("languages sets IgnorePlurals=null, languages=list", func(t *testing.T) {
		langs := []search.SupportedLanguage{
			search.SupportedLanguage("en"),
			search.SupportedLanguage("fr"),
		}
		ip := search.ArrayOfSupportedLanguageAsIgnorePlurals(langs)
		var block LanguagesModel
		flattenIgnorePlurals(ctx, ip, &block)
		if !block.IgnorePlurals.IsNull() {
			t.Error("expected IgnorePlurals to be null when languages are set")
		}
		if block.IgnorePluralsLanguages.IsNull() {
			t.Fatal("expected IgnorePluralsLanguages to be non-null")
		}
		elems := block.IgnorePluralsLanguages.Elements()
		if len(elems) != 2 {
			t.Fatalf("expected 2 language elements, got %d", len(elems))
		}
	})
}

// ---------------------------------------------------------------------------
// flattenRemoveStopWords
// ---------------------------------------------------------------------------

func TestFlattenRemoveStopWords(t *testing.T) {
	ctx := context.Background()

	t.Run("nil sets both to null", func(t *testing.T) {
		var block LanguagesModel
		flattenRemoveStopWords(ctx, nil, &block)
		if !block.RemoveStopWords.IsNull() {
			t.Error("expected RemoveStopWords to be null")
		}
		if !block.RemoveStopWordsLanguages.IsNull() {
			t.Error("expected RemoveStopWordsLanguages to be null")
		}
	})

	t.Run("bool true sets RemoveStopWords=true, languages=null", func(t *testing.T) {
		rsw := search.BoolAsRemoveStopWords(true)
		var block LanguagesModel
		flattenRemoveStopWords(ctx, rsw, &block)
		if block.RemoveStopWords.IsNull() {
			t.Fatal("expected RemoveStopWords to be non-null")
		}
		if !block.RemoveStopWords.ValueBool() {
			t.Error("expected RemoveStopWords=true")
		}
		if !block.RemoveStopWordsLanguages.IsNull() {
			t.Error("expected RemoveStopWordsLanguages to be null")
		}
	})

	t.Run("languages sets RemoveStopWords=null, languages=list", func(t *testing.T) {
		langs := []search.SupportedLanguage{
			search.SupportedLanguage("de"),
			search.SupportedLanguage("es"),
		}
		rsw := search.ArrayOfSupportedLanguageAsRemoveStopWords(langs)
		var block LanguagesModel
		flattenRemoveStopWords(ctx, rsw, &block)
		if !block.RemoveStopWords.IsNull() {
			t.Error("expected RemoveStopWords to be null when languages are set")
		}
		if block.RemoveStopWordsLanguages.IsNull() {
			t.Fatal("expected RemoveStopWordsLanguages to be non-null")
		}
		elems := block.RemoveStopWordsLanguages.Elements()
		if len(elems) != 2 {
			t.Fatalf("expected 2 language elements, got %d", len(elems))
		}
	})
}

// ---------------------------------------------------------------------------
// flattenDecompoundedAttributes
// ---------------------------------------------------------------------------

func TestFlattenDecompoundedAttributes(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenDecompoundedAttributes(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("non-nil returns JSON string", func(t *testing.T) {
		da := map[string]any{
			"de": []any{"attr1", "attr2"},
		}
		got := flattenDecompoundedAttributes(da)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		// Verify it contains expected key.
		v := got.ValueString()
		if v == "" {
			t.Error("expected non-empty JSON string")
		}
	})
}

// ---------------------------------------------------------------------------
// flattenCustomNormalization
// ---------------------------------------------------------------------------

func TestFlattenCustomNormalization(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenCustomNormalization(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("non-nil returns JSON string", func(t *testing.T) {
		cn := map[string]map[string]string{
			"default": {"a": "b"},
		}
		got := flattenCustomNormalization(&cn)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() == "" {
			t.Error("expected non-empty JSON string")
		}
	})
}

// ---------------------------------------------------------------------------
// flattenUserData
// ---------------------------------------------------------------------------

func TestFlattenUserData(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenUserData(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("string value returns JSON string", func(t *testing.T) {
		got := flattenUserData("custom data")
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != `"custom data"` {
			t.Errorf("expected '\"custom data\"', got %q", got.ValueString())
		}
	})

	t.Run("map value returns JSON object", func(t *testing.T) {
		got := flattenUserData(map[string]any{"key": "value"})
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() == "" {
			t.Error("expected non-empty JSON string")
		}
	})
}

// ---------------------------------------------------------------------------
// flattenQueryType
// ---------------------------------------------------------------------------

func TestFlattenQueryType(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenQueryType(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("non-nil returns string", func(t *testing.T) {
		qt := search.QUERY_TYPE_PREFIX_LAST
		got := flattenQueryType(&qt)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != string(search.QUERY_TYPE_PREFIX_LAST) {
			t.Errorf("expected %q, got %q", string(search.QUERY_TYPE_PREFIX_LAST), got.ValueString())
		}
	})
}

// ---------------------------------------------------------------------------
// flattenMode
// ---------------------------------------------------------------------------

func TestFlattenMode(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenMode(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("non-nil returns string", func(t *testing.T) {
		m := search.MODE_NEURAL_SEARCH
		got := flattenMode(&m)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() != string(search.MODE_NEURAL_SEARCH) {
			t.Errorf("expected %q, got %q", string(search.MODE_NEURAL_SEARCH), got.ValueString())
		}
	})
}

// ---------------------------------------------------------------------------
// flattenSemanticSearch
// ---------------------------------------------------------------------------

func TestFlattenSemanticSearch(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := flattenSemanticSearch(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %q", got.ValueString())
		}
	})

	t.Run("non-nil returns JSON", func(t *testing.T) {
		ss := &search.SemanticSearch{}
		got := flattenSemanticSearch(ss)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueString() == "" {
			t.Error("expected non-empty JSON string")
		}
	})
}

