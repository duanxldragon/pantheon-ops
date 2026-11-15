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
  buildPageRoutePath,
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
  isTenantTemplate,
  validateGeneratorContract,
} from './schema';
import { TYPE_MAPPING, generateStructTags, getRequiredImports } from './typeMapping';

/**
 * GO_MODULE_PATH 是生成物 Go 导入前缀的唯一来源（AUD-001）。
 *
 * 必须与 backend/go.mod 的 module 行一致：模板此前手写 `pantheon-base/...` 前缀，
 * 而 internal/scaffold/workspace.go 的注册器写真实模块路径，两处不同源，
 * 生成包因此从未编译成功。tests/generator/generator-quality-contract.test.ts
 * 直接读 backend/go.mod 钉住这条常量——改模块名但忘改常量会先在测试里红。
 */
export const GO_MODULE_PATH = 'github.com/duanxldragon/pantheon-base/backend';

/** 由单一模块路径常量派生项目内部 Go 导入路径（模板侧不得再手写前缀）。 */
function goImportPath(subpath: string): string {
  return `${GO_MODULE_PATH}/${subpath}`;
}

/**
 * 组装 Go 导入块（AUD-001b）。
 *
 * gofmt 会把同一个导入组按路径字典序排序，因此这里必须排序后再发出；
 * 此前模板按写入顺序 emit，`gofmt -l` 会点名每一个生成文件。
 * 条目允许传带引号（`typeMapping.getRequiredImports` 的历史格式）或不带
 * 引号的路径，统一在 `normalizeGoImportPaths` 归一化；falsy 条目直接丢弃，
 * 这样条件导入不再留下纯制表符行。
 */
function normalizeGoImportPaths(importPaths: Array<string | false | null | undefined>): string[] {
  return (
    importPaths
      .map((importPath) =>
        typeof importPath === 'string' ? importPath.trim().replace(/^"|"$/g, '') : '',
      )
      .filter((importPath) => importPath.length > 0)
      // Go import paths use byte order, keeping emitted blocks gofmt-stable.
      .sort((left, right) => (left < right ? -1 : Number(left > right)))
  );
}

function buildGoImportBlock(importPaths: Array<string | false | null | undefined>): string {
  return normalizeGoImportPaths(importPaths)
    .map((importPath) => `\t"${importPath}"`)
    .join('\n');
}

/**
 * 按 gofmt 的列对齐规则渲染一个连续块（AUD-001b 的根因修复点）。
 *
 * gofmt 用空格（不是制表符）补齐：块内除最后一列外，每列右侧补空格到
 * 「该列最宽单元格 + 1」。模板此前要么写死单空格、要么按想象中的列宽手填
 * 空格，两者都会让产物落在 `gofmt -l` 里。调用方必须把「在产物里物理连续、
 * 且列结构相同」的行放进同一个块——中间夹空行、注释行或以 `{` 结尾的行会
 * 打断 gofmt 的列块，所以那些行不进 rows。
 */
function alignGoBlock(rows: string[][], indent: string): string {
  if (rows.length === 0) {
    return '';
  }
  const widths: number[] = [];
  for (const row of rows) {
    for (let column = 0; column < row.length - 1; column += 1) {
      widths[column] = Math.max(widths[column] ?? 0, row[column].length);
    }
  }
  return rows
    .map(
      (row) =>
        indent +
        row
          .map((cell, column) => (column < row.length - 1 ? cell.padEnd(widths[column] + 1) : cell))
          .join(''),
    )
    .join('\n');
}

/**
 * Go 产物的空白归一（AUD-001b）。
 *
 * 只收口 gofmt 一定会做的三件事：剥行尾空白、把连续空行压成一个空行、
 * 文件末尾只留一个换行。列对齐不在这里做——那是 `alignGoBlock` 在模板里
 * 逐块负责的，否则等于把模板缺陷推给事后格式化。
 */
function normalizeGoWhitespace(source: string): string {
  return (
    source
      .split('\n')
      .map((line) => line.trimEnd())
      .join('\n')
      // 空体：gofmt 把空结构体声明写成 `struct{}`、空复合字面量/映射写成 `{}`，
      // 两者都不会保留 `{` 与 `}` 之间的空行（全部字段不进表单时会撞上）。
      .replace(/struct \{\n\n[ \t]*\}/gu, 'struct{}')
      .replace(/\{\n\n[ \t]*\}/gu, '{}')
      .replace(/\n{3,}/gu, '\n\n')
      .trimEnd()
      .concat('\n')
  );
}

type StructTagOptions = NonNullable<Parameters<typeof generateStructTags>[2]>;

export class BackendGenerator {
  private readonly schema: ModuleSchema;
  private readonly packageName: string;
  private readonly modelName: string;

  constructor(schema: ModuleSchema) {
    const issues = validateGeneratorContract(schema);
    if (issues.length) throw new Error(issues.map((issue) => issue.messageKey).join(', '));
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

    // Model 名称
    const structName = scope === 'system' ? `System${this.modelName}` : this.modelName;

    // 字段定义：gofmt 把「主键 + 业务字段 + 审计字段」当成同一个连续列块对齐，
    // 所以三部分必须进同一次 alignGoBlock 调用，不能各自拼字符串。
    const auditRows: string[][] =
      templateLevel === 'enterprise'
        ? [
            ['CreatedAt', 'time.Time', '`json:"createdAt"`'],
            ['UpdatedAt', 'time.Time', '`json:"updatedAt"`'],
            ['DeletedAt', 'gorm.DeletedAt', '`gorm:"index" json:"-"`'],
          ]
        : [['CreatedAt', 'time.Time', '`json:"createdAt"`']];

    const tenantMode = isTenantTemplate(this.schema);
    const tenantIndexSuffix = tenantMode ? `;index:idx_${tableName}_tenant_id,priority:2` : '';
    const idColumnTag = `\`gorm:"primaryKey;autoIncrement${tenantIndexSuffix}" json:"id"\``;
    const tenantUniqueSuffix = tenantMode
      ? this.schema.model.fields
          .filter((field) => field.validation?.unique)
          .map(
            (field) =>
              `;uniqueIndex:uidx_${tableName}_tenant_${this.toDBColumn(field.name)},priority:1`,
          )
          .join('')
      : '';
    const tenantIdColumnTag = `\`gorm:"type:bigint unsigned;not null;default:0;index:idx_${tableName}_tenant_id,priority:1${tenantUniqueSuffix}" json:"-"\``;

    const structRows: string[][] = [
      ['ID', 'uint64', idColumnTag],
      ...(tenantMode ? [['TenantID', 'uint64', tenantIdColumnTag]] : []),
      ...this.generateModelRows(),
      ...auditRows,
    ];
    const fieldRows = alignGoBlock(structRows, '\t');

    // 导入：`getRequiredImports` 默认带上 gorm，但 basic 层级不写软删字段，
    // 未使用的 import 会被 Go 编译器判错（AUD-001b 编译验证暴露的既有缺陷），
    // 所以只有真正写出 gorm.DeletedAt 列时才保留该导入。
    const importBlock = buildGoImportBlock(
      Array.from(getRequiredImports(model.fields)).filter(
        (importPath) =>
          importPath !== '"gorm.io/gorm"' || structRows.some((row) => row[1] === 'gorm.DeletedAt'),
      ),
    );

    const junctionModels = this.generateManyToManyJunctionModels();

    return normalizeGoWhitespace(`package ${this.packageName}

import (
${importBlock}
)

// ${structName} ${this.schema.displayName}模型
type ${structName} struct {
${fieldRows}
}

// TableName 指定表名
func (${structName}) TableName() string {
\treturn "${tableName}"
}
${junctionModels ? `\n${junctionModels}` : ''}
`);
  }

  /**
   * 生成 model 字段行（列数组，交由 alignGoBlock 对齐）
   */
  private generateModelRows(): string[][] {
    return this.schema.model.fields
      .filter((f) => isTenantTemplate(this.schema) || f.visibleInForm !== false)
      .map((field) => {
        const mapping = TYPE_MAPPING[field.type];
        const options: StructTagOptions = {
          notNull: field.required,
          jsonName: field.name,
        };

        if (field.validation?.unique) {
          options.unique = !isTenantTemplate(this.schema);
        }

        let tags = generateStructTags(field.name, field.type, options);
        if (isTenantTemplate(this.schema) && field.validation?.unique) {
          tags = tags.replace(
            'gorm:"',
            `gorm:"uniqueIndex:uidx_${this.schema.model.tableName}_tenant_${this.toDBColumn(field.name)},priority:2;`,
          );
        }
        return [this.capitalize(field.name), mapping.go, `\`${tags}\``];
      });
  }

  /**
   * 生成 dto.go
   *
   * 参考: system/user/user_dto.go
   */
  generateDTO(): string {
    const { templateLevel = 'enterprise' } = this.schema;
    const structName = this.modelName;
    const importBlock = this.generateDTOImportBlock();
    const relationDTOs = this.generateManyToManyRelationDTOs();

    const listRows = [
      ['ID', 'uint64', '`json:"id"`'],
      ...this.generateDTORows('list'),
      ['CreatedAt', 'string', '`json:"createdAt"`'],
    ];
    // basic 层级的模型没有 UpdatedAt 列，DTO 也不能声明它，否则 service 里的
    // `item.UpdatedAt` 会编译不过（AUD-001b 编译验证暴露的既有缺陷）。
    const detailRows = [
      ['ID', 'uint64', '`json:"id"`'],
      ...this.generateDTORows('detail'),
      ['CreatedAt', 'string', '`json:"createdAt"`'],
      ...(templateLevel === 'enterprise' ? [['UpdatedAt', 'string', '`json:"updatedAt"`']] : []),
    ];
    const queryRows = [
      ...this.generateQueryRows(),
      ['Page', 'int', '`form:"page" json:"page"`'],
      ['PageSize', 'int', '`form:"pageSize" json:"pageSize"`'],
      ['SortField', 'string', '`form:"sortField" json:"sortField"`'],
      ['SortOrder', 'string', '`form:"sortOrder" json:"sortOrder"`'],
    ];
    const pageRows = [
      ['Items', `[]${structName}ListResp`, '`json:"items"`'],
      ['Total', 'int64', '`json:"total"`'],
      ['Page', 'int', '`json:"page"`'],
      ['PageSize', 'int', '`json:"pageSize"`'],
    ];
    const optionRows = [
      ['Label', 'string', '`json:"label"`'],
      ['Value', 'uint64', '`json:"value"`'],
      ['ID', 'uint64', '`json:"id"`'],
      ['Name', 'string', '`json:"name"`'],
    ];

    return normalizeGoWhitespace(`package ${this.packageName}

${importBlock}

// ${structName}ListResp 列表页返回 DTO
type ${structName}ListResp struct {
${alignGoBlock(listRows, '\t')}
}

// ${structName}ListPageResp 分页响应
type ${structName}ListPageResp struct {
${alignGoBlock(pageRows, '\t')}
}

// ${structName}DetailResp 详情返回 DTO
type ${structName}DetailResp struct {
${alignGoBlock(detailRows, '\t')}
}

// ${structName}ListQuery 查询参数
type ${structName}ListQuery struct {
${alignGoBlock(queryRows, '\t')}
}

// ${structName}CreateReq 创建请求
type ${structName}CreateReq struct {
${alignGoBlock(this.generateDTORows('create'), '\t')}
}

// ${structName}UpdateReq 更新请求
type ${structName}UpdateReq struct {
${alignGoBlock(this.generateDTORows('update'), '\t')}
}

type ${structName}OptionItem struct {
${alignGoBlock(optionRows, '\t')}
}
${relationDTOs ? `\n${relationDTOs}` : ''}
`);
  }

  private generateDTOImportBlock(): string {
    const dtoImports = new Set<string>();
    const usesTime = this.schema.model.fields.some(
      (field) => this.goTypeFromField(field.type) === 'time.Time',
    );
    if (usesTime) {
      dtoImports.add('time');
    }
    if (dtoImports.size === 0) {
      return '';
    }
    return `import (
${buildGoImportBlock(Array.from(dtoImports))}
)`;
  }

  /**
   * 生成 DTO 字段行（列数组，交由 alignGoBlock 对齐）
   */
  private generateDTORows(mode: 'list' | 'detail' | 'create' | 'update'): string[][] {
    return this.schema.model.fields
      .filter((f) => {
        if (mode === 'list') return f.visibleInList !== false;
        if (mode === 'create' || mode === 'update') return f.visibleInForm !== false;
        return true;
      })
      .map((field) => {
        const baseType = this.goTypeFromField(field.type);
        const isRequired = mode === 'create' && field.required;
        const isPointer = mode === 'update' || (isRequired && baseType === 'bool');
        const tsType = isPointer ? `*${baseType}` : baseType;
        const binding = isRequired ? ' binding:"required"' : '';
        return [this.capitalize(field.name), tsType, `\`json:"${field.name}"${binding}\``];
      });
  }

  /**
   * 生成查询字段行
   */
  private generateQueryRows(): string[][] {
    const searchableFields = this.schema.model.fields.filter((f) => f.searchable);

    if (searchableFields.length === 0) {
      // 注释行会打断 gofmt 的列块，这里单独成行，不参与对齐。
      return [['// 无搜索字段']];
    }

    const rows = searchableFields.map((field) => {
      const tsType = this.goTypeFromField(field.type);
      // 可搜索字段通常可选
      const isPointer = field.type === 'int' || field.type === 'float';
      const goType = isPointer ? `*${tsType}` : tsType;
      return [this.capitalize(field.name), goType, `\`form:"${field.name}" json:"${field.name}"\``];
    });
    // 统一关键词：跨所有文本型可搜索字段 OR LIKE
    if (this.getKeywordSearchColumns().length > 0) {
      rows.unshift(['Keyword', 'string', '`form:"keyword" json:"keyword"`']);
    }
    return rows;
  }

  /** 文本型可搜索字段的数据库列名，供统一关键词 OR LIKE。 */
  private getKeywordSearchColumns(): string[] {
    return this.schema.model.fields
      .filter((f) => f.searchable && f.type !== 'int' && f.type !== 'float' && f.type !== 'enum')
      .map((f) => this.toDBColumn(f.name));
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
    const tenantMode = isTenantTemplate(this.schema);
    const readDB = tenantMode ? 's.scopedDB()' : 's.db';
    const guard = (result = 'nil, '): string =>
      tenantMode
        ? `\tif err := s.requireTenant(); err != nil {\n\t\treturn ${result}err\n\t}\n`
        : '';
    // UpdatedAt 只存在于 enterprise 层级的模型里，DTO 与转换方法必须同步收敛，
    // 否则 basic 层级会引用不存在的列。
    const hasAudit = templateLevel === 'enterprise';
    const relationMigrations = this.generateManyToManyMigrations();
    const relationServices = this.generateManyToManyServiceMethods();
    const requiresStrconv =
      this.getManyToManyRelations().length > 0 || this.resolveOptionLabelField() === null;
    const importBlock = this.generateServiceImportBlock(tenantMode, hasDataScope, requiresStrconv);
    return normalizeGoWhitespace(`package ${this.packageName}

import (
${importBlock}
)

type ${modelName}Service struct {
${this.generateServiceStructFields(tenantMode, hasDataScope)}
}

// New${modelName}Service 构造函数
func New${modelName}Service(db *gorm.DB) *${modelName}Service {
\treturn &${modelName}Service{db: db}
}
${this.generateServiceTenantMethods(modelName, structName, tenantMode, hasDataScope)}

${this.generateServiceMigrate(modelName, structName, tenantMode, relationMigrations)}

// List${modelName}s 分页列表查询
func (s *${modelName}Service) List${modelName}s(query *${modelName}ListQuery${hasDataScope ? ', dataScope *common.DataScopeReq' : ''}) (*${modelName}ListPageResp, error) {
${guard()}\tif query == nil {
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

\tdb := ${readDB}.Model(&${structName}{})${hasDataScope ? `.Scopes(database.WithDataScope(dataScope))` : ``}
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
${guard()}\tvar items []${structName}
\tif err := ${readDB}.Model(&${structName}{}).Order("id desc").Limit(100).Find(&items).Error; err != nil {
\t\treturn nil, err
\t}
\toptions := make([]${modelName}OptionItem, len(items))
\tfor i, item := range items {
\t\toptions[i] = ${modelName}OptionItem{
${alignGoBlock(
  [
    ['Label:', `${this.generateOptionLabelExpression('item')},`],
    ['Value:', 'item.ID,'],
    ['ID:', 'item.ID,'],
    ['Name:', `${this.generateOptionNameExpression('item')},`],
  ],
  '\t\t\t',
)}
\t\t}
\t}
\treturn options, nil
}

// Get${modelName}Detail 详情查询
func (s *${modelName}Service) Get${modelName}Detail(id uint64) (*${modelName}DetailResp, error) {
${guard()}\tvar item ${structName}
\tif err := ${readDB}.First(&item, id).Error; err != nil {
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
${guard()}\titem := s.fromCreateReq(req)
${tenantMode ? '\titem.TenantID = s.TenantOwnerID()\n' : ''}\tif err := s.db.Create(&item).Error; err != nil {
\t\treturn nil, err
\t}
\tresp := s.toListResp(item)
\treturn &resp, nil
}

// Update${modelName} 更新
func (s *${modelName}Service) Update${modelName}(id uint64, req *${modelName}UpdateReq) (*${modelName}ListResp, error) {
${guard()}\tvar item ${structName}
\tif err := ${readDB}.First(&item, id).Error; err != nil {
\t\tif errors.Is(err, gorm.ErrRecordNotFound) {
\t\t\treturn nil, errors.New("${modelName.toLowerCase()}.not_found")
\t\t}
\t\treturn nil, err
\t}

\t// 更新字段
${this.generateUpdateFields()}

${
  tenantMode
    ? `\t// Updates never use Save's unscoped upsert fallback.
\tresult := s.scopedDB().Where("id = ?", id).Select("*").Omit("id", "tenant_id", "created_at").Updates(&item)
\tif result.Error != nil {
\t\treturn nil, result.Error
\t}
\tif result.RowsAffected == 0 {
\t\tif err := s.scopedDB().First(&item, id).Error; err != nil {
\t\t\treturn nil, err
\t\t}
\t}
`
    : `\tif err := s.db.Save(&item).Error; err != nil {
\t\treturn nil, err
\t}
`
}\tresp := s.toListResp(item)
\treturn &resp, nil
}

// Delete${modelName} 删除(软删除)
func (s *${modelName}Service) Delete${modelName}(id uint64) error {
${guard('')}\tresult := ${readDB}.Delete(&${structName}{}, id)
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
${alignGoBlock([['ID:', 'item.ID,'], ...this.toListRespRows(), ['CreatedAt:', 'item.CreatedAt.Format(time.RFC3339),']], '\t\t')}
\t}
}

func (s *${modelName}Service) toDetailResp(item ${structName}) ${modelName}DetailResp {
\treturn ${modelName}DetailResp{
${alignGoBlock([['ID:', 'item.ID,'], ...this.toDetailRespRows(), ['CreatedAt:', 'item.CreatedAt.Format(time.RFC3339),'], ...(hasAudit ? [['UpdatedAt:', 'item.UpdatedAt.Format(time.RFC3339),']] : [])], '\t\t')}
\t}
}

func (s *${modelName}Service) fromCreateReq(req *${modelName}CreateReq) ${structName} {
\treturn ${structName}{
${alignGoBlock(this.fromCreateReqRows(), '\t\t')}
\t}
}
${relationServices ? `\n${relationServices}` : ''}
`);
  }

  private generateServiceImportBlock(
    tenantMode: boolean,
    hasDataScope: boolean,
    requiresStrconv: boolean,
  ): string {
    const importPaths: Array<string | false> = ['errors', 'strings', 'time', 'gorm.io/gorm'];
    if (tenantMode) {
      importPaths.push('context', goImportPath('pkg/tenant'));
    }
    if (hasDataScope) {
      importPaths.push(goImportPath('pkg/common'), goImportPath('pkg/database'));
    }
    if (requiresStrconv) {
      importPaths.push('strconv');
    }
    return buildGoImportBlock(importPaths);
  }

  private generateServiceStructFields(tenantMode: boolean, hasDataScope: boolean): string {
    const rows: string[][] = [];
    if (tenantMode) {
      rows.push(['tenant.Scoper']);
      if (hasDataScope) {
        rows.push(['dataScope', '*common.DataScopeReq']);
      }
    }
    rows.push(['db', '*gorm.DB']);
    return alignGoBlock(rows, '\t');
  }

  private generateServiceTenantMethods(
    modelName: string,
    structName: string,
    tenantMode: boolean,
    hasDataScope: boolean,
  ): string {
    if (!tenantMode) {
      return '';
    }
    const dataScopeMethod = hasDataScope
      ? `
func (s *${modelName}Service) WithDataScope(scope *common.DataScopeReq) *${modelName}Service {
\tclone := *s
\tclone.dataScope = nil
\tif scope != nil {
\t\towned := *scope
\t\towned.DeptIDs = append([]uint64(nil), scope.DeptIDs...)
\t\towned.RoleKeys = append([]string(nil), scope.RoleKeys...)
\t\tclone.dataScope = &owned
\t}
\treturn &clone
}
`
      : '';
    const scopedDBDataScope = hasDataScope ? ', database.WithDataScope(s.dataScope)' : '';
    return `
// WithTenantContext owns a snapshot without mutating the shared service.
func (s *${modelName}Service) WithTenantContext(ctx *tenant.Context) *${modelName}Service {
\tclone := *s
\tclone.TenantCtx = nil
\tif ctx != nil {
\t\towned := *ctx
\t\tclone.TenantCtx = &owned
\t}
\treturn &clone
}

func (s *${modelName}Service) WithContext(ctx context.Context) *${modelName}Service {
\tclone := *s
\tif clone.db != nil {
\t\tclone.db = clone.db.WithContext(ctx)
\t}
\treturn &clone
}
${dataScopeMethod}
func (s *${modelName}Service) requireTenant() error {
\tif s.TenantCtx == nil {
\t\treturn tenant.ErrTenantContextMissing
\t}
\tif s.db == nil {
\t\treturn errors.New("database.not_initialized")
\t}
\treturn nil
}

func (s *${modelName}Service) scopedDB() *gorm.DB {
\treturn s.db.Model(&${structName}{}).Scopes(s.Scope()${scopedDBDataScope})
}
`;
  }

  private generateServiceMigrate(
    modelName: string,
    structName: string,
    tenantMode: boolean,
    relationMigrations: string,
  ): string {
    const tenantSchemaCheck = tenantMode
      ? `\t// Existing tables must already satisfy the opted-in tenant contract.
\tif s.db.Migrator().HasTable(&${structName}{}) {
\t\tif err := s.verifyTenantSchema(); err != nil {
\t\t\treturn err
\t\t}
\t}
`
      : '';
    const verifyMethod = tenantMode
      ? `
func (s *${modelName}Service) verifyTenantSchema() error {
${this.generateTenantSchemaVerification(structName)}\treturn nil
}
`
      : '';
    return `// Migrate 数据库迁移
func (s *${modelName}Service) Migrate() error {
\tif s.db == nil {
\t\treturn errors.New("database.not_initialized")
\t}
${tenantSchemaCheck}\tif err := s.db.AutoMigrate(&${structName}{}${relationMigrations}); err != nil {
\t\treturn err
\t}
\treturn ${tenantMode ? 's.verifyTenantSchema()' : 'nil'}
}
${verifyMethod}`;
  }

  /**
   * 生成查询过滤条件
   */
  private generateTenantSchemaVerification(structName: string): string {
    const table = this.schema.model.tableName;
    const uniqueFields = this.schema.model.fields.filter((field) => field.validation?.unique);
    const indexRows = [
      [`"idx_${table}_tenant_id":`, '{columns: []string{"tenant_id", "id"}},'],
      ...uniqueFields.map((field) => [
        `"uidx_${table}_tenant_${this.toDBColumn(field.name)}":`,
        `{columns: []string{"tenant_id", "${this.toDBColumn(field.name)}"}, unique: true},`,
      ]),
    ];
    return `\tcolumns, err := s.db.Migrator().ColumnTypes(&${structName}{})
\tif err != nil {
\t\treturn err
\t}
\tvalidTenant := false
\tfor _, column := range columns {
\t\tif column.Name() == "tenant_id" {
\t\t\tnullable, known := column.Nullable()
\t\t\tdefaultValue, hasDefault := column.DefaultValue()
\t\t\tcolumnType, hasType := column.ColumnType()
\t\t\tvalidTenant = known && !nullable && hasDefault && defaultValue == "0" && strings.EqualFold(column.DatabaseTypeName(), "bigint") && hasType && strings.Contains(strings.ToLower(columnType), "unsigned")
\t\t}
\t}
\tif !validTenant {
\t\treturn errors.New("module.generate.invalid_data_scope")
\t}
\tindexes, err := s.db.Migrator().GetIndexes(&${structName}{})
\tif err != nil {
\t\treturn err
\t}
\texpected := map[string]struct {
\t\tcolumns []string
\t\tunique  bool
\t}{
${alignGoBlock(indexRows, '\t\t')}
\t}
\tfor _, index := range indexes {
\t\tunique, known := index.Unique()
\t\tcolumns := index.Columns()
\t\tif want, ok := expected[index.Name()]; ok {
\t\t\tif !known || unique != want.unique || strings.Join(columns, ",") != strings.Join(want.columns, ",") {
\t\t\t\treturn errors.New("module.generate.invalid_data_scope")
\t\t\t}
\t\t\tdelete(expected, index.Name())
\t\t}
\t\tif unique && len(columns) > 0 && columns[0] != "tenant_id" && !(len(columns) == 1 && columns[0] == "id") {
\t\t\treturn errors.New("module.generate.invalid_data_scope")
\t\t}
\t}
\tif len(expected) != 0 {
\t\treturn errors.New("module.generate.invalid_data_scope")
\t}
`;
  }

  private generateQueryFilters(): string {
    const searchableFields = this.schema.model.fields.filter((f) => f.searchable);

    if (searchableFields.length === 0) {
      return '// 无搜索条件';
    }

    const filters = searchableFields.map((field) => {
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
    });
    // 统一关键词：跨所有文本型可搜索字段 OR LIKE
    const keywordColumns = this.getKeywordSearchColumns();
    if (keywordColumns.length > 0) {
      const clause = keywordColumns.map((column) => `${column} LIKE ?`).join(' OR ');
      const args = keywordColumns.map(() => 'keyword').join(', ');
      filters.unshift(`if strings.TrimSpace(query.Keyword) != "" {
\t\tkeyword := "%" + strings.TrimSpace(query.Keyword) + "%"
\t\tdb = db.Where("${clause}", ${args})
\t}`);
    }
    return filters.join('\n\t');
  }

  private toDBColumn(name: string): string {
    return name
      .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
      .replace(/[-\s]+/g, '_')
      .toLowerCase();
  }

  private generateSortFieldMap(): string {
    const sortableFields = this.schema.model.fields.filter((field) => field.sortable);
    const rows = sortableFields
      .map((field) => [`"${field.name}":`, `"${this.toDBColumn(field.name)}",`])
      .concat([
        ['"createdAt":', '"created_at",'],
        ['"updatedAt":', '"updated_at",'],
      ]);
    return alignGoBlock(rows, '\t\t');
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
   * 生成 toListResp 字段行
   */
  private toListRespRows(): string[][] {
    return this.schema.model.fields
      .filter((f) => f.visibleInList !== false)
      .map((field) => [`${this.capitalize(field.name)}:`, `item.${this.capitalize(field.name)},`]);
  }

  /**
   * 生成 toDetailResp 字段行
   */
  private toDetailRespRows(): string[][] {
    return this.schema.model.fields.map((field) => [
      `${this.capitalize(field.name)}:`,
      `item.${this.capitalize(field.name)},`,
    ]);
  }

  /**
   * 生成 fromCreateReq 字段行
   */
  private fromCreateReqRows(): string[][] {
    return this.schema.model.fields
      .filter((f) => f.visibleInForm !== false)
      .map((field) => {
        const dereference =
          field.required && this.goTypeFromField(field.type) === 'bool' ? '*' : '';
        return [
          `${this.capitalize(field.name)}:`,
          `${dereference}req.${this.capitalize(field.name)},`,
        ];
      });
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
    const tenantMode = isTenantTemplate(this.schema);
    const service = tenantMode ? 'service' : 'h.service';
    const dataScopeChain = hasDataScope ? '.WithDataScope(common.GetDataScope(c))' : '';
    const bindService = tenantMode
      ? `\tif tenant.FromGin(c) == nil {
\t\tcommon.Fail(c, common.CodeForbidden, "tenant.context.missing")
\t\treturn
\t}
\tservice := h.service.WithTenantContext(tenant.FromGin(c)).WithContext(c.Request.Context())${dataScopeChain}
`
      : '';
    const relationHandlers = this.generateManyToManyHandlerMethods();
    // 审计埋点是可选整行：换行必须一起放进条件里，否则 basic 模板会在 `{` 后
    // 留下一个空行，而 gofmt 会删掉块首空行——生成物因此不干净（AUD-001b）。
    const auditLine = (action: 'create' | 'update' | 'delete', businessType: string): string =>
      hasAudit
        ? `\tcommon.SetAuditMetadata(c, "${buildAuditActionKey(this.schema.scope, this.schema.name, action)}", ${businessType})\n`
        : '';

    return normalizeGoWhitespace(`package ${this.packageName}

import (
${buildGoImportBlock([goImportPath('pkg/common'), tenantMode && goImportPath('pkg/tenant'), 'strconv', 'github.com/gin-gonic/gin'])}
)

type ${modelName}Handler struct {
\tservice *${modelName}Service
}

func New${modelName}Handler(s *${modelName}Service) *${modelName}Handler {
\treturn &${modelName}Handler{service: s}
}

// Get${modelName}List 获取列表
func (h *${modelName}Handler) Get${modelName}List(c *gin.Context) {
${bindService}\tvar query ${modelName}ListQuery
\tif err := c.ShouldBindQuery(&query); err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

${hasDataScope ? '\tdataScope := common.GetDataScope(c)' : ''}
\tlist, err := ${service}.List${modelName}s(&query${hasDataScope ? ', dataScope' : ''})
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.list.error")
\t\treturn
\t}
\tcommon.Success(c, list)
}

// Get${modelName}Options 获取关系选项
func (h *${modelName}Handler) Get${modelName}Options(c *gin.Context) {
${bindService}\toptions, err := ${service}.List${modelName}Options()
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.options.error")
\t\treturn
\t}
\tcommon.Success(c, options)
}

// Get${modelName}Detail 获取详情
func (h *${modelName}Handler) Get${modelName}Detail(c *gin.Context) {
${bindService}\tid, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\tdetail, err := ${service}.Get${modelName}Detail(id)
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.detail.error")
\t\treturn
\t}
\tcommon.Success(c, detail)
}

// Create${modelName} 创建
func (h *${modelName}Handler) Create${modelName}(c *gin.Context) {
${bindService}${auditLine('create', 'common.BusinessInsert')}\tvar req ${modelName}CreateReq
\tif err := c.ShouldBindJSON(&req); err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\titem, err := ${service}.Create${modelName}(&req)
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.create.error")
\t\treturn
\t}
\tcommon.Success(c, item)
}

// Update${modelName} 更新
func (h *${modelName}Handler) Update${modelName}(c *gin.Context) {
${bindService}${auditLine('update', 'common.BusinessUpdate')}\tvar req ${modelName}UpdateReq
\tif err := c.ShouldBindJSON(&req); err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\tid, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\titem, err := ${service}.Update${modelName}(id, &req)
\tif err != nil {
\t\tcommon.Fail(c, common.CodeError, "${modelName.toLowerCase()}.update.error")
\t\treturn
\t}
\tcommon.Success(c, item)
}

// Delete${modelName} 删除
func (h *${modelName}Handler) Delete${modelName}(c *gin.Context) {
${bindService}${auditLine('delete', 'common.BusinessDelete')}\tid, err := parseUintParam(c, "id")
\tif err != nil {
\t\tcommon.Fail(c, common.CodeParamInvalid, "param.invalid")
\t\treturn
\t}

\tif err := ${service}.Delete${modelName}(id); err != nil {
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
`);
  }

  /**
   * 生成 module.go
   *
   * 参考: business/cmdb/module.go, auth/module.go
   */
  generateModule(): string {
    const modelName = this.modelName;
    const { scope, templateLevel = 'enterprise' } = this.schema;
    const hasDataScope = this.schema.enableDataScope ?? templateLevel === 'enterprise';
    const tenantMode = isTenantTemplate(this.schema);
    const moduleName = buildModuleNamespace(scope, this.schema.name);
    const pageTitleKey = buildTitleKey(scope, this.schema.name);
    const componentKey = buildComponentKey(scope, this.schema.name, modelName);
    const routePath = buildRoutePath(scope, this.schema.name);
    const relationRoutes = this.generateManyToManyRouteRegistrations();
    const menuSeedEntries = this.generateMenuSeedEntries(componentKey, pageTitleKey);
    const i18nSeedEntries = this.generateI18nSeedEntries(pageTitleKey);
    // 种子数组可能整体为空（关系表不生成导航）。空数组必须写成 `[]T{}`：
    // `[]T{\n\n}` 是 gofmt 会立刻压掉的一个空行，留给 normalizeGoWhitespace 也救不回来。
    const seedList = (typeName: string, entries: string): string =>
      `var generated${typeName}Seeds = []generated${typeName}Seed{${
        entries ? `\n${entries}\n` : ''
      }}`;
    const protectedGroupCall = hasDataScope
      ? `contracts.DataScopedGroup(r, "${routePath}", db)`
      : `contracts.ProtectedGroup(r, "${routePath}")`;
    const routeProtection = tenantMode ? '' : `\t\t\tprotected := ${protectedGroupCall}\n`;

    return normalizeGoWhitespace(`package ${this.packageName}

import (
${buildGoImportBlock([goImportPath('pkg/contracts'), tenantMode && goImportPath('pkg/database'), 'strings', 'github.com/gin-gonic/gin', 'gorm.io/gorm'])}
)

${[
  'type generatedMenuSeed struct {',
  alignGoBlock(
    [
      ['Key', 'string'],
      ['ParentKey', 'string'],
      ['ParentPath', 'string'],
      ['TitleKey', 'string'],
      ['Path', 'string'],
      ['Component', 'string'],
      ['PagePerm', 'string'],
      ['Perms', 'string'],
      ['Type', 'string'],
      ['Icon', 'string'],
      ['RouteName', 'string'],
      ['Module', 'string'],
      ['Sort', 'int'],
    ],
    '\t',
  ),
  '}',
].join('\n')}

${[
  'type generatedI18nSeed struct {',
  alignGoBlock(
    [
      ['Module', 'string'],
      ['Locale', 'string'],
      ['Group', 'string'],
      ['Key', 'string'],
      ['Value', 'string'],
    ],
    '\t',
  ),
  '}',
].join('\n')}

${seedList('Menu', menuSeedEntries)}

${seedList('I18n', i18nSeedEntries)}

func Init${modelName}Module(r *gin.RouterGroup, db *gorm.DB) {
\tservice := New${modelName}Service(db)
\thandler := New${modelName}Handler(service)

${
  tenantMode
    ? `\tvar protected gin.IRoutes
\tbootstrap := func(_ *gorm.DB) error {
\t\tif err := service.Migrate(); err != nil {
\t\t\treturn err
\t\t}
\t\troutes, err := contracts.TenantScopedBusinessGroup(r, "${routePath}", contracts.BusinessTenantGroupOptions{
\t\t\tDB: db, RDB: database.RDB, DataScope: ${hasDataScope},
\t\t})
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\tprotected = routes
\t\treturn nil
\t}

\tif err := contracts.RegisterRequiredBackendModules(r, db, contracts.FuncModule{`
    : '\tcontracts.RegisterBackendModules(r, db, contracts.FuncModule{'
}
${alignGoBlock(
  [
    ['ModuleName:', `"${moduleName}",`],
    [
      'MigrateFunc:',
      tenantMode ? 'bootstrap,' : 'func(_ *gorm.DB) error { return service.Migrate() },',
    ],
    [
      'BootstrapFunc:',
      tenantMode ? 'bootstrap,' : 'func(_ *gorm.DB) error { return service.Migrate() },',
    ],
    ['SeedMenusFunc:', `seed${modelName}Menus,`],
    ['SeedI18nFunc:', `seed${modelName}I18n,`],
  ],
  '\t\t',
)}
\t\tRegister: func(r *gin.RouterGroup) {
${routeProtection}\t\t\t{
\t\t\t\tprotected.GET("/list", handler.Get${modelName}List)
\t\t\t\tprotected.GET("/options", handler.Get${modelName}Options)
\t\t\t\tprotected.GET("/:id", handler.Get${modelName}Detail)
${relationRoutes ? `\t\t\t\t${relationRoutes}\n` : ''}\t\t\t\tprotected.POST("", handler.Create${modelName})
\t\t\t\tprotected.PUT("/:id", handler.Update${modelName})
\t\t\t\tprotected.DELETE("/:id", handler.Delete${modelName})
\t\t\t}
\t\t},
${tenantMode ? '\t}); err != nil {\n\t\tpanic(err)\n\t}' : '\t})'}
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
`);
  }

  private generateMenuSeedEntries(componentKey: string, pageTitleKey: string): string {
    if (!shouldGenerateNavigation(this.schema)) {
      return '';
    }
    const moduleKey = buildModuleNamespace(this.schema.scope, this.schema.name);
    const permissionPrefix = buildPermissionPrefix(this.schema.scope, this.schema.name);
    const routePath = buildPageRoutePath(this.schema.scope, this.schema.name);
    const routeName = buildRouteName(this.schema.scope, this.schema.name);
    const segments = splitModuleSegments(this.schema.name);
    const menuKey = segments.join('-');
    const explicitParentPath = normalizeMenuPath(this.schema.parentMenu || '');
    const shouldGenerateAncestorMenus = !explicitParentPath && segments.length > 1;
    let inferredParentPath = '';
    if (!shouldGenerateAncestorMenus && segments.length > 1) {
      inferredParentPath = buildPageRoutePath(this.schema.scope, segments.slice(0, -1).join('/'));
    }
    const parentPath = normalizeMenuPath(explicitParentPath || inferredParentPath || '');
    const parentKey = shouldGenerateAncestorMenus ? segments.slice(0, -1).join('-') : '';
    const actionSeeds = getPageActions(this.schema)
      .filter((action) => action !== 'detail')
      .map((action) => ({
        action,
        title: this.renderActionTitle(action, 'zh'),
        titleEn: this.renderActionTitle(action, 'en'),
      }));

    // 每个菜单种子是一个独立的 gofmt 列块（`{` 与 `},` 都会断块），所以逐条调用
    // alignGoBlock；此前只有 mainSeed 多出的 ParentPath 参与列宽，写死的空格
    // 就把它的兄弟列全部算短了（AUD-001b）。
    const renderSeedEntry = (rows: Array<[string, string]>): string =>
      `\t{\n${alignGoBlock(
        rows.map(([key, value]) => [key, `${value},`]),
        '\t\t',
      )}\n\t},`;

    const ancestorEntries = shouldGenerateAncestorMenus
      ? segments.slice(0, -1).map((_, index) => {
          const groupSegments = segments.slice(0, index + 1);
          const groupKey = groupSegments.join('-');
          const parentSegments = groupSegments.slice(0, -1);
          const groupModuleKey = `${this.schema.scope}.${groupSegments.join('.')}`;
          const groupTitleKey = buildMenuGroupTitleKey(this.schema.scope, groupSegments);
          return renderSeedEntry([
            ['Key:', `"${groupKey}"`],
            ['ParentKey:', `"${parentSegments.join('-')}"`],
            ['TitleKey:', `"${groupTitleKey}"`],
            ['Path:', `"${buildPageRoutePath(this.schema.scope, groupSegments.join('/'))}"`],
            ['Type:', '"M"'],
            ['Icon:', '"apps"'],
            ['RouteName:', `"${this.schema.scope}-${groupSegments.join('-')}"`],
            ['Module:', `"${groupModuleKey}"`],
            ['Sort:', '10'],
          ]);
        })
      : [];

    const mainSeed = renderSeedEntry([
      ['Key:', `"${menuKey}"`],
      ['ParentKey:', `"${parentKey}"`],
      ['ParentPath:', `"${parentPath}"`],
      ['TitleKey:', `"${pageTitleKey}"`],
      ['Path:', `"${routePath}"`],
      ['Component:', `"${componentKey}"`],
      ['PagePerm:', `"${permissionPrefix}:list"`],
      ['Type:', '"C"'],
      ['Icon:', '"apps"'],
      ['RouteName:', `"${routeName}"`],
      ['Module:', `"${moduleKey}"`],
      ['Sort:', '10'],
    ]);

    const actionEntries = actionSeeds.map((item, index) =>
      renderSeedEntry([
        ['Key:', `"${menuKey}-${item.action}"`],
        ['ParentKey:', `"${menuKey}"`],
        [
          'TitleKey:',
          `"${buildPermissionTitleKey(this.schema.scope, this.schema.name, item.action)}"`,
        ],
        ['Perms:', `"${permissionPrefix}:${item.action}"`],
        ['Type:', '"F"'],
        ['Module:', `"${moduleKey}"`],
        ['Sort:', `${index + 1}`],
      ]),
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
      const fallback = inferMenuGroupDisplayName(groupSegments.at(-1) ?? '');
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
        const targetColumn = this.getManyToManyTargetColumn(
          relation.targetModule,
          relation.targetField,
        );
        const indexName = `idx_${this.toDBColumn(this.modelName)}_${this.toDBColumn(relation.name)}_rel`;
        return `type ${structName} struct {
${alignGoBlock(
  [
    [
      'OwnerID',
      'uint64',
      `\`gorm:"column:${ownerColumn};not null;uniqueIndex:${indexName},priority:1" json:"ownerId"\``,
    ],
    [
      'TargetID',
      'uint64',
      `\`gorm:"column:${targetColumn};not null;uniqueIndex:${indexName},priority:2" json:"targetId"\``,
    ],
  ],
  '\t',
)}
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
${alignGoBlock(
  [
    ['ID', 'uint64', '`json:"id"`'],
    ['Value', 'uint64', '`json:"value"`'],
  ],
  '\t',
)}
}

type ${relationName}RelationListResp struct {
${alignGoBlock([['Items', `[]${relationName}RelationRow`, '`json:"items"`']], '\t')}
}

type ${relationName}RelationBindReq struct {
${alignGoBlock([['TargetIDs', '[]interface{}', '`json:"targetIds" binding:"required"`']], '\t')}
}`;
      })
      .join('\n\n');
  }

  private generateManyToManyMigrations(): string {
    const relations = this.getManyToManyRelations();
    if (relations.length === 0) {
      return '';
    }
    return relations
      .map((relation) => `, &${this.getManyToManyStructName(relation.name)}{}`)
      .join('');
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
        const targetColumn = this.getManyToManyTargetColumn(
          relation.targetModule,
          relation.targetField,
        );
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
${alignGoBlock(
  [
    ['ID:', 'row.Value,'],
    ['Value:', 'row.Value,'],
  ],
  '\t\t\t',
)}
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
${alignGoBlock(
  [
    [`"${ownerColumn}":`, 'ownerID,'],
    [`"${targetColumn}":`, 'targetID,'],
  ],
  '\t\t\t\t',
)}
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
    return this.generateOptionLabelExpression(sourceVar);
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
    return TYPE_MAPPING[fieldType as keyof typeof TYPE_MAPPING]?.go || 'string';
  }

  private escapeGoString(value: string): string {
    return value.replaceAll('\\', String.raw`\\`).replaceAll('"', String.raw`\"`);
  }

  private toPascalCase(value: string): string {
    return value
      .split(/[_-]/)
      .filter(Boolean)
      .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
      .join('');
  }
}
