package config

import (
	"bytes"
	"io"
	"mime/multipart"
	"testing"
)

// trackedPart wraps a multipart.File section reader and tolerates repeated
// Close calls (mirroring the os.File and multipart behavior the import
// handlers rely on when both ReadCSV and the handler's explicit defer Close
// release the same part).
type trackedPart struct {
	*io.SectionReader
	closed bool
}

func (p *trackedPart) Close() error {
	p.closed = true
	return nil
}

// openTestCSVPart builds a multipart.File over the given CSV payload the way
// c.FormFile("file").Open() would hand one to a handler. io.SectionReader
// provides the ReadAt half of multipart.File; bytes.Reader backs it so the
// payload stays a plain string in tests.
func openTestCSVPart(t *testing.T, payload string) multipart.File {
	t.Helper()
	reader := bytes.NewReader([]byte(payload))
	return &trackedPart{SectionReader: io.NewSectionReader(reader, 0, int64(reader.Len()))}
}
