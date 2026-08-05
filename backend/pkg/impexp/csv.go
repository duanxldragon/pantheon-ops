package impexp

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// maxImportBytes 导入 CSV 的最大字节数上限（与全局 body limit 对齐的防御性兜底）。
const maxImportBytes = 10 << 20 // 10MB

// maxImportRows 单次导入允许的最大数据行数，防止超大文件拖垮逐行导入流程。
const maxImportRows = 5000

// ErrTooManyRows 表示导入文件超过最大行数限制，对应 i18n key "import.error.too_many_rows"。
var ErrTooManyRows = fmt.Errorf("import.error.too_many_rows")

type CSVFile struct {
	Filename string
	Headers  []string
	Rows     [][]string
}

// sanitizeCSVCell 防御 CSV 公式注入：以 = + - @ 或制表符/回车开头的
// 非数值单元格在 Excel 中会被当作公式执行，统一前置单引号中和。
func sanitizeCSVCell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		// 合法数值（如负数 -5、+3.14）不做转义，保证导出可读性与再导入兼容。
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return value
		}
		return "'" + value
	}
	return value
}

func sanitizeCSVRow(row []string) []string {
	sanitized := make([]string, len(row))
	for i, cell := range row {
		sanitized[i] = sanitizeCSVCell(cell)
	}
	return sanitized
}

func WriteCSV(c *gin.Context, file CSVFile) error {
	var buffer bytes.Buffer
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buffer)
	if len(file.Headers) > 0 {
		if err := writer.Write(file.Headers); err != nil {
			return err
		}
	}
	for _, row := range file.Rows {
		if err := writer.Write(sanitizeCSVRow(row)); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	filename := strings.TrimSpace(file.Filename)
	if filename == "" {
		filename = "export.csv"
	}

	escapedFilename := url.PathEscape(filename)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", filename, escapedFilename))
	c.Data(200, "text/csv; charset=utf-8", buffer.Bytes())
	return nil
}

func ReadCSV(file multipart.File) ([][]string, error) {
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxImportBytes {
		return nil, ErrTooManyRows
	}

	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) > maxImportRows+1 { // +1 表头行
		return nil, ErrTooManyRows
	}
	return records, nil
}
