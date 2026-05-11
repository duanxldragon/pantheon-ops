import type { LazyExoticComponent, ComponentType } from 'react';
import type { RegisteredComponentKey } from './componentRegistry';
import type { MenuIconKey } from '../menu/icon';
import type { DashboardWidgetDefinition } from '../workbench/dashboard';

export type ModuleScope = 'platform' | 'system' | 'business';

interface ModuleRouteConfigBase {
  path: string;
  routeName?: string;
  titleKey: string;
  icon?: MenuIconKey;
  isCache?: boolean;
  activeMenu?: string;
  pagePermission?: string;
}

export type ModuleRouteConfig = ModuleRouteConfigBase &
  (
    | {
        component: LazyExoticComponent<ComponentType>;
        componentKey?: RegisteredComponentKey;
      }
    | {
        component?: undefined;
        componentKey: RegisteredComponentKey;
      }
  );

export interface ModuleMenuMeta {
  titleKey: string;
  path: string;
  icon?: MenuIconKey;
  routeName?: string;
  module?: string;
  isCache?: boolean;
  isExternal?: boolean;
  activeMenu?: string;
}

export interface ModuleConfig {
  name: string;
  scope: ModuleScope;
  routes: ModuleRouteConfig[];
  menus?: ModuleMenuMeta[];
  dashboardWidgets?: DashboardWidgetDefinition[];
  permissions?: string[];
  i18nNamespaces?: string[];
  featureFlags?: string[];
}

export function defineModule(config: ModuleConfig): ModuleConfig {
  return config;
}
