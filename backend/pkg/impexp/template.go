package impexp

// BuildImportTemplate assembles an import template CSV from its columns. The
// note becomes the first cell of a row padded to the column count, so callers
// pass the note text including its leading '#' (the unmodified template is
// skipped by IsCSVRecordEmpty on re-import). Example rows are passed through
// verbatim; their first cell conventionally carries the '#' marker too, but
// that is the caller's choice.
func BuildImportTemplate(filename string, note string, headers []string, exampleRows ...[]string) *CSVFile {
	noteRow := make([]string, len(headers))
	if len(noteRow) > 0 {
		noteRow[0] = note
	}
	rows := make([][]string, 0, len(exampleRows)+1)
	rows = append(rows, noteRow)
	rows = append(rows, exampleRows...)
	return &CSVFile{
		Filename: filename,
		Headers:  headers,
		Rows:     rows,
	}
}
