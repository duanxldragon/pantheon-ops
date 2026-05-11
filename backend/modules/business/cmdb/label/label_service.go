package label

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	cmdbgroup "pantheon-ops/backend/modules/business/cmdb/group"
	cmdbhost "pantheon-ops/backend/modules/business/cmdb/host"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrLabelSchemaKeyExists = errors.New("cmdblabel.key_exists")
	ErrLabelSchemaNotFound  = errors.New("cmdblabel.not_found")
	ErrLabelSchemaInvalid   = errors.New("cmdblabel.invalid")
	ErrLabelSchemaInUse     = errors.New("cmdblabel.in_use")
)

type LabelService struct {
	db *gorm.DB
}

func NewLabelService(db *gorm.DB) *LabelService {
	return &LabelService{db: db}
}

func (s *LabelService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&LabelSchema{})
}

func (s *LabelService) List(query LabelSchemaQuery) ([]LabelSchemaResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	db := s.db.Model(&LabelSchema{})
	if strings.TrimSpace(query.Keyword) != "" {
		like := "%" + strings.TrimSpace(query.Keyword) + "%"
		db = db.Where("`key` LIKE ? OR name LIKE ?", like, like)
	}
	if strings.TrimSpace(query.Status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(query.Status))
	}
	var rows []LabelSchema
	if err := db.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	resp := make([]LabelSchemaResponse, len(rows))
	for i := range rows {
		resp[i] = toResponse(&rows[i])
	}
	return resp, nil
}

func (s *LabelService) Create(req CreateLabelSchemaRequest) (*LabelSchemaResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return nil, ErrLabelSchemaInvalid
	}
	if err := validateValueMode(req.ValueMode, req.DictCode); err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.Model(&LabelSchema{}).Where("`key` = ?", key).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrLabelSchemaKeyExists
	}
	status := normalizeStatus(req.Status)
	row := LabelSchema{
		Key:         key,
		Name:        strings.TrimSpace(req.Name),
		ValueMode:   normalizeValueMode(req.ValueMode),
		DictCode:    strings.TrimSpace(req.DictCode),
		Options:     marshalOptions(normalizeOptions(req.Options)),
		Required:    req.Required,
		Status:      status,
		Description: strings.TrimSpace(req.Description),
	}
	if row.Name == "" {
		row.Name = key
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	resp := toResponse(&row)
	return &resp, nil
}

func (s *LabelService) Update(id uint64, req UpdateLabelSchemaRequest) (*LabelSchemaResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var row LabelSchema
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLabelSchemaNotFound
		}
		return nil, err
	}
	nextMode := row.ValueMode
	nextDictCode := row.DictCode
	if req.ValueMode != nil {
		nextMode = *req.ValueMode
	}
	if req.DictCode != nil {
		nextDictCode = *req.DictCode
	}
	if err := validateValueMode(nextMode, nextDictCode); err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.ValueMode != nil {
		updates["value_mode"] = normalizeValueMode(*req.ValueMode)
	}
	if req.DictCode != nil {
		updates["dict_code"] = strings.TrimSpace(*req.DictCode)
	}
	if req.Options != nil {
		updates["options"] = marshalOptions(normalizeOptions(*req.Options))
	}
	if req.Required != nil {
		updates["required"] = *req.Required
	}
	if req.Status != nil {
		updates["status"] = normalizeStatus(*req.Status)
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if err := s.db.Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	resp := toResponse(&row)
	return &resp, nil
}

func (s *LabelService) Delete(id uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	var row LabelSchema
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLabelSchemaNotFound
		}
		return err
	}
	inUse, err := s.isLabelKeyInUse(row.Key)
	if err != nil {
		return err
	}
	if inUse {
		return ErrLabelSchemaInUse
	}
	return s.db.Delete(&row).Error
}

func (s *LabelService) isLabelKeyInUse(key string) (bool, error) {
	var hosts []cmdbhost.Host
	if err := s.db.Model(&cmdbhost.Host{}).Find(&hosts).Error; err != nil {
		return false, err
	}
	for _, h := range hosts {
		var labels []struct {
			Key string `json:"key"`
			Val string `json:"val"`
		}
		if len(h.LabelValues) > 0 && json.Unmarshal(h.LabelValues, &labels) == nil {
			for _, label := range labels {
				if label.Key == key {
					return true, nil
				}
			}
		}
	}
	var groups []cmdbgroup.Group
	if err := s.db.Model(&cmdbgroup.Group{}).Find(&groups).Error; err != nil {
		return false, err
	}
	for _, g := range groups {
		var expr struct {
			Rules []struct {
				Key string `json:"key"`
			} `json:"rules"`
		}
		if len(g.Conditions) > 0 && json.Unmarshal(g.Conditions, &expr) == nil {
			for _, rule := range expr.Rules {
				if rule.Key == key {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func validateValueMode(mode string, dictCode string) error {
	normalized := normalizeValueMode(mode)
	switch normalized {
	case "free", "enum", "dict":
	default:
		return ErrLabelSchemaInvalid
	}
	if normalized == "dict" && strings.TrimSpace(dictCode) == "" {
		return ErrLabelSchemaInvalid
	}
	return nil
}

func normalizeValueMode(mode string) string {
	normalized := strings.TrimSpace(mode)
	if normalized == "" {
		return "free"
	}
	return normalized
}

func normalizeStatus(status string) string {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return "enabled"
	}
	return normalized
}

func normalizeOptions(options []string) []string {
	result := make([]string, 0, len(options))
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		normalized := strings.TrimSpace(option)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func marshalOptions(options []string) datatypes.JSON {
	if len(options) == 0 {
		return datatypes.JSON([]byte("[]"))
	}
	payload, _ := json.Marshal(options)
	return datatypes.JSON(payload)
}

func unmarshalOptions(payload datatypes.JSON) []string {
	if len(payload) == 0 {
		return []string{}
	}
	var options []string
	if err := json.Unmarshal(payload, &options); err != nil {
		return []string{}
	}
	return normalizeOptions(options)
}

func toResponse(row *LabelSchema) LabelSchemaResponse {
	return LabelSchemaResponse{
		ID:          row.ID,
		Key:         row.Key,
		Name:        row.Name,
		ValueMode:   row.ValueMode,
		DictCode:    row.DictCode,
		Options:     unmarshalOptions(row.Options),
		Required:    row.Required,
		Status:      row.Status,
		Description: row.Description,
		CreatedAt:   row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   row.UpdatedAt.Format(time.RFC3339),
	}
}
