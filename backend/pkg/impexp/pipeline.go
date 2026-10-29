package impexp

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

// The shared receive half of the CSV import pipeline. Every system-domain
// import endpoint used to repeat this exact five-step block: locate the
// multipart file, open it, keep ownership with an explicit defer, parse it
// with the shared size and row caps, and answer with one of three envelope
// keys. The keys and the multipart field name are part of the observable
// contract and must not change per module.

// ImportFileField is the multipart field name every import endpoint accepts.
const ImportFileField = "file"

// Envelope message keys emitted by the shared receive path.
const (
	MsgImportFileRequired   = "import.file.required"
	MsgImportFileReadError  = "import.file.read.error"
	MsgImportFileInvalidCSV = "import.file.invalid_csv"
)

// ReadImportCSV performs the shared receive half of the import pipeline and
// returns the parsed records. On failure it writes the envelope error itself
// and returns ok=false, so callers can return immediately.
func ReadImportCSV(c *gin.Context) (records [][]string, ok bool) {
	fileHeader, err := c.FormFile(ImportFileField)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, MsgImportFileRequired)
		return nil, false
	}
	file, err := fileHeader.Open()
	if err != nil {
		common.Fail(c, common.CodeError, MsgImportFileReadError)
		return nil, false
	}
	// 显式接管 multipart part 的生命周期：ReadCSV 内部也会关闭，
	// 这里 defer 保证所有权清晰，且覆盖未来提前 return 的路径。
	defer func() { _ = file.Close() }()
	records, err = ReadCSV(file)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, MsgImportFileInvalidCSV)
		return nil, false
	}
	return records, true
}

// NewImportResult returns the empty result every import service starts from:
// not yet applied, with a non-nil error slice so the JSON envelope always
// serialises an array.
func NewImportResult() *ImportResult {
	return &ImportResult{
		Applied: false,
		Errors:  []ImportError{},
	}
}

// BuildHeaderIndex maps each trimmed header cell of the first record to its
// column index. On duplicate headers the later column wins, matching every
// module's original inline implementation.
func BuildHeaderIndex(records [][]string) map[string]int {
	headerIndex := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headerIndex[strings.TrimSpace(header)] = index
	}
	return headerIndex
}

// RequireHeaders appends an "import.header.missing" row-0 error for every
// required header absent from the index. The required list stays at the call
// site because it is module schema.
func RequireHeaders(result *ImportResult, headerIndex map[string]int, required ...string) {
	for _, header := range required {
		if _, ok := headerIndex[header]; !ok {
			AppendImportError(result, 0, header, "import.header.missing")
		}
	}
}

// ForEachDataRow walks the data rows below the header row, skipping blank and
// #-comment rows, and calls fn with the 1-based spreadsheet row number
// (header row is row 1) and the raw record.
func ForEachDataRow(records [][]string, fn func(rowNumber int, record []string)) {
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if IsCSVRecordEmpty(record) {
			continue
		}
		fn(rowIndex+1, record)
	}
}
