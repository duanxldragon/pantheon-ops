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

	"gorm.io/gorm"
)

// DatasourceService manages metric, log, and trace data sources.
type DatasourceService struct {
	db            *gorm.DB
	ssrfValidator *ssrf.Validator
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
		if dto.IsDefault {
			if err := tx.Model(&PrometheusDatasource{}).Scopes(database.WithDataScope(scope)).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(record).Error
	}); err != nil {
		return nil, err
	}

	// Test connection in background
	go s.testConnectionAsync(record.ID, scope)

	return s.toResponse(record), nil
}

// List returns all observability data sources.
func (s *DatasourceService) List(scopes ...*common.DataScopeReq) (*DatasourceListResponse, error) {
	var records []PrometheusDatasource
	if err := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).Order("is_default DESC, created_at DESC").Find(&records).Error; err != nil {
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
	var record PrometheusDatasource
	if err := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toResponse(&record), nil
}

// Update modifies an existing data source.
func (s *DatasourceService) Update(id uint64, dto DatasourceDTO, scopes ...*common.DataScopeReq) error {
	scope := firstDataScope(scopes)

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
		var record PrometheusDatasource
		if err := tx.Scopes(database.WithDataScope(scope)).First(&record, id).Error; err != nil {
			return err
		}
		if dto.IsDefault {
			if err := tx.Model(&PrometheusDatasource{}).Scopes(database.WithDataScope(scope)).Where("is_default = ? AND id != ?", true, id).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Model(&record).Updates(updates).Error
	}); err != nil {
		return err
	}

	// Test connection in background
	go s.testConnectionAsync(id, scope)

	return nil
}

// Delete removes a data source.
func (s *DatasourceService) Delete(id uint64, scopes ...*common.DataScopeReq) error {
	scope := firstDataScope(scopes)
	var record PrometheusDatasource
	if err := s.db.Scopes(database.WithDataScope(scope)).First(&record, id).Error; err != nil {
		return err
	}
	var references int64
	if err := s.db.Model(&AlertRule{}).Scopes(database.WithDataScope(scope)).Where("datasource_id = ?", id).Count(&references).Error; err != nil {
		return err
	}
	if references > 0 {
		return errors.New("observability.datasource_in_use")
	}
	return s.db.Delete(&record).Error
}

// TestConnection tests connectivity to an observability data source.
func (s *DatasourceService) TestConnection(id uint64, scopes ...*common.DataScopeReq) (bool, error) {
	var record PrometheusDatasource
	if err := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).First(&record, id).Error; err != nil {
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
	healthy, err := testDatasourceConnection(record.Type, record.URL, record.Username, password, bearerToken, record.Timeout)

	// Update status
	now := time.Now()
	status := "unhealthy"
	if healthy {
		status = "healthy"
	}

	_ = s.db.Model(&record).Updates(map[string]interface{}{
		"status":       status,
		"last_checked": now,
	}).Error

	return healthy, err
}

func (s *DatasourceService) testConnectionAsync(id uint64, scope *common.DataScopeReq) {
	_, _ = s.TestConnection(id, scope)
}

func (s *DatasourceService) toResponse(r *PrometheusDatasource) *DatasourceResponse {
	return &DatasourceResponse{
		ID:          r.ID,
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
	record, err := s.datasourceForType(id, firstDataScope(scopes), "prometheus", "victoriametrics")
	if err != nil {
		return nil, err
	}
	values := url.Values{"query": []string{query}}
	return s.proxyJSON(record, "/api/v1/query", values)
}

// QueryLogs proxies a Loki range query.
func (s *DatasourceService) QueryLogs(id uint64, query, start, end string, limit int, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	record, err := s.datasourceForType(id, firstDataScope(scopes), "loki")
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
	return s.proxyJSON(record, "/loki/api/v1/query_range", values)
}

// SearchTraces proxies a Tempo TraceQL search.
func (s *DatasourceService) SearchTraces(id uint64, query, start, end string, limit int, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	record, err := s.datasourceForType(id, firstDataScope(scopes), "tempo")
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	values := url.Values{"q": []string{query}, "limit": []string{strconv.Itoa(limit)}}
	if start != "" {
		values.Set("start", start)
	}
	if end != "" {
		values.Set("end", end)
	}
	return s.proxyJSON(record, "/api/search", values)
}

// GetTrace proxies one Tempo trace lookup.
func (s *DatasourceService) GetTrace(id uint64, traceID string, scopes ...*common.DataScopeReq) (map[string]interface{}, error) {
	record, err := s.datasourceForType(id, firstDataScope(scopes), "tempo")
	if err != nil {
		return nil, err
	}
	traceID = strings.TrimSpace(traceID)
	if traceID == "" || strings.ContainsAny(traceID, "/\\") {
		return nil, errors.New("observability.trace_id_invalid")
	}
	return s.proxyJSON(record, "/api/traces/"+url.PathEscape(traceID), nil)
}

func (s *DatasourceService) datasourceForType(id uint64, scope *common.DataScopeReq, allowed ...string) (*PrometheusDatasource, error) {
	var record PrometheusDatasource
	if err := s.db.Scopes(database.WithDataScope(scope)).First(&record, id).Error; err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	baseURL, err := url.Parse(record.URL)
	if err != nil {
		return nil, err
	}
	baseURL.Path = path.Join(baseURL.Path, requestPath)
	baseURL.RawQuery = values.Encode()

	// SSRF Protection: Use validator's HTTP client with redirect protection
	client, err := s.ssrfValidator.CreateHTTPClient()
	if err != nil {
		return nil, err
	}
	client.Timeout = time.Duration(timeout) * time.Second

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return nil, err
	}
	applyDatasourceAuth(req, record.Username, password, bearerToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("observability.datasource_query_status_%d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if status, ok := result["status"].(string); ok && status != "success" {
		return nil, errors.New("observability.datasource_query_failed")
	}
	return result, nil
}

func normalizeTimeout(timeout int) int {
	if timeout <= 0 || timeout > 120 {
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
	// SSRF Protection: Validate URL before testing connection
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = []int{80, 443, 8080, 9090, 3000, 3100, 16686}
	if err := validator.ValidateURL(rawURL); err != nil {
		return false, fmt.Errorf("observability.datasource_invalid_url: %w", err)
	}

	timeout = normalizeTimeout(timeout)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
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
	client, err := validator.CreateHTTPClient()
	if err != nil {
		return false, err
	}
	client.Timeout = time.Duration(timeout) * time.Second

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, fmt.Errorf("observability.datasource_test_status_%d", resp.StatusCode)
	}
	if datasourceType == "loki" || datasourceType == "tempo" {
		return true, nil
	}

	// Parse response to verify it's valid Prometheus
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if status, ok := result["status"].(string); !ok || status != "success" {
		return false, errors.New("observability.datasource_test_invalid_response")
	}

	return true, nil
}
