import type { ModuleConfig } from '../core/router/types';
import { BizScopeModule } from './business/bizscope';
import { CmdbModule } from './business/cmdb';
import { DeployModule } from './business/deploy';
import { K8sModule } from './business/k8s';
import { ServiceModule } from './business/service';
import { CICDModule } from './business/cicd';
import { FiledistModule } from './business/filedist';
import { ObservabilityModule } from './business/observability';

export const overlayBusinessModules: ModuleConfig[] = [
  BizScopeModule,
  CmdbModule,
  DeployModule,
  K8sModule,
  ServiceModule,
  CICDModule,
  FiledistModule,
  ObservabilityModule,
];
