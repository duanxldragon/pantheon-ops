package system

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pantheon-ops/backend/pkg/common"
)

type SystemRefreshVersion struct {
	Topic     string    `gorm:"primaryKey;size:64"`
	Version   int64     `gorm:"not null;default:0"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (SystemRefreshVersion) TableName() string {
	return "system_refresh_version"
}

type RefreshStateResp struct {
	Topics map[string]int64 `json:"topics"`
}

type RefreshStateQuery struct {
	Topics string `form:"topics" json:"topics"`
}

type RefreshSyncService struct {
	db *gorm.DB
}

func NewRefreshSyncService(db *gorm.DB) *RefreshSyncService {
	return &RefreshSyncService{db: db}
}

func (s *RefreshSyncService) Migrate() error {
	if s.db == nil {
		return nil
	}
	return s.db.AutoMigrate(&SystemRefreshVersion{})
}

func (s *RefreshSyncService) Touch(topics []string) error {
	if s.db == nil {
		return nil
	}
	now := time.Now()
	for _, topic := range normalizeRefreshTopics(topics) {
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "topic"}},
			DoUpdates: clause.Assignments(map[string]any{"version": gorm.Expr("version + 1"), "updated_at": now}),
		}).Create(&SystemRefreshVersion{
			Topic:     topic,
			Version:   1,
			UpdatedAt: now,
			CreatedAt: now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *RefreshSyncService) GetState(topics []string) (*RefreshStateResp, error) {
	resp := &RefreshStateResp{Topics: map[string]int64{}}
	if s.db == nil {
		return resp, nil
	}
	normalized := normalizeRefreshTopics(topics)
	var rows []SystemRefreshVersion
	db := s.db.Model(&SystemRefreshVersion{})
	if len(normalized) > 0 {
		db = db.Where("topic IN ?", normalized)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, topic := range normalized {
		resp.Topics[topic] = 0
	}
	for _, row := range rows {
		resp.Topics[row.Topic] = row.Version
	}
	return resp, nil
}

type RefreshSyncHandler struct {
	service *RefreshSyncService
}

func NewRefreshSyncHandler(service *RefreshSyncService) *RefreshSyncHandler {
	return &RefreshSyncHandler{service: service}
}

func (h *RefreshSyncHandler) GetState(c *gin.Context) {
	var query RefreshStateQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.GetState(parseRefreshTopicQuery(query.Topics))
	if err != nil {
		common.Fail(c, common.CodeError, "refresh.state.error")
		return
	}
	common.Success(c, resp)
}

func RefreshSyncMiddleware(service *RefreshSyncService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		c.Next()

		if c.Writer.Status() >= http.StatusBadRequest {
			return
		}
		if !isRefreshSyncSuccess(c) {
			return
		}

		topics := detectRefreshTopics(c.Request.URL.Path)
		if len(topics) == 0 {
			return
		}
		_ = service.Touch(topics)
	}
}

func isRefreshSyncSuccess(c *gin.Context) bool {
	return c.Writer.Status() < http.StatusBadRequest
}

func detectRefreshTopics(path string) []string {
	normalized := strings.TrimSpace(path)
	if shouldSkipRefreshSyncPath(normalized) {
		return nil
	}
	switch {
	case strings.Contains(normalized, "/system/user"):
		return []string{"system:user:changed"}
	case strings.Contains(normalized, "/system/role"):
		return []string{"system:role:changed"}
	case strings.Contains(normalized, "/system/menu"):
		return []string{"system:menu:changed"}
	case strings.Contains(normalized, "/system/dept"):
		return []string{"system:dept:changed"}
	case strings.Contains(normalized, "/system/post"):
		return []string{"system:post:changed"}
	case strings.Contains(normalized, "/system/permission"):
		return []string{"system:permission:changed"}
	case strings.Contains(normalized, "/system/dict"):
		return []string{"system:dict:changed"}
	case strings.Contains(normalized, "/system/setting"), strings.Contains(normalized, "/system/upload"):
		return []string{"system:setting:changed"}
	case strings.Contains(normalized, "/system/i18n"):
		return []string{"system:i18n:changed"}
	default:
		return nil
	}
}

func shouldSkipRefreshSyncPath(path string) bool {
	switch {
	case strings.Contains(path, "/export"),
		strings.Contains(path, "/import-template"),
		strings.Contains(path, "/rename/preview"),
		strings.Contains(path, "/audit/list"),
		strings.Contains(path, "/audit/export"):
		return true
	default:
		return false
	}
}

func parseRefreshTopicQuery(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func normalizeRefreshTopics(topics []string) []string {
	if len(topics) == 0 {
		return nil
	}
	uniq := make(map[string]struct{}, len(topics))
	for _, topic := range topics {
		trimmed := strings.TrimSpace(topic)
		if trimmed == "" {
			continue
		}
		uniq[trimmed] = struct{}{}
	}
	result := make([]string, 0, len(uniq))
	for topic := range uniq {
		result = append(result, topic)
	}
	sort.Strings(result)
	return result
}
