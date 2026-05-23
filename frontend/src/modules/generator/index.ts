/**
 * 兼容旧导入路径的生成器工具导出。
 */

// Schema 定义
export {
  type ModuleSchema,
  type ModuleScope,
  type TemplateLevel,
  type FieldType,
  type ModuleField,
  type EnumOption,
  type PageActionKey,
  type PageActionTemplate,
  type FieldTemplateKey,
  type FieldValidation,
  type MenuSeedConfig,
  type MenuType,
  type PermissionConfig,
  type I18nConfig,
  FIELD_TEMPLATE_DEFINITIONS,
  PAGE_ACTION_TEMPLATE_DEFINITIONS,
  getPageActions,
  normalizeField,
  normalizeFields,
  applyFieldTemplate,
  inferPackageName,
  inferModelName,
  generateDefaultMenus,
  generateDefaultPermissions,
} from './schema';

// 类型映射
export {
  type TypeMapping,
  TYPE_MAPPING,
  getGoType,
  getTSType,
  getSQLType,
  getGORMTag,
  generateStructTags,
  getRequiredImports,
  TS_TYPE_UTILS,
} from './type-mapping';

// 后端生成器
export { BackendGenerator } from './backend-generator';

// 前端生成器
export { FrontendGenerator } from './frontend-generator';

// 导出器
export { ModuleExporter, type GeneratedFile } from './exporter';
