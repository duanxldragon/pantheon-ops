package observability

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"
)

// QueryCatalogVersion is the version of the query intent catalog encoded in a recipe.
const QueryCatalogVersion = 1
const queryBodyLimit = 16 << 10
const queryRecipeLimit = 4 << 10

// Source-kind and result-mode constants shared across the query pipeline.
const (
	sourceKindPrometheus      = "prometheus"
	sourceKindVictoriaMetrics = "victoriametrics"
	sourceKindTempo           = "tempo"
	resultModeTable           = "table"
	historyModeRecord         = "record"
	historyModeCoalesced      = "coalesced"
	queryOutcomeInvalid       = "invalid"
)

// datasourceTableName is the storage table for Prometheus datasources.
const datasourceTableName = "obs_prometheus_datasource"

// ErrQueryIntent indicates a request failed the query-intent contract validation.
var ErrQueryIntent = errors.New("param.invalid")

// QueryRecipe is an intent over code-owned catalog IDs. Provider literals and
// raw editor text have no representation in this persisted type.
type QueryRecipe struct {
	Version    uint16      `json:"version"`
	SourceKind string      `json:"sourceKind"`
	Rules      []QueryRule `json:"rules"`
	Time       QueryTime   `json:"time"`
	Limit      int         `json:"limit"`
	ResultMode string      `json:"resultMode"`
}

// QueryRule is a single catalog predicate applied to a query intent.
type QueryRule struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value,omitempty"`
}

// QueryTime is the requested time range for a query intent.
type QueryTime struct {
	Mode     string `json:"mode"`
	Duration string `json:"duration,omitempty"`
	Start    string `json:"start,omitempty"`
	End      string `json:"end,omitempty"`
}

// CompiledQuery is the validated, executable form of a query recipe.
type CompiledQuery struct {
	Expression string    `json:"-"`
	Start      time.Time `json:"-"`
	End        time.Time `json:"-"`
	Canonical  []byte    `json:"-"`
}

// strictQueryJSON rejects duplicate keys before typed decoding. DisallowUnknownFields
// alone would silently accept duplicate security-relevant fields.
func strictQueryJSON(reader io.Reader, target any) error {
	data, err := io.ReadAll(io.LimitReader(reader, queryBodyLimit+1))
	if err != nil || len(data) == 0 || len(data) > queryBodyLimit {
		return ErrQueryIntent
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var visit func(int) error
	visit = func(depth int) error {
		if depth > 5 {
			return ErrQueryIntent
		}
		token, err := d.Token()
		if err != nil {
			return ErrQueryIntent
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				name, ok := key.(string)
				if err != nil || !ok || seen[name] {
					return ErrQueryIntent
				}
				seen[name] = true
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			close, err := d.Token()
			if err != nil || close != json.Delim('}') {
				return ErrQueryIntent
			}
		case '[':
			count := 0
			for d.More() {
				count++
				if count > 16 {
					return ErrQueryIntent
				}
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			close, err := d.Token()
			if err != nil || close != json.Delim(']') {
				return ErrQueryIntent
			}
		default:
			return ErrQueryIntent
		}
		return nil
	}
	if err := visit(0); err != nil {
		return ErrQueryIntent
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrQueryIntent
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return ErrQueryIntent
	}
	return nil
}

// DecodeQueryRecipe parses and strictly validates a persisted query recipe.
func DecodeQueryRecipe(reader io.Reader) (QueryRecipe, error) {
	var recipe QueryRecipe
	if err := strictQueryJSON(reader, &recipe); err != nil {
		return recipe, err
	}
	return recipe, nil
}

func queryTimeBounds(intent QueryTime, now time.Time) (time.Time, time.Time, error) {
	switch intent.Mode {
	case "relative":
		if intent.Start != "" || intent.End != "" {
			return time.Time{}, time.Time{}, ErrQueryIntent
		}
		durations := map[string]time.Duration{"5m": 5 * time.Minute, "15m": 15 * time.Minute, "1h": time.Hour, "6h": 6 * time.Hour, "24h": 24 * time.Hour}
		duration, ok := durations[intent.Duration]
		if !ok {
			return time.Time{}, time.Time{}, ErrQueryIntent
		}
		return now.UTC().Add(-duration), now.UTC(), nil
	case "absolute":
		if intent.Duration != "" || len(intent.Start) > 30 || len(intent.End) > 30 || !strings.HasSuffix(intent.Start, "Z") || !strings.HasSuffix(intent.End, "Z") {
			return time.Time{}, time.Time{}, ErrQueryIntent
		}
		start, e1 := time.Parse(time.RFC3339Nano, intent.Start)
		end, e2 := time.Parse(time.RFC3339Nano, intent.End)
		if e1 != nil || e2 != nil || !end.After(start) || end.Sub(start) > 24*time.Hour {
			return time.Time{}, time.Time{}, ErrQueryIntent
		}
		return start.UTC(), end.UTC(), nil
	default:
		return time.Time{}, time.Time{}, ErrQueryIntent
	}
}

// CompileQueryRecipe validates a recipe and builds its executable CompiledQuery.
func CompileQueryRecipe(recipe QueryRecipe, sourceKind string, now time.Time) (CompiledQuery, error) {
	var result CompiledQuery
	if recipe.Version != QueryCatalogVersion || recipe.SourceKind != sourceKind || len(recipe.Rules) == 0 || len(recipe.Rules) > 16 || recipe.ResultMode != resultModeTable || recipe.Limit <= 0 {
		return result, ErrQueryIntent
	}
	maxLimit := 1000
	if sourceKind == "loki" {
		maxLimit = 5000
	}
	if recipe.Limit > maxLimit {
		return result, ErrQueryIntent
	}
	start, end, err := queryTimeBounds(recipe.Time, now)
	if err != nil {
		return result, err
	}
	seen := map[string]bool{}
	for _, rule := range recipe.Rules {
		if seen[rule.Field] {
			return result, ErrQueryIntent
		}
		seen[rule.Field] = true
	}
	rules := append([]QueryRule(nil), recipe.Rules...)
	sort.Slice(rules, func(i, j int) bool { return rules[i].Field < rules[j].Field })
	switch sourceKind {
	case sourceKindPrometheus, sourceKindVictoriaMetrics:
		if len(rules) != 1 || rules[0] != (QueryRule{Field: "availability", Op: "instant"}) {
			return result, ErrQueryIntent
		}
		result.Expression = "up"
	case "loki":
		if len(rules) > 2 || !seen["stream"] {
			return result, ErrQueryIntent
		}
		result.Expression = `{service_name=~".+"}`
		for _, rule := range rules {
			if rule.Field == "stream" && rule == (QueryRule{Field: "stream", Op: "exists"}) {
				continue
			}
			if rule.Field != "severity" || rule.Op != "equals" {
				return result, ErrQueryIntent
			}
			values := map[string]string{"info": "info", "warning": "warn", "error": "error"}
			value, ok := values[rule.Value]
			if !ok {
				return result, ErrQueryIntent
			}
			result.Expression = `{service_name=~".+",level="` + value + `"}`
		}
	case sourceKindTempo:
		if len(rules) != 1 || rules[0] != (QueryRule{Field: "service", Op: "exists"}) {
			return result, ErrQueryIntent
		}
		result.Expression = "{ resource.service.name != nil }"
	default:
		return result, ErrQueryIntent
	}
	recipe.Rules = rules
	if recipe.Time.Mode == "absolute" {
		recipe.Time.Start = start.Format(time.RFC3339Nano)
		recipe.Time.End = end.Format(time.RFC3339Nano)
	}
	result.Canonical, err = json.Marshal(recipe)
	if err != nil || len(result.Canonical) > queryRecipeLimit {
		return CompiledQuery{}, ErrQueryIntent
	}
	result.Start, result.End = start, end
	return result, nil
}
