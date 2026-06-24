import type { ModuleConfig } from './types';
import { AuthModule } from '../../modules/auth';
import { DashboardModule } from '../../modules/dashboard';
import { GeneratorModule } from '../../modules/system/generator';
import { UserModule } from '../../modules/system/iam/user';
import { RoleModule } from '../../modules/system/iam/role';
import { MenuModule } from '../../modules/system/iam/menu';
import { PermissionModule } from '../../modules/system/iam/permission';
import { DeptModule } from '../../modules/system/org/dept';
import { PostModule } from '../../modules/system/org/post';
import { DictModule } from '../../modules/system/config/dict';
import { SettingModule } from '../../modules/system/config/setting';
import { I18nModule } from '../../modules/system/i18n';
import { DynamicModuleModule } from '../../modules/system/dynamicmodule';
import { AuditModule } from '../../modules/system/audit';
import { ProfileModule } from '../../modules/system/profile';
import { generatedSystemModules } from '../../modules/generated/system';
import { CmdbModule } from '../../modules/business/cmdb';
import { DeployModule } from '../../modules/business/deploy';
import { generatedBusinessModules } from '../../modules/generated/business';

export const systemModules: ModuleConfig[] = [
  DashboardModule,
  AuthModule,
  DictModule,
  DeptModule,
  PostModule,
  PermissionModule,
  UserModule,
  RoleModule,
  MenuModule,
  SettingModule,
  I18nModule,
  DynamicModuleModule,
  GeneratorModule,
  AuditModule,
  ProfileModule,
  ...generatedSystemModules,
];

export const businessModules: ModuleConfig[] = [CmdbModule, ...generatedBusinessModules, DeployModule];

export const registeredModules: ModuleConfig[] = [...systemModules, ...businessModules];

// 构建路由路径到路由配置的映射
const routePathMap = new Map<string, ModuleConfig['routes'][number]>();

registeredModules.forEach((module) => {
  module.routes.forEach((route) => {
    const normalizedPath = route.path.startsWith('/') ? route.path : `/${route.path}`;
    routePathMap.set(normalizedPath, route);
  });
});

// 构建系统路由标题映射表（用于后备显示）
export const systemRouteTitleMap: Record<string, string> = {};

registeredModules.forEach((module) => {
  if (module.scope === 'system' || module.scope === 'platform') {
    module.routes.forEach((route) => {
      const normalizedPath = route.path.startsWith('/') ? route.path : `/${route.path}`;
      systemRouteTitleMap[normalizedPath] = route.titleKey;
    });
  }
});

/**
 * 根据路径查找路由配置
 * @param path - 路由路径
 * @returns 路由配置对象，如果未找到则返回 undefined
 */
export function findRouteByPath(path: string) {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;

  // 直接匹配
  const exactMatch = routePathMap.get(normalizedPath);
  if (exactMatch) {
    return exactMatch;
  }

  // 处理动态路由（例如 /system/user/:id）
  for (const [routePath, route] of routePathMap.entries()) {
    if (routePath.includes(':')) {
      // 将动态路由转换为正则表达式
      const pattern = routePath.replace(/:(\w+)/g, '([^/]+)');
      const regex = new RegExp(`^${pattern}$`);
      if (regex.test(normalizedPath)) {
        return route;
      }
    }
  }

  return undefined;
}
