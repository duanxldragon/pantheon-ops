package observability

import (
	"strings"
	"testing"
	"time"
)

func TestQueryRecipeCatalogOnly(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	for _, item := range []struct{ kind, rule, expression string }{
		{"prometheus", `{"field":"availability","op":"instant"}`, "up"},
		{"victoriametrics", `{"field":"availability","op":"instant"}`, "up"},
		{"loki", `{"field":"stream","op":"exists"}`, `{service_name=~".+"}`},
		{"tempo", `{"field":"service","op":"exists"}`, `{ resource.service.name != nil }`},
	} {
		t.Run(item.kind, func(t *testing.T) {
			raw := `{"version":1,"sourceKind":"` + item.kind + `","rules":[` + item.rule + `],"time":{"mode":"relative","duration":"15m"},"limit":100,"resultMode":"table"}`
			recipe, err := DecodeQueryRecipe(strings.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			compiled, err := CompileQueryRecipe(recipe, item.kind, now)
			if err != nil || compiled.Expression != item.expression || !compiled.Start.Equal(now.Add(-15*time.Minute)) {
				t.Fatalf("unexpected compile: %+v %v", compiled, err)
			}
			if strings.Contains(string(compiled.Canonical), item.expression) && item.kind != "prometheus" && item.kind != "victoriametrics" {
				t.Fatal("provider expression entered stored canonical recipe")
			}
		})
	}
}

func TestQueryRecipeRejectsLiteralsAndAmbiguousJSON(t *testing.T) {
	valid := `{"version":1,"sourceKind":"loki","rules":[{"field":"stream","op":"exists"}],"time":{"mode":"relative","duration":"15m"},"limit":100,"resultMode":"table"}`
	bad := []string{
		strings.Replace(valid, `"version":1`, `"version":1,"version":1`, 1),
		strings.Replace(valid, `"version":1`, `"version":1,"name":"SECRET_SENTINEL"`, 1),
		strings.Replace(valid, `"op":"exists"`, `"op":"exists","value":"SECRET_SENTINEL"`, 1),
		strings.Replace(valid, `"field":"stream"`, `"field":"SECRET_SENTINEL"`, 1),
		strings.Replace(valid, `"duration":"15m"`, `"duration":"SECRET_SENTINEL"`, 1),
		strings.Replace(valid, `"rules":[`, `"rules":[{"field":"group","op":"or","rules":[]},`, 1),
		strings.Replace(valid, `"version":1`, `"version":2`, 1), valid + `{}`,
		strings.Repeat(" ", 16385) + valid,
		strings.Replace(valid, `"mode":"relative","duration":"15m"`, `"mode":"absolute","start":"2026-11-09T00:00:00Z","end":"2026-11-11T00:00:00Z"`, 1),
		strings.Replace(valid, `"limit":100`, `"limit":5001`, 1),
		strings.Replace(valid, `"sourceKind":"loki"`, `"sourceKind":"tempo"`, 1),
	}
	for i, raw := range bad {
		recipe, err := DecodeQueryRecipe(strings.NewReader(raw))
		if err == nil {
			_, err = CompileQueryRecipe(recipe, "loki", time.Now().UTC())
		}
		if err == nil {
			t.Errorf("forgery %d accepted", i)
		}
		if err != nil && strings.Contains(err.Error(), "SECRET_SENTINEL") {
			t.Errorf("forgery %d leaked literal", i)
		}
	}
}
