package observability

import (
	"io"
	"strings"
	"time"
)

// RunQueryRequest carries transient editor text OR reviewed intent. Owner IDs,
// provider configuration and free names have no representation.
type RunQueryRequest struct {
	RawQuery    *string      `json:"rawQuery,omitempty"`
	Recipe      *QueryRecipe `json:"recipe,omitempty"`
	Time        *QueryTime   `json:"time,omitempty"`
	Limit       *int         `json:"limit,omitempty"`
	ResultMode  *string      `json:"resultMode,omitempty"`
	HistoryMode string       `json:"historyMode,omitempty"`
	HistoryID   uint64       `json:"historyId,omitempty"`
}

// SaveViewRequest persists a recipe as a saved view with a concurrency guard.
type SaveViewRequest struct {
	DatasourceID     uint64      `json:"datasourceId"`
	ExpectedRevision uint64      `json:"expectedRevision,omitempty"`
	Recipe           QueryRecipe `json:"recipe"`
}

// HistoryDTO is the read model returned for a query history entry.
type HistoryDTO struct {
	ID                    uint64       `json:"id"`
	DatasourceID          uint64       `json:"datasourceId"`
	SourceKind            string       `json:"sourceKind"`
	Time                  QueryTime    `json:"time"`
	Limit                 int          `json:"limit"`
	ResultMode            string       `json:"resultMode"`
	Outcome               string       `json:"outcome"`
	DurationMS            uint32       `json:"durationMs"`
	CreatedAt             time.Time    `json:"createdAt"`
	Recipe                *QueryRecipe `json:"recipe,omitempty"`
	ExpressionUnavailable bool         `json:"expressionUnavailable"`
}

// SavedViewDTO is the read model returned for a saved view.
type SavedViewDTO struct {
	ID           uint64      `json:"id"`
	DatasourceID uint64      `json:"datasourceId"`
	Revision     uint64      `json:"revision"`
	Recipe       QueryRecipe `json:"recipe"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// QueryCatalogField declares the operations and values available for a catalog field.
type QueryCatalogField struct {
	Field  string   `json:"field"`
	Ops    []string `json:"ops"`
	Values []string `json:"values,omitempty"`
}

// QueryCatalogDTO describes the query intent catalog for a datasource kind.
type QueryCatalogDTO struct {
	Version     uint16              `json:"version"`
	SourceKind  string              `json:"sourceKind"`
	Fields      []QueryCatalogField `json:"fields"`
	Durations   []string            `json:"durations"`
	MaxLimit    int                 `json:"maxLimit"`
	ResultModes []string            `json:"resultModes"`
}

func queryCatalog(kind string) (QueryCatalogDTO, error) {
	c := QueryCatalogDTO{Version: QueryCatalogVersion, SourceKind: kind, Durations: []string{"5m", "15m", "1h", "6h", "24h"}, MaxLimit: 1000, ResultModes: []string{resultModeTable}}
	switch kind {
	case sourceKindPrometheus, sourceKindVictoriaMetrics:
		c.Fields = []QueryCatalogField{{Field: "availability", Ops: []string{"instant"}}}
	case "loki":
		c.MaxLimit = 5000
		c.Fields = []QueryCatalogField{{Field: "stream", Ops: []string{"exists"}}, {Field: "severity", Ops: []string{"equals"}, Values: []string{"info", "warning", "error"}}}
	case sourceKindTempo:
		c.Fields = []QueryCatalogField{{Field: "service", Ops: []string{"exists"}}}
	default:
		return c, ErrQueryIntent
	}
	return c, nil
}
func decodeRunQuery(reader io.Reader, kind string, now time.Time) (RunQueryRequest, CompiledQuery, QueryTime, int, string, error) {
	var request RunQueryRequest
	if err := strictQueryJSON(reader, &request); err != nil {
		return request, CompiledQuery{}, QueryTime{}, 0, "", err
	}
	if request.HistoryMode == "" {
		request.HistoryMode = historyModeRecord
	}
	if (request.HistoryMode != historyModeRecord && request.HistoryMode != historyModeCoalesced) || (request.HistoryMode == historyModeRecord && request.HistoryID != 0) {
		return request, CompiledQuery{}, QueryTime{}, 0, "", ErrQueryIntent
	}
	if (request.RawQuery == nil) == (request.Recipe == nil) {
		return request, CompiledQuery{}, QueryTime{}, 0, "", ErrQueryIntent
	}
	if request.Recipe != nil {
		if request.Time != nil || request.Limit != nil || request.ResultMode != nil {
			return request, CompiledQuery{}, QueryTime{}, 0, "", ErrQueryIntent
		}
		compiled, err := CompileQueryRecipe(*request.Recipe, kind, now)
		return request, compiled, request.Recipe.Time, request.Recipe.Limit, request.Recipe.ResultMode, err
	}
	if kind == sourceKindTempo || request.Time == nil || request.Limit == nil || request.ResultMode == nil || *request.ResultMode != resultModeTable || len(*request.RawQuery) > 4096 || strings.TrimSpace(*request.RawQuery) == "" {
		return request, CompiledQuery{}, QueryTime{}, 0, "", ErrQueryIntent
	}
	catalog, err := queryCatalog(kind)
	if err != nil || *request.Limit <= 0 || *request.Limit > catalog.MaxLimit {
		return request, CompiledQuery{}, QueryTime{}, 0, "", ErrQueryIntent
	}
	start, end, err := queryTimeBounds(*request.Time, now)
	return request, CompiledQuery{Expression: *request.RawQuery, Start: start, End: end}, *request.Time, *request.Limit, *request.ResultMode, err
}
