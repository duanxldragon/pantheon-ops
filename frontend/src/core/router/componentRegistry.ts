import { lazy, type LazyExoticComponent, type ComponentType } from 'react';
import { generatedComponentRegistry } from './generatedComponentRegistry';

type ComponentLoader = () => Promise<{ default: ComponentType }>;

interface RegistryEntry {
  component: LazyExoticComponent<ComponentType>;
  preload: ComponentLoader;
}

function defineRegistryEntry(loader: ComponentLoader): RegistryEntry {
  return {
    component: lazy(loader),
    preload: loader,
  };
}

const staticComponentRegistry = {
  'dashboard': defineRegistryEntry(() => import('../../modules/dashboard/Dashboard')),
  'auth/SecurityCenter': defineRegistryEntry(() => import('../../modules/auth/SecurityCenter')),
  'auth/LoginLogList': defineRegistryEntry(() => import('../../modules/auth/LoginLogList')),
  'auth/SecurityEventList': defineRegistryEntry(
    () => import('../../modules/auth/SecurityEventList'),
  ),
  'auth/SessionList': defineRegistryEntry(() => import('../../modules/auth/SessionList')),
  'system/profile/ProfileCenter': defineRegistryEntry(
    () => import('../../modules/system/profile/ProfileCenter'),
  ),
  'system/dict/DictPage': defineRegistryEntry(() => import('../../modules/system/dict/DictPage')),
  'system/i18n/I18nList': defineRegistryEntry(() => import('../../modules/system/i18n/I18nList')),
  'system/dept/DeptList': defineRegistryEntry(() => import('../../modules/system/dept/DeptList')),
  'system/menu/MenuList': defineRegistryEntry(() => import('../../modules/system/menu/MenuList')),
  'system/permission/PermissionList': defineRegistryEntry(
    () => import('../../modules/system/permission/PermissionList'),
  ),
  'system/post/PostList': defineRegistryEntry(() => import('../../modules/system/post/PostList')),
  'system/role/RoleList': defineRegistryEntry(() => import('../../modules/system/role/RoleList')),
  'system/setting/SettingOverviewPage': defineRegistryEntry(
    () => import('../../modules/system/setting/SettingOverviewPage'),
  ),
  'system/setting/SettingGroupPage': defineRegistryEntry(
    () => import('../../modules/system/setting/SettingGroupPage'),
  ),
  'system/user/UserList': defineRegistryEntry(() => import('../../modules/system/user/UserList')),
  'system/user/UserDetail': defineRegistryEntry(
    () => import('../../modules/system/user/UserDetail'),
  ),
  'system/audit/OperationLogList': defineRegistryEntry(
    () => import('../../modules/system/audit/OperationLogList'),
  ),
  'system/dynamicmodule/ModuleManager': defineRegistryEntry(
    () => import('../../modules/system/dynamicmodule/ModuleManager'),
  ),
  'system/generator/ModuleWizard': defineRegistryEntry(
    () => import('../../modules/system/generator/pages/ModuleWizard'),
  ),
  'business/cmdb/host/CmdbHostList': defineRegistryEntry(
    () => import('../../modules/business/cmdb/host/CmdbHostList'),
  ),
  'business/cmdb/host/CmdbHostDetail': defineRegistryEntry(
    () => import('../../modules/business/cmdb/host/CmdbHostDetail'),
  ),
  'business/cmdb/group/CmdbGroupList': defineRegistryEntry(
    () => import('../../modules/business/cmdb/group/CmdbGroupList'),
  ),
  'business/cmdb/label/CmdbLabelSchemaList': defineRegistryEntry(
    () => import('../../modules/business/cmdb/label/CmdbLabelSchemaList'),
  ),
  'business/deploy/package/DeployPackageList': defineRegistryEntry(
    () => import('../../modules/business/deploy/package/DeployPackageList'),
  ),
  'business/deploy/task/DeployTaskList': defineRegistryEntry(
    () => import('../../modules/business/deploy/task/DeployTaskList'),
  ),
  'business/deploy/task/DeployTaskDetail': defineRegistryEntry(
    () => import('../../modules/business/deploy/task/DeployTaskDetail'),
  ),
} satisfies Record<string, RegistryEntry>;

const componentRegistry = {
  ...staticComponentRegistry,
  ...generatedComponentRegistry,
} satisfies Record<string, RegistryEntry>;

export type RegisteredComponentKey = keyof typeof componentRegistry;

export function getRegisteredComponent(key?: string) {
  if (!key) {
    return undefined;
  }
  return componentRegistry[key as RegisteredComponentKey]?.component;
}

export function preloadRegisteredComponent(key?: string) {
  if (!key) {
    return Promise.resolve(undefined);
  }
  return componentRegistry[key as RegisteredComponentKey]?.preload();
}

export function isRegisteredComponentKey(key?: string): key is RegisteredComponentKey {
  if (!key) {
    return false;
  }
  return Object.prototype.hasOwnProperty.call(componentRegistry, key);
}

export function listRegisteredComponentKeys(): RegisteredComponentKey[] {
  return Object.keys(componentRegistry) as RegisteredComponentKey[];
}
