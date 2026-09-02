// Business 组件注册表
// 为 ops 业务模块注册组件

import { lazy } from 'react';
import type { ComponentRegistryMap } from '@duanxldragon/pantheon-base-ui';

export const businessOverlayComponentRegistry: ComponentRegistryMap = {
  'business/bizscope/BizScopeList': lazy(() => import('../../modules/business/bizscope/BizScopeList')),
  'business/bizscope/BizScopeDetail': lazy(() => import('../../modules/business/bizscope/BizScopeDetail')),
  'business/cmdb/host/CmdbHostList': lazy(() => import('../../modules/business/cmdb/host/CmdbHostList')),
  'business/cmdb/host/CmdbHostDetail': lazy(() => import('../../modules/business/cmdb/host/CmdbHostDetail')),
  'business/cmdb/group/CmdbGroupList': lazy(() => import('../../modules/business/cmdb/group/CmdbGroupList')),
  'business/cmdb/label/CmdbLabelSchemaList': lazy(() => import('../../modules/business/cmdb/label/CmdbLabelSchemaList')),
  'business/deploy/package/DeployPackageList': lazy(() => import('../../modules/business/deploy/package/DeployPackageList')),
  'business/deploy/template/DeployTemplateList': lazy(() => import('../../modules/business/deploy/template/DeployTemplateList')),
  'business/deploy/task/DeployTaskList': lazy(() => import('../../modules/business/deploy/task/DeployTaskList')),
  'business/deploy/task/DeployTaskDetail': lazy(() => import('../../modules/business/deploy/task/DeployTaskDetail')),
  'business/k8s/cluster/ClusterList': lazy(() => import('../../modules/business/k8s/cluster/ClusterList')),
  'business/k8s/cluster/ClusterDetail': lazy(() => import('../../modules/business/k8s/cluster/ClusterDetail')),
  'business/k8s/workload/WorkloadList': lazy(() => import('../../modules/business/k8s/workload/WorkloadList')),
  'business/k8s/service/ServiceList': lazy(() => import('../../modules/business/k8s/service/ServiceList')),
  'business/k8s/release/ReleaseList': lazy(() => import('../../modules/business/k8s/release/ReleaseList')),
  'business/k8s/ingress/IngressList': lazy(() => import('../../modules/business/k8s/ingress/IngressList')),
  'business/k8s/storage/StorageList': lazy(() => import('../../modules/business/k8s/storage/StorageList')),
  'business/k8s/backup/BackupList': lazy(() => import('../../modules/business/k8s/backup/BackupList')),
  'business/service/ServiceCatalog': lazy(() => import('../../modules/business/service/ServiceCatalog')),
  'business/observability/MonitoringCenter': lazy(() => import('../../modules/business/observability/MonitoringCenter')),
  'business/observability/AlertCenter': lazy(() => import('../../modules/business/observability/AlertCenter')),
  'business/observability/LogExplorer': lazy(() => import('../../modules/business/observability/LogExplorer')),
  'business/observability/TraceExplorer': lazy(() => import('../../modules/business/observability/TraceExplorer')),
  'business/observability/MetricSourceList': lazy(() => import('../../modules/business/observability/MetricSourceList')),
  'business/observability/AlertRuleList': lazy(() => import('../../modules/business/observability/AlertRuleList')),
  'business/observability/AlertRecordList': lazy(() => import('../../modules/business/observability/AlertRecordList')),
  'business/filedist/FiledistList': lazy(() => import('../../modules/business/filedist/FiledistList')),
  'business/filedist/FiledistDetail': lazy(() => import('../../modules/business/filedist/FiledistDetail')),
  'business/cicd/argocd/ArgoCDList': lazy(() => import('../../modules/business/cicd/argocd/ArgoCDList')),
};

