/**
 * 模块生成器 - 入口文件
 *
 * 导出所有生成器类和工具函数
 */

import { defineModule } from '../../../core/router/types';

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
} from './typeMapping';

// 后端生成器
export { BackendGenerator } from './backendGenerator';

// 前端生成器
export { FrontendGenerator } from './frontendGenerator';

// 导出器
export { ModuleExporter, type GeneratedFile } from './exporter';

export const GeneratorModule = defineModule({
  name: 'generator',
  scope: 'lowcode',
  routes: [
    {
      path: 'system/generator',
      routeName: 'system-generator',
      titleKey: 'system.menu.generator',
      icon: 'code',
      pagePermission: 'system:generator:use',
      componentKey: 'lowcode/generator/ModuleWizard',
    },
  ],
  menus: [
    {
      path: '/system/generator',
      titleKey: 'system.menu.generator',
      icon: 'code',
      routeName: 'system-generator',
      module: 'system.lowcode',
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.generator',
      slot: 'quick-action',
      sourceDomain: 'system/lowcode',
      titleKey: 'system.menu.generator',
      descriptionKey: 'dashboard.quickAction.generator',
      path: '/system/generator',
      permission: 'system:generator:use',
      icon: 'code',
      cleanupPolicy: 'hide_when_forbidden',
    },
  ],
  permissions: [
    'system:generator:use',
    'system:module:generate',
    'system:generator:datasource:manage',
  ],
  i18nNamespaces: ['generator', 'system.menu'],
});
