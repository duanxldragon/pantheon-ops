package generator

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func (s *GeneratorService) ListDatasources() ([]GeneratorDatasourceResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	schemaName, err := s.currentSchema()
	if err != nil {
		return nil, err
	}

	items := []GeneratorDatasourceResp{{
		ID:            generatorDatasourceCurrentID,
		Name:          "当前平台库",
		Driver:        "mysql",
		DatabaseName:  schemaName,
		Status:        generatorDatasourceEnabled,
		ReadonlyScope: "metadata_only",
		IsCurrent:     true,
	}}

	var rows []GeneratorDatasource
	if err := s.db.Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		items = append(items, buildDatasourceResp(row))
	}
	return items, nil
}

func (s *GeneratorService) CreateDatasource(req *UpsertGeneratorDatasourceReq) (*GeneratorDatasourceResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	row, err := normalizeDatasourceReq(req, true)
	if err != nil {
		return nil, err
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	resp := buildDatasourceResp(*row)
	return &resp, nil
}

func (s *GeneratorService) UpdateDatasource(id string, req *UpsertGeneratorDatasourceReq) (*GeneratorDatasourceResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	numericID, err := parseDatasourceNumericID(id)
	if err != nil {
		return nil, err
	}

	var existing GeneratorDatasource
	if err := s.db.First(&existing, numericID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("generator.datasource.not_found")
		}
		return nil, err
	}

	row, err := normalizeDatasourceReq(req, false)
	if err != nil {
		return nil, err
	}
	existing.Name = row.Name
	existing.Driver = row.Driver
	existing.Host = row.Host
	existing.Port = row.Port
	existing.DatabaseName = row.DatabaseName
	existing.Username = row.Username
	existing.Status = row.Status
	existing.Remark = row.Remark
	existing.ReadonlyScope = row.ReadonlyScope
	if strings.TrimSpace(row.PasswordEncrypted) != "" {
		existing.PasswordEncrypted = row.PasswordEncrypted
	}
	if err := s.db.Save(&existing).Error; err != nil {
		return nil, err
	}
	resp := buildDatasourceResp(existing)
	return &resp, nil
}

func (s *GeneratorService) DeleteDatasource(id string) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	numericID, err := parseDatasourceNumericID(id)
	if err != nil {
		return err
	}
	result := s.db.Delete(&GeneratorDatasource{}, numericID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("generator.datasource.not_found")
	}
	return nil
}

func (s *GeneratorService) TestDatasource(id string) (*GeneratorDatasourceResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if strings.TrimSpace(id) == "" || id == generatorDatasourceCurrentID {
		schemaName, err := s.currentSchema()
		if err != nil {
			return nil, err
		}
		now := time.Now().Format(time.RFC3339)
		return &GeneratorDatasourceResp{
			ID:              generatorDatasourceCurrentID,
			Name:            "当前平台库",
			Driver:          "mysql",
			DatabaseName:    schemaName,
			Status:          generatorDatasourceEnabled,
			ReadonlyScope:   "metadata_only",
			LastCheckedAt:   now,
			LastCheckStatus: "success",
			IsCurrent:       true,
		}, nil
	}

	row, err := s.loadDatasource(id)
	if err != nil {
		return nil, err
	}
	reader, err := s.openSchemaReader(id)
	now := time.Now()
	status := "success"
	lastError := ""
	if err != nil {
		status = "failed"
		lastError = trimErrorMessage(err.Error())
	} else if reader.close != nil {
		_ = reader.close()
	}

	if saveErr := s.db.Model(row).Updates(map[string]interface{}{
		"last_checked_at":   &now,
		"last_check_status": status,
		"last_check_error":  lastError,
	}).Error; saveErr != nil {
		return nil, saveErr
	}
	if err != nil {
		return nil, err
	}
	row.LastCheckedAt = &now
	row.LastCheckStatus = status
	row.LastCheckError = lastError
	resp := buildDatasourceResp(*row)
	return &resp, nil
}

func (s *GeneratorService) openSchemaReader(datasourceID string) (*generatorSchemaReader, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if strings.TrimSpace(datasourceID) == "" || datasourceID == generatorDatasourceCurrentID {
		schemaName, err := s.currentSchema()
		if err != nil {
			return nil, err
		}
		return &generatorSchemaReader{db: s.db, schema: schemaName}, nil
	}

	row, err := s.loadDatasource(datasourceID)
	if err != nil {
		return nil, err
	}
	if row.Status != generatorDatasourceEnabled {
		return nil, errors.New("generator.datasource.disabled")
	}
	if strings.TrimSpace(row.Driver) != "" && !strings.EqualFold(strings.TrimSpace(row.Driver), "mysql") {
		return nil, errors.New("generator.datasource.driver_unsupported")
	}
	password, err := decryptDatasourcePassword(row.PasswordEncrypted)
	if err != nil {
		return nil, err
	}

	cfg := mysqlDriver.NewConfig()
	cfg.User = row.Username
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", row.Host, row.Port)
	cfg.DBName = row.DatabaseName
	cfg.ParseTime = true
	cfg.Collation = "utf8mb4_general_ci"
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	cfg.Params = map[string]string{"charset": "utf8mb4"}

	db, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		return nil, errors.New("generator.datasource.connect_failed")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(0)

	return &generatorSchemaReader{
		db:     db,
		schema: row.DatabaseName,
		close:  sqlDB.Close,
	}, nil
}

func (s *GeneratorService) loadDatasource(id string) (*GeneratorDatasource, error) {
	numericID, err := parseDatasourceNumericID(id)
	if err != nil {
		return nil, err
	}
	var row GeneratorDatasource
	if err := s.db.First(&row, numericID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("generator.datasource.not_found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *GeneratorService) currentSchema() (string, error) {
	var schemaName string
	if err := s.db.Raw("select database()").Scan(&schemaName).Error; err != nil {
		return "", err
	}
	if strings.TrimSpace(schemaName) == "" {
		return "", errors.New("database.schema_unknown")
	}
	return schemaName, nil
}

func normalizeDatasourceReq(req *UpsertGeneratorDatasourceReq, requirePassword bool) (*GeneratorDatasource, error) {
	if req == nil {
		return nil, errors.New("param.invalid")
	}
	name := strings.TrimSpace(req.Name)
	host := strings.TrimSpace(req.Host)
	databaseName := strings.TrimSpace(req.DatabaseName)
	username := strings.TrimSpace(req.Username)
	driver := strings.ToLower(strings.TrimSpace(req.Driver))
	if driver == "" {
		driver = "mysql"
	}
	if name == "" || host == "" || databaseName == "" || username == "" {
		return nil, errors.New("generator.datasource.required")
	}
	if driver != "mysql" {
		return nil, errors.New("generator.datasource.driver_unsupported")
	}
	if err := validateDatasourceHost(host); err != nil {
		return nil, err
	}
	port := req.Port
	if port <= 0 {
		port = 3306
	}
	if port > 65535 {
		return nil, errors.New("generator.datasource.port_invalid")
	}
	if req.Status != generatorDatasourceEnabled && req.Status != generatorDatasourceDisabled {
		req.Status = generatorDatasourceEnabled
	}
	password := strings.TrimSpace(req.Password)
	if requirePassword && password == "" {
		return nil, errors.New("generator.datasource.password_required")
	}
	encrypted := ""
	var err error
	if password != "" {
		encrypted, err = encryptDatasourcePassword(password)
		if err != nil {
			return nil, err
		}
	}

	return &GeneratorDatasource{
		Name:              name,
		Driver:            driver,
		Host:              host,
		Port:              port,
		DatabaseName:      databaseName,
		Username:          username,
		PasswordEncrypted: encrypted,
		Status:            req.Status,
		ReadonlyScope:     "metadata_only",
		Remark:            strings.TrimSpace(req.Remark),
	}, nil
}

func validateDatasourceHost(host string) error {
	normalizedHost := strings.ToLower(strings.TrimSpace(host))
	if normalizedHost == "" {
		return errors.New("generator.datasource.required")
	}
	if strings.ContainsAny(normalizedHost, `/\:@`) {
		return errors.New("generator.datasource.host_invalid")
	}

	if addr, err := netip.ParseAddr(normalizedHost); err == nil {
		if addr.IsLoopback() || addr.IsMulticast() || addr.IsLinkLocalMulticast() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() {
			return errors.New("generator.datasource.host_invalid")
		}
		if addr.IsPrivate() && !allowPrivateGeneratorDatasourceHosts() {
			return errors.New("generator.datasource.host_private_disabled")
		}
		return nil
	}

	if normalizedHost == "localhost" || strings.HasSuffix(normalizedHost, ".localhost") {
		return errors.New("generator.datasource.host_invalid")
	}
	if strings.HasSuffix(normalizedHost, ".local") || strings.HasSuffix(normalizedHost, ".internal") {
		if !allowPrivateGeneratorDatasourceHosts() {
			return errors.New("generator.datasource.host_private_disabled")
		}
	}
	if !regexp.MustCompile(`^[a-z0-9.-]+$`).MatchString(normalizedHost) {
		return errors.New("generator.datasource.host_invalid")
	}
	if strings.HasPrefix(normalizedHost, ".") || strings.HasSuffix(normalizedHost, ".") || strings.Contains(normalizedHost, "..") {
		return errors.New("generator.datasource.host_invalid")
	}
	for _, label := range strings.Split(normalizedHost, ".") {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return errors.New("generator.datasource.host_invalid")
		}
	}
	return nil
}

func allowPrivateGeneratorDatasourceHosts() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("PANTHEON_ALLOW_PRIVATE_GENERATOR_DATASOURCE")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func parseDatasourceNumericID(id string) (uint64, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" || trimmed == generatorDatasourceCurrentID {
		return 0, errors.New("generator.datasource.not_found")
	}
	value, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || value == 0 {
		return 0, errors.New("generator.datasource.not_found")
	}
	return value, nil
}

func buildDatasourceResp(row GeneratorDatasource) GeneratorDatasourceResp {
	resp := GeneratorDatasourceResp{
		ID:              strconv.FormatUint(row.ID, 10),
		Name:            row.Name,
		Driver:          row.Driver,
		Host:            row.Host,
		Port:            row.Port,
		DatabaseName:    row.DatabaseName,
		Username:        row.Username,
		Status:          row.Status,
		Remark:          row.Remark,
		ReadonlyScope:   row.ReadonlyScope,
		LastCheckStatus: row.LastCheckStatus,
		LastCheckError:  row.LastCheckError,
		IsCurrent:       false,
	}
	if row.LastCheckedAt != nil {
		resp.LastCheckedAt = row.LastCheckedAt.Format(time.RFC3339)
	}
	return resp
}

func trimErrorMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if len(trimmed) <= 255 {
		return trimmed
	}
	return trimmed[:255]
}
