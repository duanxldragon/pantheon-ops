/**
 * 模块生成器 - 模块描述 Schema 定义
 *
 * 基于项目真实结构约定:
 * - system/* 使用 package system
 * - business/* 使用 package {module}
 * - 支持基础/企业级两种模板级别
 */

export type ModuleScope = 'system' | 'business';

export type TemplateLevel = 'basic' | 'enterprise';

export type FieldType = 'string' | 'text' | 'int' | 'float' | 'bool' | 'date' | 'enum' | 'relation';

export type PageActionKey =
  | 'view'
  | 'create'
  | 'update'
  | 'delete'
  | 'export'
  | 'import'
  | 'detail';

export type PageActionTemplate = 'standard' | 'masterData' | 'lookup';

export type BusinessTableRole = 'main' | 'detail' | 'relation' | 'dictionary';

export type GeneratorTemplateVersion = 'v1';

export type DataScopeMode = 'none' | 'owner' | 'dept' | 'tenant' | 'custom';

export type ModuleRelationType = 'oneToMany' | 'manyToMany' | 'lookup';

export type FieldTemplateKey =
  | 'none'
  | 'code'
  | 'name'
  | 'status'
  | 'sort'
  | 'remark'
  | 'owner'
  | 'phone'
  | 'email'
  | 'ipAddress'
  | 'hostname'
  | 'environment';

export interface EnumOption {
  value: string;
  label: string;
  labelEn?: string;
  color?: string;
}

export interface FieldValidation {
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  min?: number;
  max?: number;
  pattern?: string;
  enum?: string[];
  unique?: boolean;
}

export interface ModuleField {
  name: string;
  type: FieldType;
  label: string;
  labelEn?: string;
  required?: boolean;
  searchable?: boolean;
  sortable?: boolean;
  visibleInList?: boolean;
  visibleInForm?: boolean;
  defaultValue?: string;
  validation?: FieldValidation;
  placeholder?: string;
  placeholderEn?: string;
  helpText?: string;
  helpTextEn?: string;
  templateKey?: FieldTemplateKey;
  dictCode?: string;
  enumOptions?: EnumOption[];
}

export type MenuType = 'M' | 'C' | 'F';

export interface MenuSeedConfig {
  key: string;
  parentKey?: string;
  titleKey: string;
  path?: string;
  component?: string;
  pagePermission?: string;
  perms?: string;
  type: MenuType;
  icon?: string;
  routeName?: string;
  module?: string;
  sort?: number;
  isCache?: boolean;
  isExternal?: boolean;
  activeMenu?: string;
}

export interface PermissionConfig {
  key: string;
  name: string;
  type: 'menu' | 'button';
  module: string;
}

export interface I18nConfig {
  namespace: string;
  translations: {
    zh: Record<string, string>;
    en: Record<string, string>;
  };
}

export interface ModuleMetadata {
  businessContext?: string;
  businessContextTitle?: string;
  businessContextTitleEn?: string;
  tableRole?: BusinessTableRole;
  primaryTable?: string;
  relationFromField?: string;
  relationToField?: string;
  boundedContext?: string;
  owner?: string;
  summary?: string;
  sourceMode?: 'manual' | 'database';
  sourceDatasourceId?: string;
  sourceDatasourceName?: string;
  sourceTable?: string;
  autoRecycle?: boolean;
}

export interface ModuleListLayoutConfig {
  governance?: boolean;
  search?: boolean;
  headerActions?: boolean;
  batchActions?: boolean;
  rowActions?: boolean;
}

export interface ModuleDependency {
  module: string;
  required?: boolean;
  reason?: string;
}

export interface ModuleRelation {
  name: string;
  type: ModuleRelationType;
  targetModule: string;
  localField: string;
  targetField: string;
  targetLabelField?: string;
  lookupApi?: string;
  lookupValueField?: string;
  junctionTable?: string;
}

export interface ModuleSchema {
  name: string;
  templateVersion?: GeneratorTemplateVersion;
  displayName: string;
  displayNameEn?: string;
  description?: string;
  scope: ModuleScope;
  parentMenu?: string;
  templateLevel?: TemplateLevel;
  pageActionTemplate?: PageActionTemplate;
  pageActions?: PageActionKey[];
  dependencies?: ModuleDependency[];
  relations?: ModuleRelation[];
  dataScopeMode?: DataScopeMode;
  listLayout?: ModuleListLayoutConfig;
  metadata?: ModuleMetadata;
  model: {
    tableName: string;
    modelName?: string;
    packageName?: string;
    fields: ModuleField[];
  };
  menus: MenuSeedConfig[];
  permissions: PermissionConfig[];
  i18n: I18nConfig;
  enableExport?: boolean;
  enableImport?: boolean;
  enableAudit?: boolean;
  enableDataScope?: boolean;
  includeDashboardWidget?: boolean;
}

export interface FieldTemplateDefinition {
  key: FieldTemplateKey;
  labelKey: string;
  createField: (resolveText?: TemplateTextResolver) => ModuleField;
}

export interface PageActionTemplateDefinition {
  key: PageActionTemplate;
  labelKey: string;
  descriptionKey: string;
  actions: PageActionKey[];
}

export interface GeneratorCompletenessIssue {
  code: string;
  level: 'error' | 'warn';
  messageKey: string;
  detail?: string;
}

export interface GeneratorMenuPreviewNode {
  key: string;
  titleKey: string;
  path?: string;
  type: MenuType;
  module?: string;
  children: GeneratorMenuPreviewNode[];
}

type TemplateLocale = 'zh-CN' | 'en-US';
type TemplateTextResolver = (locale: TemplateLocale, key: string, fallback: string) => string;

function resolveTemplateText(
  resolveText: TemplateTextResolver | undefined,
  locale: TemplateLocale,
  key: string,
  fallback: string,
) {
  return resolveText ? resolveText(locale, key, fallback) : fallback;
}

export const FIELD_TEMPLATE_DEFINITIONS: FieldTemplateDefinition[] = [
  {
    key: 'code',
    labelKey: 'generator.fieldTemplates.code',
    createField: (resolveText) => ({
      name: 'code',
      type: 'string',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.code',
        'generator.fieldTemplates.code',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.code',
        'generator.fieldTemplates.code',
      ),
      required: true,
      searchable: true,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.code.placeholder',
        'generator.fieldTemplates.code.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.code.placeholder',
        'generator.fieldTemplates.code.placeholder',
      ),
      validation: { unique: true, maxLength: 64 },
      templateKey: 'code',
    }),
  },
  {
    key: 'name',
    labelKey: 'generator.fieldTemplates.name',
    createField: (resolveText) => ({
      name: 'name',
      type: 'string',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.name',
        'generator.fieldTemplates.name',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.name',
        'generator.fieldTemplates.name',
      ),
      required: true,
      searchable: true,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.name.placeholder',
        'generator.fieldTemplates.name.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.name.placeholder',
        'generator.fieldTemplates.name.placeholder',
      ),
      validation: { maxLength: 128 },
      templateKey: 'name',
    }),
  },
  {
    key: 'status',
    labelKey: 'generator.fieldTemplates.status',
    createField: (resolveText) => ({
      name: 'status',
      type: 'enum',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.status',
        'generator.fieldTemplates.status',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.status',
        'generator.fieldTemplates.status',
      ),
      required: true,
      searchable: true,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.status.placeholder',
        'generator.fieldTemplates.status.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.status.placeholder',
        'generator.fieldTemplates.status.placeholder',
      ),
      dictCode: 'status',
      enumOptions: [
        {
          value: 'active',
          label: resolveTemplateText(
            resolveText,
            'zh-CN',
            'generator.fieldTemplates.status.active',
            'generator.fieldTemplates.status.active',
          ),
          labelEn: resolveTemplateText(
            resolveText,
            'en-US',
            'generator.fieldTemplates.status.active',
            'generator.fieldTemplates.status.active',
          ),
          color: 'green',
        },
        {
          value: 'inactive',
          label: resolveTemplateText(
            resolveText,
            'zh-CN',
            'generator.fieldTemplates.status.inactive',
            'generator.fieldTemplates.status.inactive',
          ),
          labelEn: resolveTemplateText(
            resolveText,
            'en-US',
            'generator.fieldTemplates.status.inactive',
            'generator.fieldTemplates.status.inactive',
          ),
          color: 'gray',
        },
      ],
      validation: { enum: ['active', 'inactive'] },
      templateKey: 'status',
    }),
  },
  {
    key: 'sort',
    labelKey: 'generator.fieldTemplates.sort',
    createField: (resolveText) => ({
      name: 'sort',
      type: 'int',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.sort',
        'generator.fieldTemplates.sort',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.sort',
        'generator.fieldTemplates.sort',
      ),
      required: false,
      searchable: false,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.sort.placeholder',
        'generator.fieldTemplates.sort.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.sort.placeholder',
        'generator.fieldTemplates.sort.placeholder',
      ),
      templateKey: 'sort',
    }),
  },
  {
    key: 'remark',
    labelKey: 'generator.fieldTemplates.remark',
    createField: (resolveText) => ({
      name: 'remark',
      type: 'text',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.remark',
        'generator.fieldTemplates.remark',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.remark',
        'generator.fieldTemplates.remark',
      ),
      required: false,
      searchable: false,
      sortable: false,
      visibleInList: false,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.remark.placeholder',
        'generator.fieldTemplates.remark.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.remark.placeholder',
        'generator.fieldTemplates.remark.placeholder',
      ),
      templateKey: 'remark',
    }),
  },
  {
    key: 'owner',
    labelKey: 'generator.fieldTemplates.owner',
    createField: (resolveText) => ({
      name: 'owner',
      type: 'string',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.owner',
        'generator.fieldTemplates.owner',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.owner',
        'generator.fieldTemplates.owner',
      ),
      required: false,
      searchable: true,
      sortable: false,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.owner.placeholder',
        'generator.fieldTemplates.owner.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.owner.placeholder',
        'generator.fieldTemplates.owner.placeholder',
      ),
      templateKey: 'owner',
    }),
  },
  {
    key: 'phone',
    labelKey: 'generator.fieldTemplates.phone',
    createField: (resolveText) => ({
      name: 'phone',
      type: 'string',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.phone',
        'generator.fieldTemplates.phone',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.phone',
        'generator.fieldTemplates.phone',
      ),
      required: false,
      searchable: true,
      sortable: false,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.phone.placeholder',
        'generator.fieldTemplates.phone.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.phone.placeholder',
        'generator.fieldTemplates.phone.placeholder',
      ),
      validation: { pattern: '^1[3-9]\\d{9}$' },
      templateKey: 'phone',
    }),
  },
  {
    key: 'email',
    labelKey: 'generator.fieldTemplates.email',
    createField: (resolveText) => ({
      name: 'email',
      type: 'string',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.email',
        'generator.fieldTemplates.email',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.email',
        'generator.fieldTemplates.email',
      ),
      required: false,
      searchable: true,
      sortable: false,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.email.placeholder',
        'generator.fieldTemplates.email.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.email.placeholder',
        'generator.fieldTemplates.email.placeholder',
      ),
      validation: { pattern: '^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$' },
      templateKey: 'email',
    }),
  },
  {
    key: 'ipAddress',
    labelKey: 'generator.fieldTemplates.ipAddress',
    createField: (resolveText) => ({
      name: 'ipAddress',
      type: 'string',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.ipAddress',
        'generator.fieldTemplates.ipAddress',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.ipAddress',
        'generator.fieldTemplates.ipAddress',
      ),
      required: true,
      searchable: true,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.ipAddress.placeholder',
        'generator.fieldTemplates.ipAddress.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.ipAddress.placeholder',
        'generator.fieldTemplates.ipAddress.placeholder',
      ),
      validation: { unique: true },
      templateKey: 'ipAddress',
    }),
  },
  {
    key: 'hostname',
    labelKey: 'generator.fieldTemplates.hostname',
    createField: (resolveText) => ({
      name: 'hostname',
      type: 'string',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.hostname',
        'generator.fieldTemplates.hostname',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.hostname',
        'generator.fieldTemplates.hostname',
      ),
      required: true,
      searchable: true,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.hostname.placeholder',
        'generator.fieldTemplates.hostname.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.hostname.placeholder',
        'generator.fieldTemplates.hostname.placeholder',
      ),
      templateKey: 'hostname',
    }),
  },
  {
    key: 'environment',
    labelKey: 'generator.fieldTemplates.environment',
    createField: (resolveText) => ({
      name: 'environment',
      type: 'enum',
      label: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.environment',
        'generator.fieldTemplates.environment',
      ),
      labelEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.environment',
        'generator.fieldTemplates.environment',
      ),
      required: true,
      searchable: true,
      sortable: false,
      visibleInList: true,
      visibleInForm: true,
      placeholder: resolveTemplateText(
        resolveText,
        'zh-CN',
        'generator.fieldTemplates.environment.placeholder',
        'generator.fieldTemplates.environment.placeholder',
      ),
      placeholderEn: resolveTemplateText(
        resolveText,
        'en-US',
        'generator.fieldTemplates.environment.placeholder',
        'generator.fieldTemplates.environment.placeholder',
      ),
      dictCode: 'environment',
      enumOptions: [
        {
          value: 'dev',
          label: resolveTemplateText(
            resolveText,
            'zh-CN',
            'generator.fieldTemplates.environment.dev',
            'generator.fieldTemplates.environment.dev',
          ),
          labelEn: resolveTemplateText(
            resolveText,
            'en-US',
            'generator.fieldTemplates.environment.dev',
            'generator.fieldTemplates.environment.dev',
          ),
          color: 'arcoblue',
        },
        {
          value: 'test',
          label: resolveTemplateText(
            resolveText,
            'zh-CN',
            'generator.fieldTemplates.environment.test',
            'generator.fieldTemplates.environment.test',
          ),
          labelEn: resolveTemplateText(
            resolveText,
            'en-US',
            'generator.fieldTemplates.environment.test',
            'generator.fieldTemplates.environment.test',
          ),
          color: 'purple',
        },
        {
          value: 'staging',
          label: resolveTemplateText(
            resolveText,
            'zh-CN',
            'generator.fieldTemplates.environment.staging',
            'generator.fieldTemplates.environment.staging',
          ),
          labelEn: resolveTemplateText(
            resolveText,
            'en-US',
            'generator.fieldTemplates.environment.staging',
            'generator.fieldTemplates.environment.staging',
          ),
          color: 'orange',
        },
        {
          value: 'prod',
          label: resolveTemplateText(
            resolveText,
            'zh-CN',
            'generator.fieldTemplates.environment.prod',
            'generator.fieldTemplates.environment.prod',
          ),
          labelEn: resolveTemplateText(
            resolveText,
            'en-US',
            'generator.fieldTemplates.environment.prod',
            'generator.fieldTemplates.environment.prod',
          ),
          color: 'red',
        },
      ],
      validation: { enum: ['dev', 'test', 'staging', 'prod'] },
      templateKey: 'environment',
    }),
  },
];

export const PAGE_ACTION_TEMPLATE_DEFINITIONS: PageActionTemplateDefinition[] = [
  {
    key: 'standard',
    labelKey: 'generator.actionTemplates.standard',
    descriptionKey: 'generator.actionTemplates.standard.desc',
    actions: ['view', 'create', 'update', 'delete', 'detail'],
  },
  {
    key: 'masterData',
    labelKey: 'generator.actionTemplates.masterData',
    descriptionKey: 'generator.actionTemplates.masterData.desc',
    actions: ['view', 'create', 'update', 'delete', 'export', 'import'],
  },
  {
    key: 'lookup',
    labelKey: 'generator.actionTemplates.lookup',
    descriptionKey: 'generator.actionTemplates.lookup.desc',
    actions: ['view', 'detail'],
  },
];

export function splitModuleSegments(name: string): string[] {
  return name
    .replace(/\\/g, '/')
    .split('/')
    .map((segment) => segment.trim())
    .filter(Boolean);
}

export function normalizeModulePath(value?: string): string {
  return splitModuleSegments(String(value || '')).join('/');
}

export function normalizeBusinessContext(value?: string): string {
  const segments = splitModuleSegments(String(value || ''));
  return segments[0] || '';
}

export function isValidModulePath(value?: string, allowNested = true): boolean {
  const segments = splitModuleSegments(String(value || ''));
  if (segments.length === 0) {
    return false;
  }
  if (!allowNested && segments.length !== 1) {
    return false;
  }
  return segments.every((segment) => /^[a-z][a-z0-9_]*$/.test(segment));
}

export function isValidScopedModulePath(scope: ModuleScope, value?: string): boolean {
  return isValidModulePath(value, scope === 'business');
}

export function getLeafModuleName(name: string): string {
  const segments = splitModuleSegments(name);
  return segments[segments.length - 1] || name;
}

function toPascalCase(value: string): string {
  return value
    .split(/[/_-]/)
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join('');
}

export function buildModuleNamespace(scope: ModuleScope, name: string): string {
  return `${scope}.${splitModuleSegments(name).join('.')}`;
}

export function buildPermissionPrefix(scope: ModuleScope, name: string): string {
  return `${scope}:${splitModuleSegments(name).join(':')}`;
}

export function buildRoutePath(scope: ModuleScope, name: string): string {
  return `/${scope}/${splitModuleSegments(name).join('/')}`;
}

export function buildPageRoutePath(scope: ModuleScope, name: string): string {
  const segments = splitModuleSegments(name);
  if (scope === 'business') {
    return `/operations/${segments.join('/')}`;
  }
  return buildRoutePath(scope, name);
}

export function normalizeMenuPath(value?: string): string {
  const normalized = String(value || '')
    .trim()
    .replace(/\\/g, '/');
  if (!normalized) {
    return '';
  }
  const stripped = normalized.replace(/^\/+/, '');
  return `/${stripped}`;
}

export function buildRouteName(scope: ModuleScope, name: string): string {
  return `${scope}-${splitModuleSegments(name).join('-')}`;
}

export function buildTitleKey(scope: ModuleScope, name: string): string {
  return `${buildModuleNamespace(scope, name)}.title`;
}

export function buildDashboardQuickActionDescriptionKey(scope: ModuleScope, name: string): string {
  return `${buildModuleNamespace(scope, name)}.dashboard.quickAction`;
}

export function buildMenuGroupTitleKey(scope: ModuleScope, segments: string[]): string {
  return `${scope}.${segments.join('.')}.title`;
}

export function inferMenuGroupDisplayName(segment: string): string {
  const normalized = segment.trim();
  if (!normalized) {
    return '';
  }
  if (normalized.length <= 4) {
    return normalized.toUpperCase();
  }
  return normalized
    .split(/[_-]/)
    .filter(Boolean)
    .map((item) => item.charAt(0).toUpperCase() + item.slice(1))
    .join(' ');
}

export function inferBusinessContextFromName(name: string): string {
  return splitModuleSegments(name)[0] || '';
}

export function buildFieldLabelKey(scope: ModuleScope, name: string, fieldName: string): string {
  return `${buildModuleNamespace(scope, name)}.field.${fieldName}.label`;
}

export function buildFieldPlaceholderKey(
  scope: ModuleScope,
  name: string,
  fieldName: string,
): string {
  return `${buildModuleNamespace(scope, name)}.field.${fieldName}.placeholder`;
}

export function buildFieldHelpTextKey(scope: ModuleScope, name: string, fieldName: string): string {
  return `${buildModuleNamespace(scope, name)}.field.${fieldName}.helpText`;
}

export function buildEnumOptionKey(
  scope: ModuleScope,
  name: string,
  fieldName: string,
  optionValue: string,
): string {
  return `${buildModuleNamespace(scope, name)}.field.${fieldName}.option.${optionValue}`;
}

export function buildPermissionTitleKey(
  scope: ModuleScope,
  name: string,
  action: PageActionKey,
): string {
  return `${buildModuleNamespace(scope, name)}.permission.${action}`;
}

export function buildAuditActionKey(
  scope: ModuleScope,
  name: string,
  action: 'create' | 'update' | 'delete',
): string {
  return `${buildModuleNamespace(scope, name)}.audit.${action}`;
}

export function buildComponentKey(scope: ModuleScope, name: string, modelName: string): string {
  return `${scope}/${splitModuleSegments(name).join('/')}/${modelName}List`;
}

export function inferPackageName(schema: ModuleSchema): string {
  if (schema.model.packageName) {
    return schema.model.packageName;
  }
  if (schema.scope === 'system') {
    return 'system';
  }
  return getLeafModuleName(schema.name);
}

export function inferModelName(schema: ModuleSchema): string {
  if (schema.model.modelName) {
    return schema.model.modelName;
  }
  return toPascalCase(schema.name);
}

export function getPageActions(
  schema: Pick<
    ModuleSchema,
    'pageActions' | 'pageActionTemplate' | 'enableExport' | 'enableImport'
  >,
): PageActionKey[] {
  if (schema.pageActions && schema.pageActions.length > 0) {
    return Array.from(new Set(schema.pageActions));
  }
  const template =
    PAGE_ACTION_TEMPLATE_DEFINITIONS.find((item) => item.key === schema.pageActionTemplate) ??
    PAGE_ACTION_TEMPLATE_DEFINITIONS[0];
  const actions = [...template.actions];
  if (schema.enableExport && !actions.includes('export')) {
    actions.push('export');
  }
  if (schema.enableImport && !actions.includes('import')) {
    actions.push('import');
  }
  return actions;
}

export function applyFieldTemplate(
  templateKey: FieldTemplateKey,
  resolveText?: TemplateTextResolver,
): ModuleField | null {
  const template = FIELD_TEMPLATE_DEFINITIONS.find((item) => item.key === templateKey);
  return template ? template.createField(resolveText) : null;
}

export function normalizeField(field: ModuleField): ModuleField {
  const enumValues = field.enumOptions?.map((item) => item.value).filter(Boolean) ?? [];
  return {
    ...field,
    required: field.required ?? false,
    searchable: field.searchable ?? false,
    sortable: field.sortable ?? false,
    visibleInList: field.visibleInList ?? true,
    visibleInForm: field.visibleInForm ?? true,
    validation: {
      ...field.validation,
      unique: field.validation?.unique ?? false,
      enum: field.type === 'enum' ? enumValues : field.validation?.enum,
    },
    enumOptions: field.type === 'enum' ? (field.enumOptions ?? []) : undefined,
    dictCode: field.type === 'enum' ? field.dictCode : undefined,
  };
}

export function normalizeFields(fields: ModuleField[]): ModuleField[] {
  return fields.map(normalizeField);
}

export function getTableRole(schema: Pick<ModuleSchema, 'metadata'>): BusinessTableRole {
  return schema.metadata?.tableRole ?? 'main';
}

export function shouldGenerateNavigation(schema: Pick<ModuleSchema, 'metadata'>): boolean {
  return getTableRole(schema) !== 'relation';
}

export function generateDefaultMenus(schema: ModuleSchema): MenuSeedConfig[] {
  if (!shouldGenerateNavigation(schema)) {
    return [];
  }
  const { scope, name } = schema;
  const segments = splitModuleSegments(name);
  const modelName = inferModelName(schema);
  const titleKey = buildTitleKey(scope, name);
  const routePath = buildPageRoutePath(scope, name);
  const routeName = buildRouteName(scope, name);
  const moduleNamespace = buildModuleNamespace(scope, name);
  const permissionPrefix = buildPermissionPrefix(scope, name);
  const menuKey = segments.join('-');
  const actions = getPageActions(schema).filter((action) => action !== 'detail');
  const groupMenus: MenuSeedConfig[] = segments.slice(0, -1).map((_, index) => {
    const groupSegments = segments.slice(0, index + 1);
    const groupKey = groupSegments.join('-');
    const parentSegments = groupSegments.slice(0, -1);
    return {
      key: groupKey,
      parentKey: parentSegments.length > 0 ? parentSegments.join('-') : scope,
      titleKey: buildMenuGroupTitleKey(scope, groupSegments),
      path: scope === 'business' ? `/operations/${groupSegments.join('/')}` : `/${scope}/${groupSegments.join('/')}`,
      type: 'M',
      icon: 'apps',
      routeName: `${scope}-${groupSegments.join('-')}`,
      module: `${scope}.${groupSegments.join('.')}`,
      sort: 10,
    };
  });

  return [
    ...groupMenus,
    {
      key: menuKey,
      parentKey: segments.length > 1 ? segments.slice(0, -1).join('-') : scope,
      titleKey,
      path: routePath,
      component: buildComponentKey(scope, name, modelName),
      pagePermission: `${permissionPrefix}:list`,
      type: 'C',
      icon: 'apps',
      routeName,
      module: moduleNamespace,
      sort: 10,
    },
    ...actions.map((action, index) => ({
      key: `${menuKey}-${action}`,
      parentKey: menuKey,
      titleKey: buildPermissionTitleKey(scope, name, action),
      perms: `${permissionPrefix}:${action}`,
      type: 'F' as const,
      sort: index + 1,
    })),
  ];
}

export function generateDefaultPermissions(schema: ModuleSchema): PermissionConfig[] {
  if (!shouldGenerateNavigation(schema)) {
    return [];
  }
  const { scope, name } = schema;
  const permissionPrefix = buildPermissionPrefix(scope, name);
  const moduleNamespace = buildModuleNamespace(scope, name);
  const actions = getPageActions(schema);

  return [
    {
      key: `${permissionPrefix}:list`,
      name: buildPermissionTitleKey(scope, name, 'view'),
      type: 'menu',
      module: moduleNamespace,
    },
    ...actions
      .filter((action) => action !== 'detail')
      .map((action) => ({
        key: `${permissionPrefix}:${action}`,
        name: buildPermissionTitleKey(scope, name, action),
        type: 'button' as const,
        module: moduleNamespace,
      })),
  ];
}

export function buildMenuPreview(schema: Pick<ModuleSchema, 'menus'>): GeneratorMenuPreviewNode[] {
  const nodes = new Map<string, GeneratorMenuPreviewNode>();
  const roots: GeneratorMenuPreviewNode[] = [];

  for (const menu of schema.menus) {
    nodes.set(menu.key, {
      key: menu.key,
      titleKey: menu.titleKey,
      path: menu.path,
      type: menu.type,
      module: menu.module,
      children: [],
    });
  }

  for (const menu of schema.menus) {
    const node = nodes.get(menu.key);
    if (!node) {
      continue;
    }
    const parent = menu.parentKey ? nodes.get(menu.parentKey) : undefined;
    if (parent) {
      parent.children.push(node);
      continue;
    }
    roots.push(node);
  }

  return roots;
}

export function validateGeneratorCompleteness(schema: ModuleSchema): GeneratorCompletenessIssue[] {
  const issues: GeneratorCompletenessIssue[] = [];
  const segments = splitModuleSegments(schema.name);
  const zh = schema.i18n.translations.zh;
  const en = schema.i18n.translations.en;
  const requiredKeys = new Set<string>();

  if (schema.scope === 'business' && segments.length > 0) {
    const expectedContext = segments[0];
    if (!schema.metadata?.businessContext) {
      issues.push({
        code: 'business_context_missing',
        level: 'warn',
        messageKey: 'generator.validation.businessContextMissing',
        detail: expectedContext,
      });
    }
  }

  if (getTableRole(schema) === 'relation' && schema.menus.length > 0) {
    issues.push({
      code: 'relation_table_has_menu',
      level: 'error',
      messageKey: 'generator.validation.relationTableHasMenu',
      detail: schema.name,
    });
  }

  for (const dependency of schema.dependencies ?? []) {
    if (
      !isValidScopedModulePath('business', dependency.module) &&
      !isValidScopedModulePath('system', dependency.module)
    ) {
      issues.push({
        code: 'dependency_invalid',
        level: 'error',
        messageKey: 'generator.validation.dependencyInvalid',
        detail: dependency.module,
      });
    }
  }

  for (const relation of schema.relations ?? []) {
    if (!['oneToMany', 'manyToMany', 'lookup'].includes(relation.type)) {
      issues.push({
        code: 'relation_type_invalid',
        level: 'error',
        messageKey: 'generator.validation.relationTypeInvalid',
        detail: relation.type,
      });
    }
    if (!relation.name || !relation.targetModule || !relation.localField || !relation.targetField) {
      issues.push({
        code: 'relation_incomplete',
        level: 'error',
        messageKey: 'generator.validation.relationIncomplete',
        detail: relation.name || schema.name,
      });
    }
    if (relation.targetLabelField && !/^[A-Za-z][A-Za-z0-9_]*$/.test(relation.targetLabelField)) {
      issues.push({
        code: 'relation_target_label_field_invalid',
        level: 'error',
        messageKey: 'generator.validation.relationIncomplete',
        detail: relation.targetLabelField,
      });
    }
    if (relation.lookupValueField && !/^[A-Za-z][A-Za-z0-9_]*$/.test(relation.lookupValueField)) {
      issues.push({
        code: 'relation_lookup_value_field_invalid',
        level: 'error',
        messageKey: 'generator.validation.relationIncomplete',
        detail: relation.lookupValueField,
      });
    }
    if (relation.lookupApi && !String(relation.lookupApi).startsWith('/')) {
      issues.push({
        code: 'relation_lookup_api_invalid',
        level: 'error',
        messageKey: 'generator.validation.relationIncomplete',
        detail: relation.lookupApi,
      });
    }
  }

  for (const menu of schema.menus) {
    requiredKeys.add(menu.titleKey);
  }
  requiredKeys.add(buildTitleKey(schema.scope, schema.name));
  if (
    schema.scope === 'business' &&
    shouldGenerateNavigation(schema) &&
    schema.includeDashboardWidget !== false
  ) {
    requiredKeys.add(buildDashboardQuickActionDescriptionKey(schema.scope, schema.name));
  }
  for (const field of schema.model.fields) {
    requiredKeys.add(buildFieldLabelKey(schema.scope, schema.name, field.name));
    if (field.placeholder || field.placeholderEn) {
      requiredKeys.add(buildFieldPlaceholderKey(schema.scope, schema.name, field.name));
    }
    for (const option of field.enumOptions ?? []) {
      requiredKeys.add(buildEnumOptionKey(schema.scope, schema.name, field.name, option.value));
    }
  }
  for (const permission of schema.permissions) {
    requiredKeys.add(permission.name);
  }
  for (const auditAction of ['create', 'update', 'delete'] as const) {
    requiredKeys.add(buildAuditActionKey(schema.scope, schema.name, auditAction));
  }

  for (const key of requiredKeys) {
    if (!zh[key]) {
      issues.push({
        code: 'i18n_zh_missing',
        level: 'error',
        messageKey: 'generator.validation.i18nZhMissing',
        detail: key,
      });
    }
    if (!en[key]) {
      issues.push({
        code: 'i18n_en_missing',
        level: 'error',
        messageKey: 'generator.validation.i18nEnMissing',
        detail: key,
      });
    }
  }

  return issues;
}
