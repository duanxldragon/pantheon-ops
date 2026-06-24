package impexp

import (
	"bytes"
	"mime/multipart"
	"testing"
)

type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error { return nil }

func newMockMultipartFile(data string) multipart.File {
	return &mockMultipartFile{Reader: bytes.NewReader([]byte(data))}
}

func TestReadCSV_Empty(t *testing.T) {
	file := newMockMultipartFile("")
	rows, err := ReadCSV(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestReadCSV_SingleRow(t *testing.T) {
	file := newMockMultipartFile("a,b,c")
	rows, err := ReadCSV(file)
	if err != nil || len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("expected 1 row with 3 columns, got %d rows, %v", len(rows), err)
	}
}

func TestReadCSV_MultipleRows(t *testing.T) {
	data := "name,age\nAlice,30\nBob,25\n"
	file := newMockMultipartFile(data)
	rows, err := ReadCSV(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header+2), got %d", len(rows))
	}
}

func TestReadCSV_BOM(t *testing.T) {
	bom := "\xef\xbb\xbf"
	data := bom + "col1,col2\nval1,val2\n"
	file := newMockMultipartFile(data)
	rows, err := ReadCSV(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 || rows[0][0] != "col1" {
		t.Fatalf("expected BOM-stripped header, got %v", rows)
	}
}

func TestReadCSV_NilFile(t *testing.T) {
	func() {
		defer func() {
			if recover() != nil {
				t.Log("recovered from panic (expected with nil file)")
			}
		}()
		ReadCSV(nil)
	}()
}

func TestGovernanceExportHeaders(t *testing.T) {
	if len(GovernanceExportHeaders) == 0 {
		t.Fatal("GovernanceExportHeaders should not be empty")
	}
	expectedFirst := "governanceScope"
	if GovernanceExportHeaders[0] != expectedFirst {
		t.Fatalf("expected first header %q, got %q", expectedFirst, GovernanceExportHeaders[0])
	}
}
