package observability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/security/ssrf"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DatasourceService manages metric, log, and trace data sources.
type DatasourceService struct {
	db            *gorm.DB
	ssrfValidator *ssrf.Validator
	tenantID      uint64
	tenantBound   bool
	checkpoint    func(*PrometheusDatasource) error
}

// NewDatasourceService creates a new DatasourceService.
func NewDatasourceService(db *gorm.DB) *DatasourceService {
	// Configure SSRF validator for observability datasources
	// Allow common observability ports: 80, 443, 8080, 9090 (Prometheus), 3000 (Loki/Grafana)
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = []int{80, 443, 8080, 9090, 3000, 3100, 16686} // Added 3100 for Loki, 16686 for Tempo/Jaeger

	return &DatasourceService{
		db:            db,
		ssrfValidator: validator,
	}
}

// Migrate creates or updates database schema.
func (s *DatasourceService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&PrometheusDatasource{})
}

// Create adds an observability data source.
func (s *DatasourceService) Create(dto DatasourceDTO, createdBy string, scopes ...*common.DataScopeReq) (*DatasourceResponse, error) {
	scope := firstDataScope(scopes)
	if s.tenantBound && scope == nil {
		return nil, ErrWorkspaceDenied
	}

	// SSRF Protection: Validate URL before creating datasource
	if err := s.ssrfValidator.ValidateURL(dto.URL); err != nil {
		return nil, fmt.Errorf("observability.datasource_invalid_url: %w", err)
	}

	// Encrypt sensitive fields
	encryptedPassword, err := secrets.Encrypt(dto.Password)
	if err != nil {
		return nil, fmt.Errorf("encryption_failed: %w", err)
	}
	encryptedToken, err := secrets.Encrypt(dto.BearerToken)
	if err != nil {
		return nil, fmt.Errorf("encryption_failed: %w", err)
	}

	record := &PrometheusDatasource{
		TenantID: s.tenantID, Revision: 1,
		DeptID:      ownedDeptID(scope),
		Name:        dto.Name,
		Type:        dto.Type,
		URL:         dto.URL,
		AccessMode:  dto.AccessMode,
		Username:    dto.Username,
		Password:    encryptedPassword,
		BearerToken: encryptedToken,
		Timeout:     dto.Timeout,
		IsDefault:   dto.IsDefault,
		Status:      "unknown",
		CreatedBy:   createdBy,
	}

	if record.AccessMode == "" {
		record.AccessMode = "proxy"
	}
	if record.Timeout == 0 {
		record.Timeout = 30
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if s.tenantBound {
			if err := lockQueryOwners(tx, []QueryOwner{{TenantID: s.tenantID, ActorID: 0}}); err != nil {
				return err
			}
		}
		if dto.IsDefault {
			if err := tx.Model(&PrometheusDatasource{}).Where("tenant_id = ?", s.tenantID).Scopes(database.WithDataScope(scope)).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(record).Error
	}); err != nil {
		return nil, err
	}

	// Test connection in background
	s.startConnectionTest(record.ID, record.Revision, scope)

	return s.toResponse(record), nil
}

// List returns all observability data sources.
func (s *DatasourceService) List(scopes ...*common.DataScopeReq) (*DatasourceListResponse, error) {
	if s.tenantBound && firstDataScope(scopes) == nil {
		return nil, ErrWorkspaceDenied
	}
	var records []PrometheusDatasource
	if err := s.tenantDB().Scopes(database.WithDataScope(firstDataScope(scopes))).Order("is_default DESC, created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]DatasourceResponse, len(records))
	for i, r := range records {
		items[i] = *s.toResponse(&r)
	}

	return &DatasourceListResponse{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// GetByID retrieves a data source by ID.
func (s *DatasourceService) GetByID(id uint64, scopes ...*common.DataScopeReq) (*DatasourceResponse, error) {
	if s.tenantBound && firstDataScope(scopes) == nil {
		return nil, ErrWorkspaceDenied
	}
	var record PrometheusDatasource
	if err := s.tenantDB().Scopes(database.WithDataScope(firstDataScope(scopes))).First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toResponse(&record), nil
}

// Update modifies an existing data source.
func (s *DatasourceService) Update(id uint64, dto DatasourceDTO, scopes ...*common.DataScopeReq) error {
	scope := firstDataScope(scopes)
	if s.tenantBound && scope == nil {
		return ErrWorkspaceDenied
	}

	// SSRF Protection: Validate URL before updating datasource
	if err := s.ssrfValidator.ValidateURL(dto.URL); err != nil {
		return fmt.Errorf("observability.datasource_invalid_url: %w", err)
	}

	updates := map[string]interface{}{
		"name":        dto.Name,
		"type":        dto.Type,
		"url":         dto.URL,
		"access_mode": dto.AccessMode,
		"username":    dto.Username,
		"timeout":     dto.Timeout,
		"is_default":  dto.IsDefault,
		"revision":    gorm.Expr("revision + 1"),
	}

	if dto.Password != "" {
		encrypted, err := secrets.Encrypt(dto.Password)
		if err != nil {
			return fmt.Errorf("encryption_failed: %w", err)
		}
		updates["password"] = encrypted
	}

	if dto.BearerToken != "" {
		encrypted, err := secrets.Encrypt(dto.BearerToken)
		if err != nil {
			return fmt.Errorf("encryption_failed: %w", err)
		}
		updates["bearer_token"] = encrypted
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if s.tenantBound {
			if err := lockQueryOwners(tx, []QueryOwner{{TenantID: s.tenantID, ActorID: 0}}); err != nil {
				return err
			}
		}
		var record PrometheusDatasource
		if err := tx.Where("tenant_id = ?", s.tenantID).Scopes(database.WithDataScope(scope)).Clauses(clause.Locking{Strength: "UPDATE"}).First(&record, id).Error; err != nil {
			return err
		}
		if dto.IsDefault {
			if err := tx.Model(&PrometheusDatasource{}).Where("tenant_id = ?", s.tenantID).Scopes(database.WithDataScope(scope)).Where("is_default = ? AND id != ?", true, id).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Model(&record).Updates(updates).Error
	}); err != nil {
		return err
	}

	// Test connection in background
	var current PrometheusDatasource
	if err := s.tenantDB().First(&current, id).Error; err != nil {
		return err
	}
	s.startConnectionTest(id, current.Revision, scope)

	return nil
}

// Delete removes a data source.
func (s *DatasourceService) Delete(id uint64, scopes ...*common.DataScopeReq) error {
	scope := firstDataScope(scopes)
	if s.tenantBound && scope == nil {
		return ErrWorkspaceDenied
	}
	if s.tenantBound {
		return s.purgeDatasource(id, scope)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var record PrometheusDatasource
		if err := tx.Where("tenant_id = ?", s.tenantID).Scopes(database.WithDataScope(scope)).Clauses(clause.Locking{Strength: "UPDATE"}).First(&record, id).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&AlertRule{}).Where("datasource_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("observability.datasource_in_use")
		}
		return tx.Delete(&record).Error
	})
}

// TestConnection tests connectivity to an observability data source.
func (s *DatasourceService) TestConnection(id uint64, scopes ...*common.DataScopeReq) (bool, error) {
	if s.tenantBound && firstDataScope(scopes) == nil {
		return false, ErrWorkspaceDenied
	}
	var record PrometheusDatasource
	if err := s.tenantDB().Scopes(database.WithDataScope(firstDataScope(scopes))).First(&record, id).Error; err != nil {
		return false, err
	}

	return s.testConnectionRecord(&record)
}

func (s *DatasourceService) testConnectionRecord(record *PrometheusDatasource) (bool, error) {
	if err := s.recheckRecord(record); err != nil {
		return false, err
	}
	// Decrypt credentials
	password, err := secrets.Decrypt(record.Password)
	if err != nil {
		return false, err
	}
	bearerToken, err := secrets.Decrypt(record.BearerToken)
	if err != nil {
		return false, err
	}

	// Test connection
	healthy, err := testDatasourceConnectionContext(s.db.Statement.Context, s.ssrfValidator, record.Type, record.URL, record.Username, password, bearerToken, record.Timeout, func() error { return s.recheckRecord(record) })
	if checkErr := s.recheckRecord(record); checkErr != nil {
		return false, checkErr
	}

	// Update status
	now := time.Now()
	status := "unhealthy"
	if healthy {
		status = "healthy"
	}

	if storageErr := s.db.Model(&PrometheusDatasource{}).Where("id = ? AND tenant_id = ? AND revision = ?", record.ID, s.tenantID, record.Revision).Updates(map[string]interface{}{
		"status":       status,
		"last_checked": now,
	}).Error; storageErr != nil {
		return false, storageErr
	}
	if checkErr := s.recheckRecord(record); checkErr != nil {
		return false, checkErr
	}

	return healthy, err
}

func (s *DatasourceService) testConnectionAsync(id, revision uint64, scope *common.DataScopeReq) {
	var current PrometheusDatasource
	if err := s.tenantDB().Scopes(database.WithDataScope(scope)).First(&current, id).Error; err != nil || current.Revision != revision {
		return
	}
	_, _ = s.testConnectionRecord(&current)
}

func (s *DatasourceService) startConnectionTest(id, revision uint64, scope *common.DataScopeReq) {
	// Adopted CRUD has no independent read/test grant or live request after it
	// returns. The explicit protected test endpoint owns all private probes.
	if s.tenantBound {
		return
	}
	background := *s
	background.db = s.db.WithContext(context.Background())
	background.checkpoint = nil
	var immutable *common.DataScopeReq
	if scope != nil {
		copied := *scope
		copied.DeptIDs = append([]uint64(nil), scope.DeptIDs...)
		copied.RoleKeys = append([]string(nil), scope.RoleKeys...)
		immutable = &copied
	}
	go background.testConnectionAsync(id, revision, immutable)
}

func (s *DatasourceService) toResponse(r *PrometheusDatasource) *DatasourceResponse {
	return &DatasourceResponse{
		ID:       r.ID,
		TenantID: r.TenantID, Revision: r.Revision,
		DeptID:      r.DeptID,
		Name:        r.Name,
		Type:        r.Type,
		URL:         r.URL,
		AccessMode:  r.AccessMode,
		Username:    r.Username,
		Timeout:     r.Timeout,
		IsDefault:   r.IsDefault,
		Status:      r.Status,
		LastChecked: r.LastChecked,
		CreatedAt:   r.CreatedAt,
		CreatedBy:   r.CreatedBy,
	}
}

// Query proxies a PromQL instant query to the data source and returns the
// raw Prometheus response body.
func (s *DatasourceService) Query(id uint64, query string, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	return s.QueryContext(context.Background(), id, query, scopes...)
}

// QueryContext propagates stop/disconnect cancellation to Prometheus-compatible providers.
func (s *DatasourceService) QueryContext(ctx context.Context, id uint64, query string, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(query)) == 0 || len(query) > 4096 {
		return nil, errLogQueryInvalid
	}
	record, err := s.datasourceForTypeContext(ctx, id, firstDataScope(scopes), "prometheus", "victoriametrics")
	if err != nil {
		return nil, err
	}
	values := url.Values{"query": []string{query}}
	return s.proxyJSONContext(ctx, record, "/api/v1/query", values)
}

// QueryLogs proxies a Loki range query.
func (s *DatasourceService) QueryLogs(id uint64, query, start, end string, limit int, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	return s.QueryLogsContext(context.Background(), id, query, start, end, limit, scopes...)
}

// QueryLogsContext keeps disconnect/stop cancellation attached to the upstream query.
func (s *DatasourceService) QueryLogsContext(ctx context.Context, id uint64, query, start, end string, limit int, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(query) == 0 || len(query) > 4096 || !validLogQueryRange(start, end) {
		return nil, errLogQueryInvalid
	}
	record, err := s.datasourceForTypeContext(ctx, id, firstDataScope(scopes), "loki")
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	values := url.Values{"query": []string{query}, "limit": []string{strconv.Itoa(limit)}, "direction": []string{"backward"}}
	if start != "" {
		values.Set("start", start)
	}
	if end != "" {
		values.Set("end", end)
	}
	return s.proxyJSONContext(ctx, record, "/loki/api/v1/query_range", values)
}

var errLogQueryInvalid = errors.New("param.invalid")

// Explicit ranges accept the existing Loki RFC3339 or numeric timestamp contract.
// Both bounds are required together, ordered, and limited to one day.
func validLogQueryRange(start, end string) bool {
	if start == "" && end == "" {
		return true
	}
	parse := func(value string) (time.Time, error) {
		if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return parsed, nil
		}
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil || number <= 0 {
			return time.Time{}, errLogQueryInvalid
		}
		if number > 1e12 {
			return time.Unix(0, number), nil
		}
		return time.Unix(number, 0), nil
	}
	first, firstErr := parse(start)
	last, lastErr := parse(end)
	return firstErr == nil && lastErr == nil && last.After(first) && last.Sub(first) <= 24*time.Hour
}

// SearchTraces proxies a Tempo TraceQL search.
func (s *DatasourceService) SearchTraces(id uint64, query, start, end string, limit int, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	return s.SearchTracesContext(context.Background(), id, query, start, end, limit, scopes...)
}

// SearchTracesContext bounds Tempo query input and keeps request cancellation attached.
func (s *DatasourceService) SearchTracesContext(ctx context.Context, id uint64, query, start, end string, limit int, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(query)) == 0 || len(query) > 4096 || !validLogQueryRange(start, end) {
		return nil, errLogQueryInvalid
	}
	record, err := s.datasourceForTypeContext(ctx, id, firstDataScope(scopes), "tempo")
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	values := url.Values{"q": []string{query}, "limit": []string{strconv.Itoa(limit)}}
	// Tempo search requires Unix seconds, while the public query contract also
	// accepts explicit RFC3339 UTC ranges and Loki nanosecond timestamps.
	for key, value := range map[string]string{"start": start, "end": end} {
		if value == "" {
			continue
		}
		if instant, err := time.Parse(time.RFC3339Nano, value); err == nil {
			values.Set(key, strconv.FormatInt(instant.Unix(), 10))
		} else {
			seconds, _ := strconv.ParseInt(value, 10, 64) // validated above
			if seconds > 1e12 {
				seconds /= int64(time.Second)
			}
			values.Set(key, strconv.FormatInt(seconds, 10))
		}
	}
	return s.proxyJSONContext(ctx, record, "/api/search", values)
}

// GetTrace proxies one Tempo trace lookup.
func (s *DatasourceService) GetTrace(id uint64, traceID string, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	return s.GetTraceContext(context.Background(), id, traceID, scopes...)
}

// GetTraceContext cancels detail reads on source switches and closed dialogs.
func (s *DatasourceService) GetTraceContext(ctx context.Context, id uint64, traceID string, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	record, err := s.datasourceForTypeContext(ctx, id, firstDataScope(scopes), "tempo")
	if err != nil {
		return nil, err
	}
	traceID = strings.TrimSpace(traceID)
	if traceID == "" || len(traceID) > 128 || strings.ContainsAny(traceID, "/\\\r\n") {
		return nil, errors.New("observability.trace_id_invalid")
	}
	return s.proxyJSONContext(ctx, record, "/api/traces/"+url.PathEscape(traceID), nil)
}

func (s *DatasourceService) datasourceForType(id uint64, scope *common.DataScopeReq, allowed ...string) (*PrometheusDatasource, error) {
	return s.datasourceForTypeContext(context.Background(), id, scope, allowed...)
}

func (s *DatasourceService) datasourceForTypeContext(ctx context.Context, id uint64, scope *common.DataScopeReq, allowed ...string) (*PrometheusDatasource, error) {
	if s.tenantBound && scope == nil {
		return nil, ErrWorkspaceDenied
	}
	var record PrometheusDatasource
	if err := s.tenantDB().WithContext(ctx).Scopes(database.WithDataScope(scope)).First(&record, id).Error; err != nil {
		return nil, err
	}
	for _, datasourceType := range allowed {
		if record.Type == datasourceType {
			return &record, nil
		}
	}
	return nil, errors.New("observability.datasource_type_mismatch")
}

func firstDataScope(scopes []*common.DataScopeReq) *common.DataScopeReq {
	if len(scopes) == 0 {
		return nil
	}
	return scopes[0]
}

func ownedDeptID(scope *common.DataScopeReq) uint64 {
	if scope == nil || scope.IsAdmin {
		return 0
	}
	if scope.DeptID > 0 {
		return scope.DeptID
	}
	if len(scope.DeptIDs) > 0 {
		return scope.DeptIDs[0]
	}
	return 0
}

func (s *DatasourceService) proxyJSON(record *PrometheusDatasource, requestPath string, values url.Values) (map[string]interface{}, error) {
	return s.proxyJSONContext(context.Background(), record, requestPath, values)
}

func (s *DatasourceService) proxyJSONContext(parent context.Context, record *PrometheusDatasource, requestPath string, values url.Values) (map[string]interface{}, error) {
	if err := s.recheckRecord(record); err != nil {
		return nil, err
	}
	// SSRF Protection: Re-validate URL before proxying (defense in depth)
	if err := s.ssrfValidator.ValidateURL(record.URL); err != nil {
		return nil, fmt.Errorf("observability.datasource_invalid_url: %w", err)
	}

	password, err := secrets.Decrypt(record.Password)
	if err != nil {
		return nil, err
	}
	bearerToken, err := secrets.Decrypt(record.BearerToken)
	if err != nil {
		return nil, err
	}
	timeout := normalizeTimeout(record.Timeout)
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeout)*time.Second)
	defer cancel()

	baseURL, err := url.Parse(record.URL)
	if err != nil {
		return nil, err
	}
	baseURL.Path = path.Join(baseURL.Path, requestPath)
	method := http.MethodGet
	var bodyReader io.Reader
	if s.tenantBound && (record.Type == "prometheus" || record.Type == "victoriametrics" || record.Type == "loki") {
		method = http.MethodPost
		bodyReader = strings.NewReader(values.Encode())
		baseURL.RawQuery = ""
	} else {
		baseURL.RawQuery = values.Encode()
	}

	// SSRF Protection: Use validator's HTTP client with redirect protection
	client := s.ssrfValidator.CreateHTTPClient(30 * time.Second)
	client.Timeout = time.Duration(timeout) * time.Second

	req, err := http.NewRequestWithContext(ctx, method, baseURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	applyDatasourceAuth(req, record.Username, password, bearerToken)
	if err := s.recheckRecord(record); err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("observability.datasource_query_status_%d", resp.StatusCode)
	}
	// Bound provider response memory before decoding or returning it to the browser.
	const maxResponseBytes = 8 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("observability.datasource_response_too_large")
	}
	if err := s.recheckRecord(record); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if status, ok := result["status"].(string); ok && status != queryStatusSuccess {
		return nil, errors.New("observability.datasource_query_failed")
	}
	return result, nil
}

func normalizeTimeout(timeout int) int {
	if timeout <= 0 || timeout > 30 {
		return 30
	}
	return timeout
}

func applyDatasourceAuth(req *http.Request, username, password, bearerToken string) {
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	} else if username != "" {
		req.SetBasicAuth(username, password)
	}
}

func testDatasourceConnection(datasourceType, rawURL, username, password, bearerToken string, timeout int) (bool, error) {
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = []int{80, 443, 8080, 9090, 3000, 3100, 16686}
	return testDatasourceConnectionContext(context.Background(), validator, datasourceType, rawURL, username, password, bearerToken, timeout, nil)
}

func testDatasourceConnectionContext(parent context.Context, validator *ssrf.Validator, datasourceType, rawURL, username, password, bearerToken string, timeout int, checkpoint func() error) (bool, error) {
	if err := parent.Err(); err != nil {
		return false, err
	}
	if validator == nil {
		return false, ErrWorkspaceDenied
	}
	// SSRF Protection: Validate URL before testing connection
	if err := validator.ValidateURL(rawURL); err != nil {
		return false, fmt.Errorf("observability.datasource_invalid_url: %w", err)
	}

	timeout = normalizeTimeout(timeout)
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeout)*time.Second)
	defer cancel()

	testPath := "/api/v1/status/config"
	if datasourceType == "loki" || datasourceType == "tempo" {
		testPath = "/ready"
	}
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}
	baseURL.Path = path.Join(baseURL.Path, testPath)
	testURL := baseURL.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return false, err
	}

	applyDatasourceAuth(req, username, password, bearerToken)

	// SSRF Protection: Use validator's HTTP client with redirect protection
	client := validator.CreateHTTPClient(time.Duration(timeout) * time.Second)
	if checkpoint != nil {
		if err := checkpoint(); err != nil {
			return false, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("observability.datasource_test_status_%d", resp.StatusCode)
	}
	if datasourceType == "loki" || datasourceType == "tempo" {
		return true, nil
	}

	// Parse response to verify it's valid Prometheus
	body, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if err != nil || len(body) > 8<<20 {
		return false, errors.New("observability.datasource_response_too_large")
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	if status, ok := result["status"].(string); !ok || status != queryStatusSuccess {
		return false, errors.New("observability.datasource_test_invalid_response")
	}

	return true, nil
}

// WithTenantContext binds a copy for an opted-in request. Legacy internal callers
// of the unbound constructor remain explicitly tenant0, never all tenants.
func (s *DatasourceService) WithTenantContext(tc *tenant.Context) (*DatasourceService, error) {
	if s == nil || s.db == nil || tc == nil || (tc.Mode != tenant.ModeCompat && tc.Mode != tenant.ModeMulti) || (tc.Mode == tenant.ModeCompat && tc.TenantID != 0) {
		return nil, ErrWorkspaceDenied
	}
	copied := *s
	copied.tenantID = tc.TenantID
	copied.tenantBound = true
	return &copied, nil
}
func (s *DatasourceService) tenantDB() *gorm.DB { return s.db.Where("tenant_id = ?", s.tenantID) }

func (s *DatasourceService) recheckRecord(record *PrometheusDatasource) error {
	if err := s.db.Statement.Context.Err(); err != nil {
		return err
	}
	if record == nil || record.TenantID != s.tenantID {
		return ErrWorkspaceDenied
	}
	var current PrometheusDatasource
	if err := s.tenantDB().First(&current, record.ID).Error; err != nil || current.Revision != record.Revision || current.Type != record.Type {
		return ErrWorkspaceConflict
	}
	if s.checkpoint != nil {
		return s.checkpoint(record)
	}
	return nil
}
