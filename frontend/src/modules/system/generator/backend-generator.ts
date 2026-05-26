/**
 * 模块生成器 - 后端代码生成器
 *
 * 基于项目真实代码结构生成后端 Go 代码:
 * - system/* 使用 package system
 * - business/* 使用 package {module}
 * - 支持基础/企业级两种模板级别
 *
 * 参考:
 * - system/user/user_model.go
 * - system/user/user_dto.go
 * - system/user/user_service.go
 * - system/user/user_handler.go
 * - business/cmdb/module.go
 * - auth/module.go
 */

import type { ModuleSchema } from './schema';
import {
  buildAuditActionKey,
  buildComponentKey,
  buildEnumOptionKey,
  buildFieldHelpTextKey,
  buildFieldLabelKey,
  buildFieldPlaceholderKey,
  buildMenuGroupTitleKey,
  buildModuleNamespace,
  buildPermissionTitleKey,
  buildPermissionPrefix,
  buildRouteName,
  buildRoutePath,
  buildTitleKey,
  normalizeMenuPath,
  getPageActions,
  inferMenuGroupDisplayName,
  inferPackageName,
  inferModelName,
  shouldGenerateNavigation,
  splitModuleSegments,
} from './schema';
import { TYPE_MAPPING, generateStructTags, getRequiredImports } from './type-mapping';

type StructTagOptions = NonNullable<Parameters<typeof generateStructTags>[2]>;

export class BackendGenerator {
  private schema: ModuleSchema;
  private packageName: string;
  private modelName: string;

  constructor(schema: ModuleSchema) {
    this.schema = schema;
    this.packageName = inferPackageName(schema);
    this.modelName = inferModelName(schema);
  }

  /**
   * 生成 model.go
   *
   * 参考: system/user/user_model.go, business/cmdb/cmdb_model.go
   */
  generateModel(): string {
    const { scope, model, templateLevel = 'enterprise' } = this.schema;
    const tableName = model.tableName;

    // 导入
    const imports = getRequiredImports(model.fields);
    const importBlock = Array.from(imports).join('\n\t');

    // Model 名称
    const structName = scope === 'system' ? `System${this.modelName}` : this.modelName;

    // 字段定义
    const fields = this.generateModelFields();

    // 企业级包含审计字段
    const auditFields =
      templateLevel === 'enterprise'
        ? `
\tCreatedAt time.Time      \`json:"createdAt"\`
\tUpdatedAt time.Time      \`json:"updatedAt"\`
\tDeletedAt gorm.DeletedAt \`gorm:"index" json:"-"\``
        : `
\tCreatedAt time.Time \`json:"createdAt"\``;

    const junctionModels = this.generateManyToManyJunctionModels();

    return `package ${this.packageName}

import (
\t${importBlock}
)

// ${structName} ${this.schema.displayName}模型
type ${structName} struct {
\tID uint64 \`gorm:"primaryKey;autoIncrement" json:"id"\`
${fields}${auditFields}
}

// TableName 指定表名
func (${structName}) TableName() string {
\treturn "${tableName}"
}
${junctionModels ? `\n${junctionModels}` : ''}
`;
  }

  /**
   * 生成 model 字段
   */
  private generateModelFields(): string {
    return this.schema.model.fields
      .filter((f) => f.visibleInForm !== false)
      .map((field) => {
        const mapping = TYPE_MAPPING[field.type];
        const options: StructTagOptions = {
          notNull: field.required,
          jsonName: field.name,
        };

        if (field.validation?.unique) {
          options.unique = true;
        }

        const tags = generateStructTags(field.name, field.type, options);
        return `\t${this.capitalize(field.name)} ${mapping.go} \`${tags}\``;
      })
      .join('\n');
  }

  /**
   * 生成 dto.go
   *
   * 参考: system/user/user_dto.go
   */
  generateDTO(): string {
    const structName = this.modelName;
    const importBlock = this.generateDTOImportBlock();
    const relationDTOs = this.generateManyToManyRelationDTOs();

    return `package ${this.packageName}

${importBlock}

// ${structName}ListResp 列表页返回 DTO
type ${structName}ListResp struct {
\tID uint64 \`json:"id"\`
${this.generateDTOFields('list')}
\tCreatedAt string \`json:"createdAt"\`
}

// ${structName}ListPageResp 分页响应
type ${structName}ListPageResp struct {
\tItems    []${structName}ListResp \`json:"items"\`
\tTotal    int64                   \`json:"total"\`
\tPage     int                     \`json:"page"\`
\tPageSize int                     \`json:"pageSize"\`
}

// ${structName}DetailResp 详情返回 DTO
type ${structName}DetailResp struct {
\tID uint64 \`json:"id"\`
${this.generateDTOFields('detail')}
\tCreatedAt string \`json:"createdAt"\`
\tUpdatedAt string \`json:"updatedAt"\`
}

// ${structName}ListQuery 查询参数
type ${structName}ListQuery struct {
${this.generateQueryFields()}
\tPage int \`form:"page" json:"page"\`
\tPageSize int \`form:"pageSize" json:"pageSize"\`
\tSortField string \`form:"sortField" json:"sortField"\`
\tSortOrder string \`form:"sortOrder" json:"sortOrder"\`
}

// ${structName}CreateReq 创建请求
type ${structName}CreateReq struct {
${this.generateDTOFields('create')}
}

// ${structName}UpdateReq 更新请求
type ${structName}UpdateReq struct {
${this.generateDTOFields('update')}
}

type ${structName}OptionItem struct {
\tLabel string \`json:"label"\`
\tValue uint64 \`json:"value"\`
\tID    uint64 \`json:"id"\`
\tName  string \`json:"name"\`
}
${relationDTOs ? `\n${relationDTOs}` : ''}
`;
  }

  private generateDTOImportBlock(): string {
    const dtoImports = new Set<string>();
    const usesTime = this.schema.model.fields.some(
      (field) => this.goTypeFromField(field.type) === 'time.Time',
    );
    if (usesTime) {
      dtoImports.add('"time"');
    }
    if (dtoImports.size === 0) {
      return '';
    }
    return `import (
\t${Array.from(dtoImports).join('\n\t')}
)`;
  }

  /**
   * 生成 DTO 字段
   */
  private generateDTOFields(mode: 'list' | 'detail' | 'create' | 'update'): string {
    return this.schema.model.fields
      .filter((f) => {
        if (mode === 'list') return f.visibleInList !== false;
        if (mode === 'create' || mode === 'update') return f.visibleInForm !== false;
        return true;
      })
      .map((field) => {
        const baseType = this.goTypeFromField(field.type);
        const tsType = mode === 'update' ? `*${baseType}` : baseType;
        const isRequired = mode === 'create' && field.required;
        const binding = isRequired ? ' binding:"required"' : '';
        return `\t${this.capitalize(field.name)} ${tsType} \`json:"${field.name}"${binding}\``;
      })
      .join('\n');
  }

  /**
   * 生成查询字段
   */
  private generateQueryFields(): string {
    const searchableFields = this.schema.model.fields.filter((f) => f.searchable);

    if (searchableFields.length === 0) {
      return '\t// 无搜索字段';
    }

    return searchableFields
      .map((field) => {
        const tsType = this.goTypeFromField(field.type);
        // 可搜索字段通常可选
        const isPointer = field.type === 'int' || field.type === 'float';
        const goType = isPointer ? `*${tsType}` : tsType;
        return `\t${this.capitalize(field.name)} ${goType} \`form:"${field.name}" json:"${field.name}"\``;
      })
      .join('\n');
  }

  /**
   * 生成 service.go
   *
   * 参考: system/user/user_service.go
   */
  generateService(): string {
    const { scope, templateLevel = 'enterprise' } = this.schema;
    const structName = scope === 'system' ? `System${this.modelName}` : this.modelName;
    const modelName = this.modelName;

    const hasDataScope = this.schema.enableDataScope ?? templateLevel === 'enterprise';
    const relationMigrations = this.generateManyToManyMigrations();
    const relationServices = this.generateManyToManyServiceMethods();
    const requiresStrconv =
      this.getManyToManyRelations().length > 0 || this.resolveOptionLabelField() === null;
    return `package ${this.packageName}

import (
\t"errors"
\t${hasDataScope ? `"pantheon-ops/backend/pkg/common"` : ``}
\t${hasDataScope ? `"pantheon-ops/backend/pkg/database"` : ``}
\t${requiresStrconv ? `"strconv"` : ``}
\t"strings"
\t"time"
\t"gorm.io/gorm"
)

type ${modelName}Service struct {
\tdb *gorm.DB
}

// New${modelName}Service 构造函数
func New${modelName}Service(db *gorm.DB) *${modelName}Service {
\treturn &${modelName}Service{db: db}
}

// Migrate 数据库迁移
func (s *${modelName}Service) Migrate() error {
\tif s.db == nil {
\t\treturn errors.New("database.not_initialized")
\t}
\tif err := s.db.AutoMigrate(&${structName}{}${relationMigrations}); err != nil {
\t\treturn err
\t}
\treturn nil
}

// List${modelName}s 分页列表查询
func (s *${modelName}Service) List${modelName}s(query *${modelName}ListQuery${hasDataScope ? ', dataScope *common.DataScopeReq' : ''}) (*${modelName}ListPageResp, error) {
\tif query == nil {
\t\tquery = &${modelName}ListQuery{}
\t}
\tif query.Page <= 0 {
\t\tquery.Page = 1
\t}
\tif query.PageSize <= 0 || query.PageSize > 100 {
\t\tquery.PageSize = 10
\t}
\tvar items []${structName}
\tvar total int64

\tdb := s.db.Model(&${structName}{})${hasDataScope ? `.Scopes(database.WithDataScope(dataScope))` : ``}
\t${this.generateQueryFilters()}

\tif err := db.Count(&total).Error; err != nil {
\t\treturn nil, err
\t}

\t// 分页和排序
\toffset := (query.Page - 1) * query.PageSize
\torderBy := "id desc"
\tsortFieldMap := map[string]string{
${this.generateSortFieldMap()}
\t}
\tif column, ok := sortFieldMap[strings.TrimSpace(query.SortField)]; ok {
\t\torderBy = column + " desc"
\t\tif strings.EqualFold(strings.TrimSpace(query.SortOrder), "asc") {
\t\t\torderBy = column + " asc"
\t\t}
\t}

\tif err := db.Order(orderBy).Offset(offset).Limit(query.PageSize).Find(&items).Error; err != nil {
\t\treturn nil, err
\t}

\t// 转换为 DTO
\tlistResp := make([]${modelName}ListResp, len(items))
\tfor i, item := range items {
\t\tlistResp[i] = s.toListResp(item)
\t}

\treturn &${modelName}ListPageResp{
\t\tItems:    listResp,
\t\tTotal:    total,
\t\tPage:     query.Page,
\t\tPageSize: query.PageSize,
\t}, nil
}

// List${modelName}Options 关系选择器选项
func (s *${modelName}Service) List${modelName}Options() ([]${modelName}OptionItem, error) {
\tvar items []${structName}
\tif err := s.db.Model(&${structName}{}).Order("id desc").Limit(100).Find(&items).Error; err != nil {
\t\treturn nil, err
\t}
\toptions := make([]${modelName}OptionItem, len(items))
\tfor i, item := range items {
\t\toptions[i] = ${modelName}OptionItem{
\t\t\tLabel: ${this.generateOptionLabelExpression('item')},
\t\t\tValue: item.ID,
\t\t\tID: item.ID,
\t\t\tName: ${this.generateOptionNameExpression('item')},
\t\t}
\t}
\treturn options, nil
}

// Get${modelName}Detail 详情查询
func (s *${modelName}Service) Get${modelName}Detail(id uint64) (*${modelName}DetailResp, error) {
\tvar item ${structName}
\tif err := s.db.First(&item, id).Error; err != nil {
\t\tif errors.Is(err, gorm.ErrRecordNotFound) {
\t\t\treturn nil, errors.New("${modelName.toLowerCase()}.not_found")
\t\t}
\t\treturn nil, err
\t}
\tdetail := s.toDetailResp(item)
\treturn &detail, nil
}

// Create${modelName} 创建
func (s *${modelName}Service) Create${modelName}(req *${modelName}CreateReq) (*${modelName}ListResp, error) {
\titem := s.fromCreateReq(req)
\tif err := s.db.Create(&item).Error; err != nil {
\t\treturn nil, err
\t}
\tresp := s.toListResp(item)
\treturn &resp, nil
}

// Update${modelName} 更新
func (s *${modelName}Service) Update${modelName}(id uint64, req *${modelName}UpdateReq) (*${modelName}ListResp, error) {
\tvar item ${structName}
\tif err := s.db.First(&item, id).Error; err != nil {
\t\tif errors.Is(err, gorm.ErrRecordNotFound) {
\t\t\treturn nil, errors.New("${modelName.toLowerCase()}.not_found")
\t\t}
\t\treturn nil, err
\t}

\t// 更新字段
${this.generateUpdateFields()}

\tif err := s.db.Save(&item).Error; err != nil {
\t\treturn nil, err
\t}
\tresp := s.toListResp(item)
\treturn &resp, nil
}

// Delete${modelName} 删除(软删除)
func (s *${modelName}Service) Delete${modelName}(id uint64) error {
\tresult := s.db.Delete(&${structName}{}, id)
\tif result.Error != nil {
\t\treturn result.Error
\t}
\tif result.RowsAffected == 0 {
\t\treturn errors.New("${modelName.toLowerCase()}.not_found")
\t}
\treturn nil
}

// ========== DTO 转换方法 ==========

func (s *${modelName}Service) toListResp(item ${structName}) ${modelName}ListResp {
\treturn ${modelName}ListResp{
\t\tID: item.ID,
${this.toListRespFields()}
\t\tCreatedAt: item.CreatedAt.Format(time.RFC3339),
\t}
}

func (s *${modelName}Service) toDetailResp(item ${structName}) ${modelName}DetailResp {
\treturn ${modelName}DetailResp{
\t\tID: item.ID,
${this.toDetailRespFields()}
\t\tCreatedAt: item.CreatedAt.Format(time.RFC3339),
\t\tUpdatedAt: item.UpdatedAt.Format(time.RFC3339),
\t}
}

func (s *${modelName}Service) fromCreateReq(req *${modelName}CreateReq) ${structName} {
\treturn ${structName}{
${this.fromCreateReqFields()}
\t}
}
${relationServices ? `\n${relationServices}` : ''}
`;
  }

  /**
   * 生成查询过滤条件
   */
  private generateQueryFilters(): string {
    const searchableFields = this.schema.model.fields.filter((f) => f.searchable);

    if (searchableFields.length === 0) {
      return '// 无搜索条件';
    }

    return searchableFields
      .map((field) => {
        const fieldName = this.capitalize(field.name);
        const columnName = this.toDBColumn(field.name);
        const isPointer = field.type === 'int' || field.type === 'float';
        if (isPointer) {
          return `if query.${fieldName} != nil {
\t\tdb = db.Where("${columnName} = ?", query.${fieldName})
\t}`;
        } else {
          return `if query.${fieldName} != "" {
\t\tdb = db.Where("${columnName} LIKE ?", "%"+query.${fieldName}+"%")
\t}`;
        }
      })
      .join('\n\t');
  }

  private toDBColumn(name: string): string {
    return name
      .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
      .replace(/[-\s]+/g, '_')
      .toLowerCase();
  }

  private generateSortFieldMap(): string {
    const sortableFields = this.schema.model.fields.filter((field) => field.sortable);
    return sortableFields
      .map((field) => `\t\t"${field.name}": "${this.toDBColumn(field.name)}",`)
      .concat(['\t\t"createdAt": "created_at",', '\t\t"updatedAt": "updated_at",'])
      .join('\n');
  }

  /**
   * 生成更新字段
   */
  private generateUpdateFields(): string {
    return this.schema.model.fields
      .filter((f) => f.visibleInForm !== false)
      .map((field) => {
        const fieldName = this.capitalize(field.name);
        return `\tif req.${fieldName} != nil {
\t\titem.${fieldName} = *req.${fieldName}
\t}`;
      })
      .join('\n');
  }

  /**
   * 生成 toListResp 字段
   */
  private toListRespFields(): string {
    return this.schema.model.fields
      .filter((f) => f.visibleInList !== false)
      .map((field) => `\t\t${this.capitalize(field.name)}: item.${this.capitalize(field.name)},`)
      .join('\n');
  }

  /**
   * 生成 toDetailResp 字段
   */
  private toDetailRespFields(): string {
    return this.schema.model.fields
      .map((field) => `\t\t${this.capitalize(field.name)}: item.${this.capitalize(field.name)},`)
      .join('\n');
  }

  /**
   * 生成 fromCreateReq 字段
   */
  private fromCreateReqFields(): string {
    return this.schema.model.fields
      .filter((f) => f.visibleInForm !== false)
      .map((field) => `\t\t${this.capitalize(field.name)}: req.${this.capitalize(field.name)},`)
      .join('\n');
  }

  /**
   * 生成 handler.go
   *
   * 参考: system/user/user_handler.go
   */
  generateHandler(): string {
    const { templateLevel = 'enterprise' } = this.schema;
    const modelName = this.modelName;
    const hasAudit = templateLevel === 'enterprise';
    const hasDataScope = this.schema.enableDataScope ?? templateLevel === 'enterprise';
    const relationHandlers = this.generateManyToManyHandlerMethods();

    return `package ${this.packageName}

import (
\t"pantheon-ops/backend/pkg/common"
\t"strconv"
\t"github.com/gin-gonic/gin"
)

type ${modelName}Handler struct {
\tservice *${modelName}Service
}

func New${modelName}Handler(s *${modelName}Service) *${modelName}Handler {
\treturn &${modelName}Handler{service: s}
}

// Get${modelName}List 获取列表
func (h *${modelName}Handler) Get${modelName}List(c *gin.Context) {
\tvar query ${modelName}ListQuery
\tif err := c.ShouldBindQuery(&query); err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

${hasDataScope ? '\tdataScope := common.GetDataScope(c)' : ''}
\tlist, err := h.service.List${modelName}s(&query${hasDataScope ? ', dataScope' : ''})
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.list.error")
\t\treturn
\t}
\tcommon.Success(c, list)
}

// Get${modelName}Options 获取关系选项
func (h *${modelName}Handler) Get${modelName}Options(c *gin.Context) {
\toptions, err := h.service.List${modelName}Options()
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.options.error")
\t\treturn
\t}
\tcommon.Success(c, options)
}

// Get${modelName}Detail 获取详情
func (h *${modelName}Handler) Get${modelName}Detail(c *gin.Context) {
\tid, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\tdetail, err := h.service.Get${modelName}Detail(id)
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.detail.error")
\t\treturn
\t}
\tcommon.Success(c, detail)
}

// Create${modelName} 创建
func (h *${modelName}Handler) Create${modelName}(c *gin.Context) {
${hasAudit ? `\tcommon.SetAuditMetadata(c, "${buildAuditActionKey(this.schema.scope, this.schema.name, 'create')}", common.BusinessInsert)` : ''}
\tvar req ${modelName}CreateReq
\tif err := c.ShouldBindJSON(&req); err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\titem, err := h.service.Create${modelName}(&req)
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.create.error")
\t\treturn
\t}
\tcommon.Success(c, item)
}

// Update${modelName} 更新
func (h *${modelName}Handler) Update${modelName}(c *gin.Context) {
${hasAudit ? `\tcommon.SetAuditMetadata(c, "${buildAuditActionKey(this.schema.scope, this.schema.name, 'update')}", common.BusinessUpdate)` : ''}
\tvar req ${modelName}UpdateReq
\tif err := c.ShouldBindJSON(&req); err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\tid, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\titem, err := h.service.Update${modelName}(id, &req)
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.update.error")
\t\treturn
\t}
\tcommon.Success(c, item)
}

// Delete${modelName} 删除
func (h *${modelName}Handler) Delete${modelName}(c *gin.Context) {
${hasAudit ? `\tcommon.SetAuditMetadata(c, "${buildAuditActionKey(this.schema.scope, this.schema.name, 'delete')}", common.BusinessDelete)` : ''}
\tid, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\tif err := h.service.Delete${modelName}(id); err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.delete.error")
\t\treturn
\t}
\tcommon.Success(c, gin.H{"deleted": true})
}

// parseUintParam 解析路径参数
func parseUintParam(c *gin.Context, key string) (uint64, error) {
\treturn strconv.ParseUint(c.Param(key), 10, 64)
}
${relationHandlers ? `\n${relationHandlers}` : ''}
`;
  }

  /**
   * 生成 module.go
   *
   * 参考: business/cmdb/module.go, auth/module.go
   */
  generateModule(): string {
    const modelName = this.modelName;
    const { scope } = this.schema;
    const moduleName = buildModuleNamespace(scope, this.schema.name);
    const pageTitleKey = buildTitleKey(scope, this.schema.name);
    const componentKey = buildComponentKey(scope, this.schema.name, modelName);
    const routePath = buildRoutePath(scope, this.schema.name);
    const relationRoutes = this.generateManyToManyRouteRegistrations();

    return `package ${this.packageName}

import (
\t"pantheon-ops/backend/internal/middleware"
\t"pantheon-ops/backend/pkg/contracts"
\t"strings"
\t"github.com/gin-gonic/gin"
\t"gorm.io/gorm"
)

type generatedMenuSeed struct {
\tKey       string
\tParentKey string
\tParentPath string
\tTitleKey  string
\tPath      string
\tComponent string
\tPagePerm  string
\tPerms     string
\tType      string
\tIcon      string
\tRouteName string
\tModule    string
\tSort      int
}

type generatedI18nSeed struct {
\tModule string
\tLocale string
\tGroup  string
\tKey    string
\tValue  string
}

var generatedMenuSeeds = []generatedMenuSeed{
${this.generateMenuSeedEntries(componentKey, pageTitleKey)}
}

var generatedI18nSeeds = []generatedI18nSeed{
${this.generateI18nSeedEntries(pageTitleKey)}
}

func Init${modelName}Module(r *gin.RouterGroup, db *gorm.DB) {
\tservice := New${modelName}Service(db)
\thandler := New${modelName}Handler(service)

\tcontracts.RegisterBackendModules(r, db, contracts.FuncModule{
\t\tModuleName:  "${moduleName}",
\t\tMigrateFunc: func(_ *gorm.DB) error { return service.Migrate() },
\t\tSeedMenusFunc: seed${modelName}Menus,
\t\tSeedI18nFunc: seed${modelName}I18n,
\t\tRegister: func(r *gin.RouterGroup) {
\t\t\tprotected := r.Group("${routePath}").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware())
\t\t\t{
\t\t\t\tprotected.GET("/list", handler.Get${modelName}List)
\t\t\t\tprotected.GET("/options", handler.Get${modelName}Options)
\t\t\t\tprotected.GET("/:id", handler.Get${modelName}Detail)
\t\t\t\t${relationRoutes}
\t\t\t\tprotected.POST("", handler.Create${modelName})
\t\t\t\tprotected.PUT("/:id", handler.Update${modelName})
\t\t\t\tprotected.DELETE("/:id", handler.Delete${modelName})
\t\t\t}
\t\t},
\t})
}

func seed${modelName}Menus(db *gorm.DB) error {
\tif db == nil || !db.Migrator().HasTable("system_menu") {
\t\treturn nil
\t}
\treturn db.Transaction(func(tx *gorm.DB) error {
\t\tkeyToID := make(map[string]uint64, len(generatedMenuSeeds))
\t\tfor _, seed := range generatedMenuSeeds {
\t\t\tif _, err := ensureGeneratedMenuSeed(tx, keyToID, seed); err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t}
\t\treturn nil
\t})
}

func ensureGeneratedMenuSeed(tx *gorm.DB, keyToID map[string]uint64, seed generatedMenuSeed) (uint64, error) {
\tvar menuID uint64
\tif seed.Path != "" {
\t\tif err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
\t\t\treturn 0, err
\t\t}
\t} else if seed.Perms != "" {
\t\tif err := tx.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
\t\t\treturn 0, err
\t\t}
\t}

\tparentID := uint64(0)
\tif seed.ParentKey != "" {
\t\tparentID = keyToID[seed.ParentKey]
\t}
\tif parentID == 0 && seed.ParentPath != "" {
\t\tif err := tx.Table("system_menu").Select("id").Where("path = ?", seed.ParentPath).Limit(1).Pluck("id", &parentID).Error; err != nil {
\t\t\treturn 0, err
\t\t}
\t}

\tpayload := map[string]interface{}{
\t\t"parent_id":  parentID,
\t\t"title_key":  seed.TitleKey,
\t\t"path":       seed.Path,
\t\t"component":  seed.Component,
\t\t"page_perm":  seed.PagePerm,
\t\t"perms":      seed.Perms,
\t\t"type":       seed.Type,
\t\t"icon":       seed.Icon,
\t\t"route_name": seed.RouteName,
\t\t"module":     seed.Module,
\t\t"sort":       seed.Sort,
\t\t"is_visible": 1,
\t}

\tif menuID == 0 {
\t\tif err := tx.Table("system_menu").Create(payload).Error; err != nil {
\t\t\treturn 0, err
\t\t}
\t\tif seed.Path != "" {
\t\t\tif err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
\t\t\t\treturn 0, err
\t\t\t}
\t\t} else if seed.Perms != "" {
\t\t\tif err := tx.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
\t\t\t\treturn 0, err
\t\t\t}
\t\t}
\t} else if err := tx.Table("system_menu").Where("id = ?", menuID).Updates(payload).Error; err != nil {
\t\treturn 0, err
\t}

\tif seed.Key != "" {
\t\tkeyToID[seed.Key] = menuID
\t}
\tif err := bindGeneratedSeedToAdmin(tx, menuID, seed); err != nil {
\t\treturn 0, err
\t}
\treturn menuID, nil
}

func bindGeneratedSeedToAdmin(tx *gorm.DB, menuID uint64, seed generatedMenuSeed) error {
\tif menuID == 0 || !tx.Migrator().HasTable("system_role") {
\t\treturn nil
\t}

\tvar adminRoleID uint64
\tif err := tx.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
\t\treturn err
\t}
\tif adminRoleID == 0 {
\t\treturn nil
\t}

\tif seed.Type == "C" && tx.Migrator().HasTable("system_role_menu") {
\t\tvar count int64
\t\tif err := tx.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", adminRoleID, menuID).Count(&count).Error; err != nil {
\t\t\treturn err
\t\t}
\t\tif count == 0 {
\t\t\tif err := tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, menuID).Error; err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t}
\t}

\tif tx.Migrator().HasTable("system_role_permission") {
\t\tfor _, permissionKey := range []string{strings.TrimSpace(seed.PagePerm), strings.TrimSpace(seed.Perms)} {
\t\t\tif permissionKey == "" {
\t\t\t\tcontinue
\t\t\t}
\t\t\tvar count int64
\t\t\tif err := tx.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", adminRoleID, permissionKey).Count(&count).Error; err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t\tif count == 0 {
\t\t\t\tif err := tx.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", adminRoleID, permissionKey).Error; err != nil {
\t\t\t\t\treturn err
\t\t\t\t}
\t\t\t}
\t\t}
\t}

\treturn nil
}

func seed${modelName}I18n(db *gorm.DB) error {
\tif db == nil || !db.Migrator().HasTable("system_i18n") {
\t\treturn nil
\t}
\tfor _, seed := range generatedI18nSeeds {
\t\tvar count int64
\t\tif err := db.Table("system_i18n").Where("module = ? AND locale = ? AND \`key\` = ?", seed.Module, seed.Locale, seed.Key).Count(&count).Error; err != nil {
\t\t\treturn err
\t\t}
\t\tpayload := map[string]interface{}{
\t\t\t"module":     seed.Module,
\t\t\t"group_name": seed.Group,
\t\t\t"key":        seed.Key,
\t\t\t"locale":     seed.Locale,
\t\t\t"value":      seed.Value,
\t\t}
\t\tif count == 0 {
\t\t\tif err := db.Table("system_i18n").Create(payload).Error; err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t\tcontinue
\t\t}
\t\tif err := db.Table("system_i18n").Where("module = ? AND locale = ? AND \`key\` = ?", seed.Module, seed.Locale, seed.Key).Updates(map[string]interface{}{
\t\t\t"group_name": seed.Group,
\t\t\t"value":      seed.Value,
\t\t}).Error; err != nil {
\t\t\treturn err
\t\t}
\t}
\treturn nil
}
`;
  }

  private generateMenuSeedEntries(componentKey: string, pageTitleKey: string): string {
    if (!shouldGenerateNavigation(this.schema)) {
      return '';
    }
    const moduleKey = buildModuleNamespace(this.schema.scope, this.schema.name);
    const permissionPrefix = buildPermissionPrefix(this.schema.scope, this.schema.name);
    const routePath = buildRoutePath(this.schema.scope, this.schema.name);
    const routeName = buildRouteName(this.schema.scope, this.schema.name);
    const segments = splitModuleSegments(this.schema.name);
    const menuKey = segments.join('-');
    const explicitParentPath = normalizeMenuPath(this.schema.parentMenu || '');
    const shouldGenerateAncestorMenus = !explicitParentPath && segments.length > 1;
    const inferredParentPath = shouldGenerateAncestorMenus
      ? ''
      : segments.length > 1
        ? `/${this.schema.scope}/${segments.slice(0, -1).join('/')}`
        : '';
    const parentPath = normalizeMenuPath(explicitParentPath || inferredParentPath || '');
    const parentKey = shouldGenerateAncestorMenus ? segments.slice(0, -1).join('-') : '';
    const actionSeeds = getPageActions(this.schema)
      .filter((action) => action !== 'detail')
      .map((action) => ({
        action,
        title: this.renderActionTitle(action, 'zh'),
        titleEn: this.renderActionTitle(action, 'en'),
      }));

    const ancestorEntries = shouldGenerateAncestorMenus
      ? segments.slice(0, -1).map((_, index) => {
          const groupSegments = segments.slice(0, index + 1);
          const groupKey = groupSegments.join('-');
          const parentSegments = groupSegments.slice(0, -1);
          const groupModuleKey = `${this.schema.scope}.${groupSegments.join('.')}`;
          const groupTitleKey = buildMenuGroupTitleKey(this.schema.scope, groupSegments);
          return `\t{
\t\tKey:       "${groupKey}",
\t\tParentKey: "${parentSegments.join('-')}",
\t\tTitleKey:  "${groupTitleKey}",
\t\tPath:      "/${this.schema.scope}/${groupSegments.join('/')}",
\t\tType:      "M",
\t\tIcon:      "apps",
\t\tRouteName: "${this.schema.scope}-${groupSegments.join('-')}",
\t\tModule:    "${groupModuleKey}",
\t\tSort:      10,
\t},`;
        })
      : [];

    const mainSeed = `\t{
\t\tKey:       "${menuKey}",
\t\tParentKey: "${parentKey}",
\t\tParentPath: "${parentPath}",
\t\tTitleKey:  "${pageTitleKey}",
\t\tPath:      "${routePath}",
\t\tComponent: "${componentKey}",
\t\tPagePerm:  "${permissionPrefix}:list",
\t\tType:      "C",
\t\tIcon:      "apps",
\t\tRouteName: "${routeName}",
\t\tModule:    "${moduleKey}",
\t\tSort:      10,
\t},`;

    const actionEntries = actionSeeds.map(
      (item, index) => `\t{
\t\tKey:       "${menuKey}-${item.action}",
\t\tParentKey: "${menuKey}",
\t\tTitleKey:  "${buildPermissionTitleKey(this.schema.scope, this.schema.name, item.action)}",
\t\tPerms:     "${permissionPrefix}:${item.action}",
\t\tType:      "F",
\t\tModule:    "${moduleKey}",
\t\tSort:      ${index + 1},
\t},`,
    );

    return [...ancestorEntries, mainSeed, ...actionEntries].join('\n');
  }

  private generateI18nSeedEntries(pageTitleKey: string): string {
    const moduleKey = buildModuleNamespace(this.schema.scope, this.schema.name);
    const seenZh = new Set<string>();
    const seenEn = new Set<string>();
    const pushEntry = (
      entries: Array<{ group: string; key: string; value: string }>,
      seen: Set<string>,
      item: { group: string; key: string; value: string },
    ) => {
      if (seen.has(item.key)) {
        return;
      }
      seen.add(item.key);
      entries.push(item);
    };
    const getTranslation = (locale: 'zh' | 'en', key: string, fallback: string) => {
      return this.schema.i18n.translations[locale][key] || fallback;
    };
    const zhEntries: Array<{ group: string; key: string; value: string }> = [];
    const enEntries: Array<{ group: string; key: string; value: string }> = [];
    const segments = splitModuleSegments(this.schema.name);
    for (let index = 0; index < segments.length - 1; index += 1) {
      const groupSegments = segments.slice(0, index + 1);
      const groupTitleKey = buildMenuGroupTitleKey(this.schema.scope, groupSegments);
      const fallback = inferMenuGroupDisplayName(groupSegments[groupSegments.length - 1]);
      pushEntry(zhEntries, seenZh, {
        group: 'menu',
        key: groupTitleKey,
        value: getTranslation('zh', groupTitleKey, fallback),
      });
      pushEntry(enEntries, seenEn, {
        group: 'menu',
        key: groupTitleKey,
        value: getTranslation('en', groupTitleKey, fallback),
      });
    }
    pushEntry(zhEntries, seenZh, {
      group: 'menu',
      key: pageTitleKey,
      value: getTranslation('zh', pageTitleKey, this.schema.displayName),
    });
    pushEntry(zhEntries, seenZh, {
      group: 'page',
      key: `${moduleKey}.title`,
      value: getTranslation('zh', `${moduleKey}.title`, this.schema.displayName),
    });
    pushEntry(enEntries, seenEn, {
      group: 'menu',
      key: pageTitleKey,
      value: getTranslation(
        'en',
        pageTitleKey,
        this.schema.displayNameEn || this.schema.displayName,
      ),
    });
    pushEntry(enEntries, seenEn, {
      group: 'page',
      key: `${moduleKey}.title`,
      value: getTranslation(
        'en',
        `${moduleKey}.title`,
        this.schema.displayNameEn || this.schema.displayName,
      ),
    });

    for (const field of this.schema.model.fields) {
      const fieldLabelKey = buildFieldLabelKey(this.schema.scope, this.schema.name, field.name);
      pushEntry(zhEntries, seenZh, {
        group: 'field',
        key: fieldLabelKey,
        value: getTranslation('zh', fieldLabelKey, field.label),
      });
      pushEntry(enEntries, seenEn, {
        group: 'field',
        key: fieldLabelKey,
        value: getTranslation('en', fieldLabelKey, field.labelEn || field.label),
      });

      if (field.placeholder) {
        const placeholderKey = buildFieldPlaceholderKey(
          this.schema.scope,
          this.schema.name,
          field.name,
        );
        pushEntry(zhEntries, seenZh, {
          group: 'placeholder',
          key: placeholderKey,
          value: getTranslation('zh', placeholderKey, field.placeholder),
        });
        pushEntry(enEntries, seenEn, {
          group: 'placeholder',
          key: placeholderKey,
          value: getTranslation('en', placeholderKey, field.placeholderEn || field.placeholder),
        });
      }

      if (field.helpText) {
        const helpTextKey = buildFieldHelpTextKey(this.schema.scope, this.schema.name, field.name);
        pushEntry(zhEntries, seenZh, {
          group: 'help',
          key: helpTextKey,
          value: getTranslation('zh', helpTextKey, field.helpText),
        });
        pushEntry(enEntries, seenEn, {
          group: 'help',
          key: helpTextKey,
          value: getTranslation('en', helpTextKey, field.helpTextEn || field.helpText),
        });
      }

      for (const option of field.enumOptions ?? []) {
        const optionKey = buildEnumOptionKey(
          this.schema.scope,
          this.schema.name,
          field.name,
          option.value,
        );
        pushEntry(zhEntries, seenZh, {
          group: 'option',
          key: optionKey,
          value: getTranslation('zh', optionKey, option.label),
        });
        pushEntry(enEntries, seenEn, {
          group: 'option',
          key: optionKey,
          value: getTranslation('en', optionKey, option.labelEn || option.label),
        });
      }
    }

    const permissionActions = getPageActions(this.schema)
      .filter((action) => action !== 'detail')
      .map((action) => ({
        action,
        zh: this.renderActionTitle(action, 'zh'),
        en: this.renderActionTitle(action, 'en'),
      }));

    for (const permission of permissionActions) {
      pushEntry(zhEntries, seenZh, {
        group: 'permission',
        key: buildPermissionTitleKey(this.schema.scope, this.schema.name, permission.action),
        value: getTranslation(
          'zh',
          buildPermissionTitleKey(this.schema.scope, this.schema.name, permission.action),
          permission.zh,
        ),
      });
      pushEntry(enEntries, seenEn, {
        group: 'permission',
        key: buildPermissionTitleKey(this.schema.scope, this.schema.name, permission.action),
        value: getTranslation(
          'en',
          buildPermissionTitleKey(this.schema.scope, this.schema.name, permission.action),
          permission.en,
        ),
      });
    }

    for (const auditAction of ['create', 'update', 'delete'] as const) {
      pushEntry(zhEntries, seenZh, {
        group: 'audit',
        key: buildAuditActionKey(this.schema.scope, this.schema.name, auditAction),
        value: getTranslation(
          'zh',
          buildAuditActionKey(this.schema.scope, this.schema.name, auditAction),
          this.renderActionTitle(auditAction, 'zh'),
        ),
      });
      pushEntry(enEntries, seenEn, {
        group: 'audit',
        key: buildAuditActionKey(this.schema.scope, this.schema.name, auditAction),
        value: getTranslation(
          'en',
          buildAuditActionKey(this.schema.scope, this.schema.name, auditAction),
          this.renderActionTitle(auditAction, 'en'),
        ),
      });
    }

    for (const [key, value] of Object.entries(this.schema.i18n.translations.zh)) {
      pushEntry(zhEntries, seenZh, { group: this.inferI18nGroup(key), key, value });
    }
    for (const [key, value] of Object.entries(this.schema.i18n.translations.en)) {
      pushEntry(enEntries, seenEn, { group: this.inferI18nGroup(key), key, value });
    }

    return [
      ...zhEntries.map((item) => this.formatI18nSeed('zh-CN', item.group, item.key, item.value)),
      ...enEntries.map((item) => this.formatI18nSeed('en-US', item.group, item.key, item.value)),
    ].join('\n');
  }

  private renderActionTitle(action: string, locale: 'zh' | 'en'): string {
    const key = buildAuditActionKey(
      this.schema.scope,
      this.schema.name,
      action as 'create' | 'update' | 'delete',
    );
    const localeMap =
      locale === 'zh' ? this.schema.i18n.translations.zh : this.schema.i18n.translations.en;
    const fallbackMap =
      locale === 'zh' ? this.schema.i18n.translations.en : this.schema.i18n.translations.zh;
    return localeMap[key] || fallbackMap[key] || this.schema.displayName;
  }

  private formatI18nSeed(locale: string, group: string, key: string, value: string): string {
    return `\t{Module: "${buildModuleNamespace(this.schema.scope, this.schema.name)}", Locale: "${locale}", Group: "${group}", Key: "${this.escapeGoString(key)}", Value: "${this.escapeGoString(value)}"},`;
  }

  private inferI18nGroup(key: string): string {
    if (key.includes('.field.')) return 'field';
    if (key.includes('.permission.')) return 'permission';
    if (key.includes('.audit.')) return 'audit';
    if (key.endsWith('.title')) return 'menu';
    return 'page';
  }

  private getManyToManyRelations() {
    return (this.schema.relations ?? []).filter((relation) => relation.type === 'manyToMany');
  }

  private generateManyToManyJunctionModels(): string {
    const relations = this.getManyToManyRelations();
    if (relations.length === 0) {
      return '';
    }
    return relations
      .map((relation) => {
        const structName = this.getManyToManyStructName(relation.name);
        const ownerColumn = this.getManyToManyOwnerColumn(relation.localField);
        const targetColumn = this.getManyToManyTargetColumn(relation.targetModule, relation.targetField);
        const indexName = `idx_${this.toDBColumn(this.modelName)}_${this.toDBColumn(relation.name)}_rel`;
        return `type ${structName} struct {
\tOwnerID uint64 \`gorm:"column:${ownerColumn};not null;uniqueIndex:${indexName},priority:1" json:"ownerId"\`
\tTargetID uint64 \`gorm:"column:${targetColumn};not null;uniqueIndex:${indexName},priority:2" json:"targetId"\`
}

func (${structName}) TableName() string {
\treturn "${relation.junctionTable}"
}`;
      })
      .join('\n\n');
  }

  private generateManyToManyRelationDTOs(): string {
    const relations = this.getManyToManyRelations();
    if (relations.length === 0) {
      return '';
    }
    return relations
      .map((relation) => {
        const relationName = this.toPascalCase(relation.name);
        return `type ${relationName}RelationRow struct {
\tID    uint64 \`json:"id"\`
\tValue uint64 \`json:"value"\`
}

type ${relationName}RelationListResp struct {
\tItems []${relationName}RelationRow \`json:"items"\`
}

type ${relationName}RelationBindReq struct {
\tTargetIDs []interface{} \`json:"targetIds" binding:"required"\`
}`;
      })
      .join('\n\n');
  }

  private generateManyToManyMigrations(): string {
    const relations = this.getManyToManyRelations();
    if (relations.length === 0) {
      return '';
    }
    return relations.map((relation) => `, &${this.getManyToManyStructName(relation.name)}{}`).join('');
  }

  private generateManyToManyServiceMethods(): string {
    const relations = this.getManyToManyRelations();
    if (relations.length === 0) {
      return '';
    }
    return relations
      .map((relation) => {
        const relationName = this.toPascalCase(relation.name);
        const ownerColumn = this.getManyToManyOwnerColumn(relation.localField);
        const targetColumn = this.getManyToManyTargetColumn(relation.targetModule, relation.targetField);
        const tableName = relation.junctionTable;
        return `func (s *${this.modelName}Service) List${relationName}Relation(ownerID uint64) (*${relationName}RelationListResp, error) {
\ttype relationValueRow struct {
\t\tValue uint64 \`gorm:"column:value"\`
\t}
\tvar rows []relationValueRow
\tif err := s.db.Table("${tableName}").
\t\tSelect("${targetColumn} AS value").
\t\tWhere("${ownerColumn} = ?", ownerID).
\t\tOrder("${targetColumn} ASC").
\t\tFind(&rows).Error; err != nil {
\t\treturn nil, err
\t}
\titems := make([]${relationName}RelationRow, len(rows))
\tfor index, row := range rows {
\t\titems[index] = ${relationName}RelationRow{
\t\t\tID: row.Value,
\t\t\tValue: row.Value,
\t\t}
\t}
\treturn &${relationName}RelationListResp{Items: items}, nil
}

func (s *${this.modelName}Service) Bind${relationName}Relation(ownerID uint64, targetIDs []interface{}) error {
\tnormalizedTargetIDs, err := normalizeGeneratedRelationTargetIDs(targetIDs)
\tif err != nil {
\t\treturn err
\t}
\treturn s.db.Transaction(func(tx *gorm.DB) error {
\t\tfor _, targetID := range normalizedTargetIDs {
\t\t\tvar count int64
\t\t\tif err := tx.Table("${tableName}").
\t\t\t\tWhere("${ownerColumn} = ? AND ${targetColumn} = ?", ownerID, targetID).
\t\t\t\tCount(&count).Error; err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t\tif count > 0 {
\t\t\t\tcontinue
\t\t\t}
\t\t\tif err := tx.Table("${tableName}").Create(map[string]interface{}{
\t\t\t\t"${ownerColumn}": ownerID,
\t\t\t\t"${targetColumn}": targetID,
\t\t\t}).Error; err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t}
\t\treturn nil
\t})
}

func (s *${this.modelName}Service) Unbind${relationName}Relation(ownerID uint64, targetID uint64) error {
\treturn s.db.Table("${tableName}").
\t\tWhere("${ownerColumn} = ? AND ${targetColumn} = ?", ownerID, targetID).
\t\tDelete(nil).Error
}`;
      })
      .concat([
        `func normalizeGeneratedRelationTargetIDs(values []interface{}) ([]uint64, error) {
\tresult := make([]uint64, 0, len(values))
\tseen := make(map[uint64]struct{}, len(values))
\tfor _, rawValue := range values {
\t\tparsed, err := normalizeGeneratedRelationTargetID(rawValue)
\t\tif err != nil {
\t\t\treturn nil, err
\t\t}
\t\tif _, ok := seen[parsed]; ok {
\t\t\tcontinue
\t\t}
\t\tseen[parsed] = struct{}{}
\t\tresult = append(result, parsed)
\t}
\tif len(result) == 0 {
\t\treturn nil, errors.New("param.invalid")
\t}
\treturn result, nil
}

func normalizeGeneratedRelationTargetID(value interface{}) (uint64, error) {
\tswitch typed := value.(type) {
\tcase string:
\t\ttrimmed := strings.TrimSpace(typed)
\t\tif trimmed == "" {
\t\t\treturn 0, errors.New("param.invalid")
\t\t}
\t\tparsed, err := strconv.ParseUint(trimmed, 10, 64)
\t\tif err != nil || parsed == 0 {
\t\t\treturn 0, errors.New("param.invalid")
\t\t}
\t\treturn parsed, nil
\tcase float64:
\t\tparsed := uint64(typed)
\t\tif parsed == 0 || float64(parsed) != typed {
\t\t\treturn 0, errors.New("param.invalid")
\t\t}
\t\treturn parsed, nil
\tcase int:
\t\tif typed <= 0 {
\t\t\treturn 0, errors.New("param.invalid")
\t\t}
\t\treturn uint64(typed), nil
\tcase int64:
\t\tif typed <= 0 {
\t\t\treturn 0, errors.New("param.invalid")
\t\t}
\t\treturn uint64(typed), nil
\tcase uint64:
\t\tif typed == 0 {
\t\t\treturn 0, errors.New("param.invalid")
\t\t}
\t\treturn typed, nil
\tdefault:
\t\treturn 0, errors.New("param.invalid")
\t}
}`,
      ])
      .join('\n\n');
  }

  private generateManyToManyHandlerMethods(): string {
    const relations = this.getManyToManyRelations();
    if (relations.length === 0) {
      return '';
    }
    return relations
      .map((relation) => {
        const relationName = this.toPascalCase(relation.name);
        return `func (h *${this.modelName}Handler) Get${relationName}Relation(c *gin.Context) {
\townerID, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}
\tresp, err := h.service.List${relationName}Relation(ownerID)
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${this.modelName.toLowerCase()}.relation.list.error")
\t\treturn
\t}
\tcommon.Success(c, resp)
}

func (h *${this.modelName}Handler) Bind${relationName}Relation(c *gin.Context) {
\townerID, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}
\tvar req ${relationName}RelationBindReq
\tif err := c.ShouldBindJSON(&req); err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}
\tif err := h.service.Bind${relationName}Relation(ownerID, req.TargetIDs); err != nil {
\t\tcommon.Fail(c, common.CodeError, "${this.modelName.toLowerCase()}.relation.bind.error")
\t\treturn
\t}
\tcommon.Success(c, gin.H{"success": true})
}

func (h *${this.modelName}Handler) Unbind${relationName}Relation(c *gin.Context) {
\townerID, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}
\ttargetID, err := parseUintParam(c, "targetId")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}
\tif err := h.service.Unbind${relationName}Relation(ownerID, targetID); err != nil {
\t\tcommon.Fail(c, common.CodeError, "${this.modelName.toLowerCase()}.relation.unbind.error")
\t\treturn
\t}
\tcommon.Success(c, gin.H{"success": true})
}`;
      })
      .join('\n\n');
  }

  private generateManyToManyRouteRegistrations(): string {
    const relations = this.getManyToManyRelations();
    if (relations.length === 0) {
      return '';
    }
    return relations
      .map((relation) => {
        const relationName = this.toPascalCase(relation.name);
        return `protected.GET("/:id/relations/${relation.name}", handler.Get${relationName}Relation)
\t\t\t\tprotected.POST("/:id/relations/${relation.name}", handler.Bind${relationName}Relation)
\t\t\t\tprotected.DELETE("/:id/relations/${relation.name}/:targetId", handler.Unbind${relationName}Relation)`;
      })
      .join('\n\t\t\t\t');
  }

  private getManyToManyStructName(relationName: string): string {
    return `${this.modelName}${this.toPascalCase(relationName)}Relation`;
  }

  private getManyToManyOwnerColumn(localField: string): string {
    return `${this.inferCurrentEntityToken()}_${this.toDBColumn(localField)}`;
  }

  private getManyToManyTargetColumn(targetModule: string, targetField: string): string {
    return `${this.inferTargetEntityToken(targetModule)}_${this.toDBColumn(targetField)}`;
  }

  private inferCurrentEntityToken(): string {
    let tableName = this.schema.model.tableName || '';
    tableName = tableName.replace(/^(biz_|sys_|system_)/, '');
    const businessContext = this.toDBColumn(this.schema.metadata?.businessContext || '');
    if (businessContext && tableName.startsWith(`${businessContext}_`)) {
      tableName = tableName.slice(businessContext.length + 1);
    }
    return tableName || this.toDBColumn(this.schema.name.split('/').pop() || 'item');
  }

  private inferTargetEntityToken(targetModule: string): string {
    return this.toDBColumn(targetModule.split('/').pop() || 'item');
  }

  private resolveOptionLabelField(): string | null {
    const fields = this.schema.model.fields;
    const preferred = ['name', 'title', 'label', 'code'];
    for (const candidate of preferred) {
      const field = fields.find((item) => item.name === candidate);
      if (field) {
        return this.capitalize(field.name);
      }
    }
    const firstStringField = fields.find((item) => item.type === 'string' || item.type === 'text');
    if (firstStringField) {
      return this.capitalize(firstStringField.name);
    }
    return null;
  }

  private generateOptionLabelExpression(sourceVar: string): string {
    const labelField = this.resolveOptionLabelField();
    if (!labelField) {
      return `strconv.FormatUint(${sourceVar}.ID, 10)`;
    }
    return `${sourceVar}.${labelField}`;
  }

  private generateOptionNameExpression(sourceVar: string): string {
    const labelField = this.resolveOptionLabelField();
    if (!labelField) {
      return `strconv.FormatUint(${sourceVar}.ID, 10)`;
    }
    return `${sourceVar}.${labelField}`;
  }

  /**
   * 辅助函数: 首字母大写
   */
  private capitalize(str: string): string {
    return str.charAt(0).toUpperCase() + str.slice(1);
  }

  /**
   * 辅助函数: 字段类型转 Go 类型
   */
  private goTypeFromField(fieldType: string): string {
    return TYPE_MAPPING[fieldType as keyof typeof TYPE_MAPPING].go || 'string';
  }

  private escapeGoString(value: string): string {
    return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
  }

  private toPascalCase(value: string): string {
    return value
      .split(/[_-]/)
      .filter(Boolean)
      .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
      .join('');
  }
}
