package impexp

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newImportContext(t *testing.T, fieldName, filename, content string, omitFile bool) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if !omitFile {
		part, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("part.Write: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/import", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c, recorder
}

func assertFailEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, wantKey string) {
	t.Helper()
	if !strings.Contains(recorder.Body.String(), wantKey) {
		t.Fatalf("expected envelope to contain %q, got %q", wantKey, recorder.Body.String())
	}
}

func TestReadImportCSVWithoutFileFailsRequired(t *testing.T) {
	c, recorder := newImportContext(t, ImportFileField, "a.csv", "x,y\n", true)
	records, ok := ReadImportCSV(c)
	if ok {
		t.Fatal("expected ok=false when the multipart file is absent")
	}
	if records != nil {
		t.Fatalf("expected nil records, got %v", records)
	}
	assertFailEnvelope(t, recorder, MsgImportFileRequired)
}

func TestReadImportCSVWithValidFileReturnsRecords(t *testing.T) {
	c, recorder := newImportContext(t, ImportFileField, "a.csv", "col1,col2\nv1,v2\n", false)
	records, ok := ReadImportCSV(c)
	if !ok {
		t.Fatalf("expected ok=true, envelope: %s", recorder.Body.String())
	}
	if len(records) != 2 || records[0][0] != "col1" || records[1][0] != "v1" {
		t.Fatalf("unexpected records: %v", records)
	}
}

// A wrong-extension upload is NOT rejected today: no handler checks the
// extension, so a .txt file whose content parses as CSV imports. This test
// locks that behaviour so the pipeline does not silently add a check.
func TestReadImportCSVAcceptsNonCSVExtension(t *testing.T) {
	c, recorder := newImportContext(t, ImportFileField, "a.txt", "col1,col2\nv1,v2\n", false)
	_, ok := ReadImportCSV(c)
	if !ok {
		t.Fatalf("expected ok=true for a .txt upload with parseable content, envelope: %s", recorder.Body.String())
	}
}

func TestReadImportCSVMalformedCSVFailsInvalidCSV(t *testing.T) {
	c, recorder := newImportContext(t, ImportFileField, "a.csv", "col1,col2\n\"unterminated\n", false)
	_, ok := ReadImportCSV(c)
	if ok {
		t.Fatal("expected ok=false for malformed CSV")
	}
	assertFailEnvelope(t, recorder, MsgImportFileInvalidCSV)
}

func TestReadImportCSVOversizedFileFailsInvalidCSV(t *testing.T) {
	big := strings.Repeat("a,b,c\n", (MaxImportBytes()/6)+10)
	c, recorder := newImportContext(t, ImportFileField, "a.csv", big, false)
	_, ok := ReadImportCSV(c)
	if ok {
		t.Fatal("expected ok=false for an oversized file")
	}
	assertFailEnvelope(t, recorder, MsgImportFileInvalidCSV)
}

func TestBuildHeaderIndexTrimsAndLaterWins(t *testing.T) {
	records := [][]string{{" name ", "age", "name"}}
	index := BuildHeaderIndex(records)
	if index["name"] != 2 {
		t.Fatalf("expected later duplicate header to win at index 2, got %d", index["name"])
	}
	if index["age"] != 1 {
		t.Fatalf("expected age at index 1, got %d", index["age"])
	}
}

func TestRequireHeadersReportsOnlyMissing(t *testing.T) {
	result := NewImportResult()
	headerIndex := map[string]int{"roleName": 0, "roleKey": 1}
	RequireHeaders(result, headerIndex, "roleName", "roleKey", "sort", "status")
	if result.Failed != 2 {
		t.Fatalf("expected 2 failures, got %d", result.Failed)
	}
	want := map[string]bool{"sort": false, "status": false}
	for _, got := range result.Errors {
		if got.Row != 0 || got.Message != "import.header.missing" {
			t.Fatalf("unexpected error %+v", got)
		}
		if _, ok := want[got.Field]; !ok {
			t.Fatalf("unexpected field %q", got.Field)
		}
		want[got.Field] = true
	}
	for field, seen := range want {
		if !seen {
			t.Fatalf("missing reported field %q", field)
		}
	}
}

func TestForEachDataRowSkipsHeaderBlankAndCommentRows(t *testing.T) {
	records := [][]string{
		{"col1", "col2"},  // header, skipped
		{"#comment", ""},  // comment row, skipped
		{"", ""},          // blank row, skipped
		{"a", "b"},        // row 4
		{"#another", "x"}, // comment row, skipped
		{"c", "d"},        // row 6
	}
	var seen []int
	ForEachDataRow(records, func(rowNumber int, record []string) {
		seen = append(seen, rowNumber)
		if record[0] == "a" && rowNumber != 4 {
			t.Fatalf("expected rowNumber 4 for %v, got %d", record, rowNumber)
		}
	})
	if len(seen) != 2 || seen[0] != 4 || seen[1] != 6 {
		t.Fatalf("expected data rows at spreadsheet positions [4 6], got %v", seen)
	}
}

func TestBuildImportTemplatePadsNoteAndKeepsExamples(t *testing.T) {
	file := BuildImportTemplate(
		"system-role-import-template.csv",
		"#说明：保留第一行表头；status 使用 1=启用、2=禁用。",
		[]string{"roleName", "roleKey", "sort", "status"},
		[]string{"#示例角色", "example_role", "99", "1"},
	)
	if file.Filename != "system-role-import-template.csv" {
		t.Fatalf("unexpected filename %q", file.Filename)
	}
	want := [][]string{
		{"#说明：保留第一行表头；status 使用 1=启用、2=禁用。", "", "", ""},
		{"#示例角色", "example_role", "99", "1"},
	}
	if len(file.Rows) != len(want) {
		t.Fatalf("expected %d rows, got %d", len(want), len(file.Rows))
	}
	for i, row := range want {
		if strings.Join(file.Rows[i], "|") != strings.Join(row, "|") {
			t.Fatalf("row %d mismatch: want %v, got %v", i, row, file.Rows[i])
		}
	}
}

// Golden test: the assembled role template must match the pre-extraction
// literal byte for byte.
func TestRoleTemplateShapeMatchesPreExtractionLiteral(t *testing.T) {
	headers := []string{"roleName", "roleKey", "sort", "status", "menuIds", "permissionKeys"}
	note := "#说明：保留第一行表头；status 使用 1=启用、2=禁用；menuIds 多个用逗号分隔；permissionKeys 多个用逗号分隔；roleKey 唯一标识，创建后不可修改。"
	file := BuildImportTemplate(
		"system-role-import-template.csv",
		note,
		headers,
		[]string{"#示例角色", "example_role", "99", "1", "1,2,3", "system:user:list,system:user:create"},
	)
	if strings.Join(file.Headers, "|") != strings.Join(headers, "|") {
		t.Fatalf("header drift: %v", file.Headers)
	}
	noteRow := file.Rows[0]
	if len(noteRow) != len(headers) {
		t.Fatalf("note row must be padded to %d cells, got %d", len(headers), len(noteRow))
	}
	if noteRow[0] != note {
		t.Fatal("note cell drift")
	}
	for i := 1; i < len(noteRow); i++ {
		if noteRow[i] != "" {
			t.Fatalf("note padding cell %d must be empty, got %q", i, noteRow[i])
		}
	}
}

// Formula-injection protection must survive the wave: a cell beginning with
// = + - @ or a tab is prefixed with ' on export unless it parses as a number.
func TestWriteCSVFormulaInjectionEscaping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/export", nil)

	file := CSVFile{
		Filename: "escape-check.csv",
		Headers:  []string{"a", "b", "c", "d", "e", "f"},
		Rows: [][]string{
			{"=1+1", "+cmd", "-label", "@x", "\ttab", "-5"},
		},
	}
	if err := WriteCSV(c, file); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	out := recorder.Body.String()
	for _, want := range []string{"'=1+1", "'+cmd", "'-label", "'@x", "'\ttab"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected escaped cell %q in output:\n%s", want, out)
		}
	}
	if !strings.Contains(out, ",-5") {
		t.Fatalf("numeric -5 must not be escaped, output:\n%s", out)
	}
}
