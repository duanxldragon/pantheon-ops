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
  'business/k8s/cluster/ClusterList': defineRegistryEntry(() => import('../../modules/business/k8s/cluster/ClusterList')),
  'business/k8s/cluster/ClusterDetail': defineRegistryEntry(() => import('../../modules/business/k8s/cluster/ClusterDetail')),
  'business/k8s/workload/WorkloadList': defineRegistryEntry(() => import('../../modules/business/k8s/workload/WorkloadList')),
  'business/k8s/release/ReleaseList': defineRegistryEntry(() => import('../../modules/business/k8s/release/ReleaseList')),
  'business/service/ServiceList': defineRegistryEntry(() => import('../../modules/business/service/ServiceList')),
  'business/observability/MetricSourceList': defineRegistryEntry(() => import('../../modules/business/observability/views/MetricSourceList')),
  'business/observability/AlertRuleList': defineRegistryEntry(() => import('../../modules/business/observability/views/AlertRuleList')),
  'business/observability/AlertRecordList': defineRegistryEntry(() => import('../../modules/business/observability/views/AlertRecordList')),
  'business/observability/ActiveAlertList': defineRegistryEntry(() => import('../../modules/business/observability/views/ActiveAlertList')),
  'business/observability/NotificationChannelList': defineRegistryEntry(() => import('../../modules/business/observability/views/NotificationChannelList')),
} satisfies Record<string, RegistryEntry>;
