package impexp

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type CSVFile struct {
	Filename string
	Headers  []string
	Rows     [][]string
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
		if err := writer.Write(row); err != nil {
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
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}
