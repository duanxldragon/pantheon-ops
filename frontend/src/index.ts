// Pantheon Base UI - 入口文件
// 导出所有公共模块和组件

// === Core Router ===
export * from './core/router/types';
export * from './core/router/modules';
export * from './core/router/componentRegistry';
export * from './core/router/RoutePermissionGuard';

// === Store ===
export * from './store/useAuthStore';
export * from './store/useMenuStore';
export * from './store/authTypes';

// === Modules - Auth ===
export * from './modules/auth';

// === Modules - Platform ===
export * from './modules/platform';

// === Components ===
export * from './components';

// === Hooks ===
export * from './hooks';

// === API (shared request / import-export / upload) ===
export * from './api/request';
export * from './api/importExport';
export * from './api/upload';

// === Shared core helpers ===
export * from './core/format/dateTime';
export * from './core/runtime/automationPolicy';

// === System i18n API (consumer bootstrap needs getLangPack) ===
export * from './modules/system/i18n/api';

// === App shell ===
export { default as App } from './App';
