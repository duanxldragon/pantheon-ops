import type { ModuleConfig } from '../core/router/types';
import { BizScopeModule } from './business/bizscope';
import { CmdbModule } from './business/cmdb';
import { DeployModule } from './business/deploy';
import { K8sModule } from './business/k8s';
import { ServiceModule } from './business/service';
import { CICDModule } from './business/cicd';
import { ObservabilityModule } from './business/observability';
import { FiledistModule } from './business/filedist';

// Observability is a supported business surface. Keep an explicit opt-out for
// constrained deployments, but make the Web UI route available by default so
// a normal local/production build does not resolve to a 404 page.
const isObservabilityEnabled = import.meta.env.VITE_PANTHEON_EXPERIMENTAL_OBSERVABILITY !== 'false';

export const overlayBusinessModules: ModuleConfig[] = [
  BizScopeModule,
  CmdbModule,
  DeployModule,
  K8sModule,
  ServiceModule,
  CICDModule,
  FiledistModule,
  ...(isObservabilityEnabled ? [ObservabilityModule] : []),
];
