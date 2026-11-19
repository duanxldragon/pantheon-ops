import type { ModuleConfig } from '../core/router/types';
import { CmdbModule } from './business/cmdb';
import { BizScopeModule } from './business/bizscope';
import { ServiceModule } from './business/service';
import { DeployModule } from './business/deploy';
import { FiledistModule } from './business/filedist';
import { K8sModule } from './business/k8s';
import { CICDModule } from './business/cicd';
import { ObservabilityModule } from './business/observability';

export const overlayBusinessModules: ModuleConfig[] = [
  CmdbModule,
  BizScopeModule,
  ServiceModule,
  DeployModule,
  FiledistModule,
  K8sModule,
  CICDModule,
  ObservabilityModule,
];
