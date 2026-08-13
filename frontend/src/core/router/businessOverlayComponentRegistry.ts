import { lazy, type LazyExoticComponent, type ComponentType } from 'react';

type ComponentLoader = () => Promise<{ default: ComponentType }>;

interface RegistryEntry {
  component: LazyExoticComponent<ComponentType>;
  preload: ComponentLoader;
}

function defineRegistryEntry(loader: ComponentLoader): RegistryEntry {
  return { component: lazy(loader), preload: loader };
}

export const businessOverlayComponentRegistry = {
  'business/bizscope/BizScopeList': defineRegistryEntry(() => import('../../modules/business/bizscope/BizScopeList')),
  'business/bizscope/BizScopeDetail': defineRegistryEntry(() => import('../../modules/business/bizscope/BizScopeDetail')),
  'business/cmdb/host/CmdbHostList': defineRegistryEntry(() => import('../../modules/business/cmdb/host/CmdbHostList')),
  'business/cmdb/host/CmdbHostDetail': defineRegistryEntry(() => import('../../modules/business/cmdb/host/CmdbHostDetail')),
  'business/cmdb/group/CmdbGroupList': defineRegistryEntry(() => import('../../modules/business/cmdb/group/CmdbGroupList')),
  'business/cmdb/label/CmdbLabelSchemaList': defineRegistryEntry(() => import('../../modules/business/cmdb/label/CmdbLabelSchemaList')),
  'business/deploy/package/DeployPackageList': defineRegistryEntry(() => import('../../modules/business/deploy/package/DeployPackageList')),
  'business/deploy/template/DeployTemplateList': defineRegistryEntry(() => import('../../modules/business/deploy/template/DeployTemplateList')),
  'business/deploy/task/DeployTaskList': defineRegistryEntry(() => import('../../modules/business/deploy/task/DeployTaskList')),
  'business/deploy/task/DeployTaskDetail': defineRegistryEntry(() => import('../../modules/business/deploy/task/DeployTaskDetail')),
} satisfies Record<string, RegistryEntry>;
