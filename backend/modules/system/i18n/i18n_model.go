package system

import (
	"time"
)

const I18nStalePlaceholderThresholdDays = 30
const I18nUnusedObservationThresholdDays = 14

const (
	I18nLifecycleStatusActive    = "active"
	I18nLifecycleStatusObserving = "observing"
	I18nLifecycleStatusArchived  = "archived"
)

// SystemI18n 国际化翻译模型
type SystemI18n struct {
	ID                uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Module            string     `gorm:"size:64;not null;index:idx_system_i18n_module_key" json:"module"`
	Group             string     `gorm:"column:group_name;size:64;not null;default:'messages';index:idx_system_i18n_module_group" json:"group"`
	Key               string     `gorm:"size:128;not null;index:idx_system_i18n_module_key" json:"key"`
	Locale            string     `gorm:"size:10;not null;index:idx_system_i18n_locale" json:"locale"`
	Value             string     `gorm:"type:text;not null" json:"value"`
	Remark            string     `gorm:"size:255" json:"remark"`
	LifecycleStatus   string     `gorm:"size:16;not null;default:'active';index:idx_system_i18n_lifecycle" json:"lifecycleStatus"`
	LifecycleMarkedAt *time.Time `json:"lifecycleMarkedAt"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func (SystemI18n) TableName() string {
	return "system_i18n"
}

// I18nQuery 搜索参数
type I18nQuery struct {
	Module    string `json:"module" form:"module"`
	Group     string `json:"group" form:"group"`
	Locale    string `json:"locale" form:"locale"`
	Key       string `json:"key" form:"key"`
	SortBy    string `json:"sortBy" form:"sortBy"`
	SortOrder string `json:"sortOrder" form:"sortOrder"`
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"pageSize" form:"pageSize"`
}

// I18nPageResp 分页返回
type I18nPageResp struct {
	Items    []SystemI18n `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// I18nUpdateReq 更新请求
type I18nUpdateReq struct {
	Value  string `json:"value" binding:"required"`
	Remark string `json:"remark"`
}

type I18nCreateReq struct {
	Module string `json:"module" binding:"required"`
	Group  string `json:"group"`
	Key    string `json:"key" binding:"required"`
	Locale string `json:"locale" binding:"required"`
	Value  string `json:"value" binding:"required"`
	Remark string `json:"remark"`
}

type I18nBatchDeleteReq struct {
	IDs []uint64 `json:"ids" binding:"required"`
}

type I18nSyncResp struct {
	Count int      `json:"count"`
	Keys  []string `json:"keys"`
}

type I18nCacheRefreshReq struct {
	Locales []string `json:"locales"`
}

type I18nLocaleCoverage struct {
	Locale       string `json:"locale"`
	EntryCount   int64  `json:"entryCount"`
	MissingCount int64  `json:"missingCount"`
}

type I18nOverviewResp struct {
	Locales            []string             `json:"locales"`
	ModuleCount        int64                `json:"moduleCount"`
	GroupCount         int64                `json:"groupCount"`
	TotalEntries       int64                `json:"totalEntries"`
	MissingValueCount  int64                `json:"missingValueCount"`
	MissingLocaleCount int64                `json:"missingLocaleCount"`
	Coverage           []I18nLocaleCoverage `json:"coverage"`
}

type I18nMissingLocaleItem struct {
	Module         string   `json:"module"`
	Group          string   `json:"group"`
	Key            string   `json:"key"`
	MissingLocales []string `json:"missingLocales"`
}

type I18nMissingLocaleResp struct {
	Items []I18nMissingLocaleItem `json:"items"`
	Total int64                   `json:"total"`
}

type I18nFillMissingLocaleResp struct {
	Created int64    `json:"created"`
	Locales []string `json:"locales"`
	Keys    []string `json:"keys"`
}

type I18nHydrateBuiltinResp struct {
	Created int64    `json:"created"`
	Updated int64    `json:"updated"`
	Locales []string `json:"locales"`
	Keys    []string `json:"keys"`
}

type I18nDuplicateKeyConflict struct {
	Key         string                 `json:"key"`
	Modules     []string               `json:"modules"`
	Groups      []string               `json:"groups"`
	Locales     []string               `json:"locales"`
	Values      []string               `json:"values"`
	RowCount    int64                  `json:"rowCount"`
	Suggestions []I18nRenameSuggestion `json:"suggestions"`
}

type I18nUnusedKeyItem struct {
	Key                string   `json:"key"`
	Module             string   `json:"module"`
	Modules            []string `json:"modules"`
	Groups             []string `json:"groups"`
	Locales            []string `json:"locales"`
	Placeholder        bool     `json:"placeholder"`
	LifecycleStatus    string   `json:"lifecycleStatus"`
	LifecycleMarkedAt  string   `json:"lifecycleMarkedAt,omitempty"`
	ObservingDays      int64    `json:"observingDays"`
	EligibleForArchive bool     `json:"eligibleForArchive"`
	EligibleForDelete  bool     `json:"eligibleForDelete"`
}

type I18nModuleAuditItem struct {
	Module                string `json:"module"`
	EntryCount            int64  `json:"entryCount"`
	KeyCount              int64  `json:"keyCount"`
	UnusedKeyCount        int64  `json:"unusedKeyCount"`
	DuplicateKeyCount     int64  `json:"duplicateKeyCount"`
	MissingLocaleCount    int64  `json:"missingLocaleCount"`
	PlaceholderCount      int64  `json:"placeholderCount"`
	StalePlaceholderCount int64  `json:"stalePlaceholderCount"`
	ObservingKeyCount     int64  `json:"observingKeyCount"`
	ArchivedKeyCount      int64  `json:"archivedKeyCount"`
}

type I18nRenameSuggestion struct {
	Module       string `json:"module"`
	SuggestedKey string `json:"suggestedKey"`
}

type I18nStalePlaceholderItem struct {
	ID        uint64 `json:"id"`
	Module    string `json:"module"`
	Group     string `json:"group"`
	Key       string `json:"key"`
	Locale    string `json:"locale"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updatedAt"`
	StaleDays int64  `json:"staleDays"`
}

type I18nAuditResp struct {
	DuplicateKeys                  []I18nDuplicateKeyConflict `json:"duplicateKeys"`
	UnusedKeys                     []I18nUnusedKeyItem        `json:"unusedKeys"`
	StalePlaceholders              []I18nStalePlaceholderItem `json:"stalePlaceholders"`
	Modules                        []I18nModuleAuditItem      `json:"modules"`
	StalePlaceholderThresholdDays  int64                      `json:"stalePlaceholderThresholdDays"`
	UnusedObservationThresholdDays int64                      `json:"unusedObservationThresholdDays"`
}

type I18nCleanupUnusedResp struct {
	Deleted int64    `json:"deleted"`
	Keys    []string `json:"keys"`
	Module  string   `json:"module"`
}

type I18nRenamePreviewReq struct {
	Module string `json:"module" binding:"required"`
	OldKey string `json:"oldKey" binding:"required"`
	NewKey string `json:"newKey" binding:"required"`
}

type I18nKeyReferenceFile struct {
	Path                 string                  `json:"path"`
	MatchCount           int                     `json:"matchCount"`
	SuggestedReplacement string                  `json:"suggestedReplacement"`
	Matches              []I18nKeyReferenceMatch `json:"matches"`
}

type I18nKeyReferenceMatch struct {
	Line            int    `json:"line"`
	Column          int    `json:"column"`
	Snippet         string `json:"snippet"`
	ReplacementHint string `json:"replacementHint"`
}

type I18nRenamePreviewResp struct {
	Module                string                 `json:"module"`
	OldKey                string                 `json:"oldKey"`
	NewKey                string                 `json:"newKey"`
	AffectedRows          int64                  `json:"affectedRows"`
	AffectedLocales       []string               `json:"affectedLocales"`
	ExistingTargetRows    int64                  `json:"existingTargetRows"`
	ExistingTargetLocales []string               `json:"existingTargetLocales"`
	ReferenceFiles        []I18nKeyReferenceFile `json:"referenceFiles"`
	RequiresCodeMigration bool                   `json:"requiresCodeMigration"`
	CanExecute            bool                   `json:"canExecute"`
}

type I18nRenameExecuteReq struct {
	Module               string `json:"module" binding:"required"`
	OldKey               string `json:"oldKey" binding:"required"`
	NewKey               string `json:"newKey" binding:"required"`
	ConfirmSourceUpdated bool   `json:"confirmSourceUpdated"`
}

type I18nRenameExecuteResp struct {
	Module         string   `json:"module"`
	OldKey         string   `json:"oldKey"`
	NewKey         string   `json:"newKey"`
	RenamedRows    int64    `json:"renamedRows"`
	RenamedLocales []string `json:"renamedLocales"`
}

type I18nUnusedLifecycleReq struct {
	Module          string `json:"module"`
	ConfirmArchived bool   `json:"confirmArchived"`
}

type I18nUnusedLifecycleResp struct {
	Module       string   `json:"module"`
	AffectedKeys []string `json:"affectedKeys"`
	AffectedRows int64    `json:"affectedRows"`
}
