package scaffold

import (
	"encoding/json"
	"strings"
	"unicode"
)

type Field struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Type       string `json:"type"`
	Searchable bool   `json:"searchable"`
}

type Schema struct {
	Module string  `json:"module"`
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

type Config struct {
	Schema    Schema
	TitleName string
	LowerName string
}

func Generate(schemaJSON string) error {
	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return err
	}

	_ = Config{
		Schema:    schema,
		TitleName: capitalize(schema.Module),
		LowerName: strings.ToLower(schema.Module),
	}

	return nil
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
