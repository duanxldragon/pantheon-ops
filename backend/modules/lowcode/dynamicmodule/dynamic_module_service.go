package dynamicmodule

import (
	"errors"
	"pantheon-ops/backend/pkg/common"
	"strings"
	"time"

	"pantheon-ops/backend/internal/scaffold"
	"pantheon-ops/backend/pkg/contracts"

	"gorm.io/gorm"
)

const (
	ModuleStatusActive            = 1
	ModuleStatusUninstalled       = 2
	ModuleStatusPendingActivation = 3
	ModuleStatusFailed            = 4
)

// DynamicModuleService 动态模块管理服务
type DynamicModuleService struct {
	db            *gorm.DB
	workspaceRoot string
}

// NewDynamicModuleService 创建服务实例
func NewDynamicModuleService(db *gorm.DB) *DynamicModuleService {
	workspaceRoot, _ := scaffold.ResolveWorkspaceRoot("")
	return &DynamicModuleService{db: db, workspaceRoot: workspaceRoot}
}

// ModuleRegistration 模块注册信息
type ModuleRegistration struct {
	ID                     uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                   string `gorm:"size:64;uniqueIndex" json:"name"`
	DisplayName            string `gorm:"size:128" json:"displayName"`
	Scope                  string `gorm:"size:32" json:"scope"`
	Source                 string `gorm:"size:32" json:"source"`
	Owner                  string `gorm:"size:128" json:"owner"`
	BoundedContext         string `gorm:"size:128;column:bounded_context" json:"boundedContext"`
	Summary                string `gorm:"size:255" json:"summary"`
	SourceTable            string `gorm:"size:128;column:source_table" json:"sourceTable"`
	AutoRecycle            bool   `gorm:"column:auto_recycle;default:false" json:"autoRecycle"`
	ModelTableName         string `gorm:"size:128;column:table_name" json:"tableName"`
	Status                 int    `gorm:"default:1" json:"status"` // 1:已激活, 2:已卸载, 3:待激活, 4:失败
	InstalledAt            string `json:"installedAt"`
	UninstalledAt          string `json:"uninstalledAt,omitempty"`
	LastVerifiedAt         string `gorm:"size:64;column:last_verified_at" json:"lastVerifiedAt,omitempty"`
	LastError              string `gorm:"size:512;column:last_error" json:"lastError,omitempty"`
	LastVerificationResult string `gorm:"type:text;column:last_verification_result" json:"lastVerificationResult,omitempty"`
	BuiltIn                bool   `gorm:"-" json:"builtIn"`
}

type ModuleRegistrationResp struct {
	ID                     uint64 `json:"id"`
	Name                   string `json:"name"`
	DisplayName            string `json:"displayName"`
	Scope                  string `json:"scope"`
	Source                 string `json:"source"`
	Owner                  string `json:"owner"`
	BoundedContext         string `json:"boundedContext"`
	Summary                string `json:"summary"`
	SourceTable            string `json:"sourceTable"`
	AutoRecycle            bool   `json:"autoRecycle"`
	ModelTableName         string `json:"tableName"`
	Status                 int    `json:"status"`
	InstalledAt            string `json:"installedAt"`
	UninstalledAt          string `json:"uninstalledAt,omitempty"`
	LastVerifiedAt         string `json:"lastVerifiedAt,omitempty"`
	LastError              string `json:"lastError,omitempty"`
	LastVerificationResult string `json:"lastVerificationResult,omitempty"`
	BuiltIn                bool   `json:"builtIn"`
}

func toModuleRegistrationResp(module ModuleRegistration) ModuleRegistrationResp {
	return ModuleRegistrationResp{
		ID:                     module.ID,
		Name:                   module.Name,
		DisplayName:            module.DisplayName,
		Scope:                  module.Scope,
		Source:                 module.Source,
		Owner:                  module.Owner,
		BoundedContext:         module.BoundedContext,
		Summary:                module.Summary,
		SourceTable:            module.SourceTable,
		AutoRecycle:            module.AutoRecycle,
		ModelTableName:         module.ModelTableName,
		Status:                 module.Status,
		InstalledAt:            module.InstalledAt,
		UninstalledAt:          module.UninstalledAt,
		LastVerifiedAt:         module.LastVerifiedAt,
		LastError:              module.LastError,
		LastVerificationResult: module.LastVerificationResult,
		BuiltIn:                module.BuiltIn,
	}
}

type GeneratedModuleVerification struct {
	Code       string `json:"code"`
	Status     string `json:"status"`
	MessageKey string `json:"messageKey"`
	Detail     string `json:"detail"`
}

type GeneratedModuleRegistrationSummary struct {
	ModuleKey             string                         `json:"moduleKey"`
	RoutePath             string                         `json:"routePath"`
	RouteName             string                         `json:"routeName"`
	ComponentKey          string                         `json:"componentKey"`
	PermissionPrefix      string                         `json:"permissionPrefix"`
	Contract              GeneratedModuleContractSummary `json:"contract"`
	ParentMenuPath        string                         `json:"parentMenuPath"`
	ParentMenuSource      string                         `json:"parentMenuSource"`
	ParentMenuExists      bool                           `json:"parentMenuExists"`
	BackendModulePath     string                         `json:"backendModulePath"`
	FrontendModulePath    string                         `json:"frontendModulePath"`
	SchemaPath            string                         `json:"schemaPath"`
	RequiresRestart       bool                           `json:"requiresRestart"`
	RequiresFrontendBuild bool                           `json:"requiresFrontendBuild"`
	Verifications         []GeneratedModuleVerification  `json:"verifications"`
}

type GeneratedModuleContractSummary struct {
	TemplateVersion  string                      `json:"templateVersion"`
	DataScopeEnabled bool                        `json:"dataScopeEnabled"`
	DataScopeMode    string                      `json:"dataScopeMode"`
	DependencyCount  int                         `json:"dependencyCount"`
	RelationCount    int                         `json:"relationCount"`
	Dependencies     []scaffold.ModuleDependency `json:"dependencies,omitempty"`
	Relations        []scaffold.ModuleRelation   `json:"relations,omitempty"`
}

type RegistryRepairSummary struct {
	CheckedModules            int `json:"checkedModules"`
	GeneratedRegistryRefs     int `json:"generatedRegistryRefs"`
	MarkedUninstalledModules  int `json:"markedUninstalledModules"`
	ArtifactReadyModules      int `json:"artifactReadyModules"`
	PreservedUninstalledCount int `json:"preservedUninstalledCount"`
}

type ActivationAuditSummary struct {
	CheckedModules       int `json:"checkedModules"`
	ActivatedModules     int `json:"activatedModules"`
	PendingModules       int `json:"pendingModules"`
	RuntimeReadyModules  int `json:"runtimeReadyModules"`
	FrontendReadyModules int `json:"frontendReadyModules"`
}

// TableName 指定表名
func (ModuleRegistration) TableName() string {
	return "system_module_registration"
}

// RegisterModule 注册新模块
// 1. 执行数据库迁移
// 2. 导入菜单/权限/i18n
// 3. 注册到模块注册表
// 4. 返回安装状态
func (s *DynamicModuleService) RegisterModule(module contracts.BackendModule) error {
	if s.db == nil {
		return nil
	}

	moduleName := module.Name()

	var count int64
	if err := s.db.Table("system_module_registration").
		Where("name = ? AND status = ?", moduleName, ModuleStatusActive).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if err := module.Migrate(s.db); err != nil {
		return err
	}
	if err := module.SeedMenus(s.db); err != nil {
		return err
	}
	if err := module.SeedPerms(s.db); err != nil {
		return err
	}
	if err := module.SeedI18n(s.db); err != nil {
		return err
	}

	registration := ModuleRegistration{
		Name:        moduleName,
		Status:      ModuleStatusActive,
		InstalledAt: time.Now().Format(time.RFC3339),
	}
	return s.db.Table("system_module_registration").Create(&registration).Error
}

func (s *DynamicModuleService) RegisterGeneratedModule(req *scaffold.RegisterGeneratedModuleRequest) (*ModuleRegistration, []string, *GeneratedModuleRegistrationSummary, error) {
	if s.db == nil {
		return nil, nil, nil, common.ErrDatabaseNotInitialized
	}
	if err := scaffold.ValidateRegisterRequest(req); err != nil {
		return nil, nil, nil, err
	}
	if strings.TrimSpace(req.Schema.Scope) != "business" {
		return nil, nil, nil, common.NewBadRequest("module.generate.business_only")
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, nil, nil, common.NewNotFound("workspace.not_found")
	}

	moduleKey := buildModuleKey(req.Schema.Scope, req.Schema.Name)

	var existing ModuleRegistration
	err := s.db.Where("name = ?", moduleKey).First(&existing).Error
	if err == nil && strings.TrimSpace(existing.ModelTableName) == "" {
		return nil, nil, nil, common.NewBadRequest("module.generate.reserved")
	}
	if err == nil && existing.Status != ModuleStatusUninstalled && !req.Overwrite {
		return nil, nil, nil, common.NewConflict("module.generate.already_exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, err
	}

	serverReq := *req
	serverReq.Files = nil

	writtenFiles, err := scaffold.WriteGeneratedModuleSource(s.workspaceRoot, &serverReq)
	if err != nil {
		return nil, nil, nil, err
	}

	now := time.Now().Format(time.RFC3339)
	existing.Name = moduleKey
	existing.DisplayName = req.Schema.DisplayName
	existing.Scope = req.Schema.Scope
	existing.Source = inferRegistrationSource(req.Schema.Scope, req.Schema.Metadata.SourceMode, req.Schema.Name, true)
	existing.Owner = strings.TrimSpace(req.Schema.Metadata.Owner)
	existing.BoundedContext = strings.TrimSpace(req.Schema.Metadata.BoundedContext)
	existing.Summary = strings.TrimSpace(req.Schema.Metadata.Summary)
	existing.SourceTable = strings.TrimSpace(req.Schema.Metadata.SourceTable)
	existing.AutoRecycle = req.Schema.Metadata.AutoRecycle
	existing.ModelTableName = req.Schema.Model.TableName
	existing.Status = ModuleStatusPendingActivation
	existing.InstalledAt = now
	existing.UninstalledAt = ""
	if err := s.db.Save(&existing).Error; err != nil {
		return nil, nil, nil, err
	}

	snapshot, _, err := s.refreshGeneratedWorkspaceArtifacts()
	if err != nil {
		_ = s.persistModuleDiagnostics(&existing, ModuleStatusFailed, err.Error(), []GeneratedModuleVerification{{
			Code:       "registry_write",
			Status:     "warn",
			MessageKey: "module.generate.verify.registry_check_failed",
			Detail:     err.Error(),
		}})
		return nil, nil, nil, err
	}

	existing.BuiltIn = false
	summary := s.buildGeneratedModuleSummary(&serverReq, writtenFiles)
	summary.Verifications = append(summary.Verifications, verifyFeatureLedgerSnapshot(snapshot))
	if err := s.persistModuleDiagnostics(&existing, ModuleStatusPendingActivation, "", summary.Verifications); err != nil {
		return nil, nil, nil, err
	}
	return &existing, writtenFiles, summary, nil
}

func (s *DynamicModuleService) RegisterManagedModule(moduleName string) (*ModuleRegistration, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	scope, shortName, err := splitModuleKey(moduleName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, common.NewNotFound("workspace.not_found")
	}

	var registration ModuleRegistration
	err = s.db.Where("name = ?", moduleName).First(&registration).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil && strings.TrimSpace(registration.ModelTableName) == "" {
		return nil, common.NewForbidden("module.register.builtin_forbidden")
	}
	if err == nil && registration.Status == ModuleStatusActive {
		registration.BuiltIn = false
		return &registration, nil
	}

	schema, err := s.loadGeneratedModuleSchema(scope, shortName)
	if err != nil {
		return nil, err
	}
	backendPath, ok := generatedModuleRelativePath("backend", "modules", scope, shortName)
	if !ok {
		return nil, common.NewBadRequest("module.invalid_name")
	}
	frontendPath, ok := generatedModuleRelativePath("frontend", "src", "modules", scope, shortName)
	if !ok {
		return nil, common.NewBadRequest("module.invalid_name")
	}
	if !generatedDirExists(s.workspaceRoot, backendPath) ||
		!generatedDirExists(s.workspaceRoot, frontendPath) {
		return nil, common.NewBadRequest("module.register.source_missing")
	}

	registration.Name = moduleName
	registration.DisplayName = strings.TrimSpace(schema.DisplayName)
	registration.Scope = strings.TrimSpace(schema.Scope)
	registration.Source = inferRegistrationSource(schema.Scope, schema.Metadata.SourceMode, schema.Name, true)
	registration.Owner = strings.TrimSpace(schema.Metadata.Owner)
	registration.BoundedContext = strings.TrimSpace(schema.Metadata.BoundedContext)
	registration.Summary = strings.TrimSpace(schema.Metadata.Summary)
	registration.SourceTable = strings.TrimSpace(schema.Metadata.SourceTable)
	registration.AutoRecycle = schema.Metadata.AutoRecycle
	registration.ModelTableName = strings.TrimSpace(schema.Model.TableName)
	registration.Status = ModuleStatusPendingActivation
	registration.InstalledAt = time.Now().Format(time.RFC3339)
	registration.UninstalledAt = ""
	if err := s.db.Save(&registration).Error; err != nil {
		return nil, err
	}

	if _, _, err := s.refreshGeneratedWorkspaceArtifacts(); err != nil {
		return nil, err
	}

	registration.BuiltIn = false
	return &registration, nil
}

func (s *DynamicModuleService) GetManagedModuleSchema(moduleName string) (*scaffold.ModuleSchema, error) {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, common.NewNotFound("workspace.not_found")
	}
	scope, shortName, err := splitModuleKey(moduleName)
	if err != nil {
		return nil, err
	}
	return s.loadGeneratedModuleSchema(scope, shortName)
}
