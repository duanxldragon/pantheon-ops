import type { ModuleConfig } from '@duanxldragon/pantheon-base-ui';

import { CICDModule } from './cicd';
import { CmdbModule } from './cmdb';
import { DeployModule } from './deploy';
import { FiledistModule } from './filedist';
import { K8sModule } from './k8s';
import { ObservabilityModule } from './observability';
import { ServiceModule } from './service';
import { BizScopeModule } from './bizscope';

// Registration order mirrors the backend seed menu Sort so the sidebar shows
// business entries as one continuous block before system governance.
export const businessModules: ModuleConfig[] = [
  CmdbModule,
  BizScopeModule,
  ServiceModule,
  DeployModule,
  FiledistModule,
  K8sModule,
  CICDModule,
  ObservabilityModule,
];
