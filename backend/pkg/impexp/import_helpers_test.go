package impexp

import (
	"testing"
)

func TestSplitPipeValues_Empty(t *testing.T) {
	got := SplitPipeValues("")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestSplitPipeValues_Whitespace(t *testing.T) {
	got := SplitPipeValues("  ")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestSplitPipeValues_Single(t *testing.T) {
	got := SplitPipeValues("a")
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("expected [a], got %v", got)
	}
}

func TestSplitPipeValues_Multiple(t *testing.T) {
	got := SplitPipeValues("a|b|c")
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(got), got)
	}
}

func TestSplitPipeValues_TrimsSpaces(t *testing.T) {
	got := SplitPipeValues("  a  |  b  ")
	if got[0] != "a" || got[1] != "b" {
		t.Fatalf("expected [a b], got %v", got)
	}
}

func TestSplitPipeValues_Deduplicates(t *testing.T) {
	got := SplitPipeValues("a|b|a")
	if len(got) != 2 {
		t.Fatalf("expected 2 unique items, got %d: %v", len(got), got)
	}
}

func TestSplitPipeValues_Sorts(t *testing.T) {
	got := SplitPipeValues("c|a|b")
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("expected sorted [a b c], got %v", got)
	}
}

func TestSplitPipeValues_EmptyParts(t *testing.T) {
	got := SplitPipeValues("a||b")
	if len(got) != 2 {
		t.Fatalf("expected 2 items (empty parts skipped), got %d: %v", len(got), got)
	}
}

func TestParseEnabledStatus_Default(t *testing.T) {
	if got := ParseEnabledStatus(""); got != 1 {
		t.Fatalf("expected 1 for empty, got %d", got)
	}
}

func TestParseEnabledStatus_1(t *testing.T) {
	if got := ParseEnabledStatus("1"); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestParseEnabledStatus_2(t *testing.T) {
	if got := ParseEnabledStatus("2"); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}

func TestParseEnabledStatus_Disabled(t *testing.T) {
	if got := ParseEnabledStatus("disabled"); got != 2 {
		t.Fatalf("expected 2 for \"disabled\", got %d", got)
	}
}

func TestParseEnabledStatus_Disable(t *testing.T) {
	if got := ParseEnabledStatus("disable"); got != 2 {
		t.Fatalf("expected 2 for \"disable\", got %d", got)
	}
}

func TestParseEnabledStatus_False(t *testing.T) {
	if got := ParseEnabledStatus("false"); got != 2 {
		t.Fatalf("expected 2 for \"false\", got %d", got)
	}
}

func TestParseEnabledStatus_Inactive(t *testing.T) {
	if got := ParseEnabledStatus("inactive"); got != 2 {
		t.Fatalf("expected 2 for \"inactive\", got %d", got)
	}
}

func TestParseEnabledStatus_True(t *testing.T) {
	if got := ParseEnabledStatus("true"); got != 1 {
		t.Fatalf("expected 1 for \"true\", got %d", got)
	}
}

func TestParseEnabledStatus_Whitespace(t *testing.T) {
	if got := ParseEnabledStatus("  disabled  "); got != 2 {
		t.Fatalf("expected 2 for whitespace-padded \"disabled\", got %d", got)
	}
}

func TestParseEnabledStatus_CaseInsensitive(t *testing.T) {
	if got := ParseEnabledStatus("DISABLED"); got != 2 {
		t.Fatalf("expected 2 for \"DISABLED\", got %d", got)
	}
}

func TestJoinStringSlice_Empty(t *testing.T) {
	if got := JoinStringSlice(nil, "|"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestJoinStringSlice_Single(t *testing.T) {
	if got := JoinStringSlice([]string{"a"}, "|"); got != "a" {
		t.Fatalf("expected \"a\", got %q", got)
	}
}

func TestJoinStringSlice_Multiple(t *testing.T) {
	got := JoinStringSlice([]string{"c", "a", "b"}, "|")
	if got != "a|b|c" {
		t.Fatalf("expected sorted \"a|b|c\", got %q", got)
	}
}

func TestJoinStringSlice_TrimsAndSkipsEmpty(t *testing.T) {
	got := JoinStringSlice([]string{"  a  ", "", "b"}, "|")
	if got != "a|b" {
		t.Fatalf("expected \"a|b\", got %q", got)
	}
}

func TestReadCSVField_Exists(t *testing.T) {
	headerIndex := map[string]int{"name": 0, "age": 1}
	record := []string{"Alice", "30"}
	if got := ReadCSVField(record, headerIndex, "name"); got != "Alice" {
		t.Fatalf("expected \"Alice\", got %q", got)
	}
}

func TestReadCSVField_MissingKey(t *testing.T) {
	headerIndex := map[string]int{"name": 0}
	record := []string{"Alice"}
	if got := ReadCSVField(record, headerIndex, "age"); got != "" {
		t.Fatalf("expected empty for missing key, got %q", got)
	}
}

func TestReadCSVField_IndexOutOfRange(t *testing.T) {
	headerIndex := map[string]int{"name": 5}
	record := []string{"Alice"}
	if got := ReadCSVField(record, headerIndex, "name"); got != "" {
		t.Fatalf("expected empty for out-of-range index, got %q", got)
	}
}

func TestReadCSVField_NegativeIndex(t *testing.T) {
	headerIndex := map[string]int{"name": -1}
	record := []string{"Alice"}
	if got := ReadCSVField(record, headerIndex, "name"); got != "" {
		t.Fatalf("expected empty for negative index, got %q", got)
	}
}

func TestIsCSVRecordEmpty_Empty(t *testing.T) {
	if !IsCSVRecordEmpty([]string{}) {
		t.Fatal("expected true for empty slice")
	}
}

func TestIsCSVRecordEmpty_Blank(t *testing.T) {
	if !IsCSVRecordEmpty([]string{"", "  "}) {
		t.Fatal("expected true for blank fields")
	}
}

func TestIsCSVRecordEmpty_Comment(t *testing.T) {
	if !IsCSVRecordEmpty([]string{"# comment", "data"}) {
		t.Fatal("expected true for comment record")
	}
}

func TestIsCSVRecordEmpty_FirstNonBlankIsNotComment(t *testing.T) {
	if IsCSVRecordEmpty([]string{"", "data"}) {
		t.Fatal("expected false when first non-blank is not a comment")
	}
}

func TestIsCSVRecordBlank_Empty(t *testing.T) {
	if !IsCSVRecordBlank([]string{}) {
		t.Fatal("expected true for empty slice")
	}
}

func TestIsCSVRecordBlank_AllBlank(t *testing.T) {
	if !IsCSVRecordBlank([]string{"", "  ", ""}) {
		t.Fatal("expected true for all blank fields")
	}
}

func TestIsCSVRecordBlank_HasData(t *testing.T) {
	if IsCSVRecordBlank([]string{"", "data", ""}) {
		t.Fatal("expected false when a field has data")
	}
}

func TestParseCSVInt_Empty(t *testing.T) {
	if got, _ := ParseCSVInt(""); got != 0 {
		t.Fatalf("expected 0 for empty, got %d", got)
	}
}

func TestParseCSVInt_Valid(t *testing.T) {
	if got, _ := ParseCSVInt("42"); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestParseCSVInt_Invalid(t *testing.T) {
	_, err := ParseCSVInt("not-a-number")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestParseCSVInt_Trims(t *testing.T) {
	if got, _ := ParseCSVInt("  99  "); got != 99 {
		t.Fatalf("expected 99, got %d", got)
	}
}

func TestAppendImportError(t *testing.T) {
	result := &ImportResult{}
	AppendImportError(result, 1, "name", "too short")
	if result.Failed != 1 {
		t.Fatalf("expected Failed=1, got %d", result.Failed)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Row != 1 || result.Errors[0].Field != "name" {
		t.Fatalf("unexpected error: %+v", result.Errors[0])
	}
}

func TestAppendImportError_Multiple(t *testing.T) {
	result := &ImportResult{}
	AppendImportError(result, 1, "name", "too short")
	AppendImportError(result, 2, "age", "invalid")
	if result.Failed != 2 || len(result.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d/%d", result.Failed, len(result.Errors))
	}
}

func TestImportResult_ZeroValue(t *testing.T) {
	var r ImportResult
	if r.Applied || r.Created != 0 || r.Updated != 0 || r.Failed != 0 || len(r.Errors) != 0 {
		t.Fatal("unexpected non-zero ImportResult")
	}
}

func TestBuildDeptPathMaps_NilDB(t *testing.T) {
	byID, byPath, err := BuildDeptPathMaps(nil)
	if byID != nil || byPath != nil {
		t.Fatal("expected nil maps for nil DB")
	}
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}
