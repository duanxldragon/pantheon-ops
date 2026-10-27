import type { ModuleConfig } from '@duanxldragon/pantheon-base-ui';

import { BizScopeModule } from './bizscope';
import { CmdbModule } from './cmdb';
import { DeployModule } from './deploy';
import { K8sModule } from './k8s';
import { ServiceModule } from './service';
import { CICDModule } from './cicd';
import { FiledistModule } from './filedist';
import { ObservabilityModule } from './observability';

export const businessModules: ModuleConfig[] = [
  BizScopeModule,
  CmdbModule,
  DeployModule,
  K8sModule,
  ServiceModule,
  CICDModule,
  FiledistModule,
  ObservabilityModule,
];
