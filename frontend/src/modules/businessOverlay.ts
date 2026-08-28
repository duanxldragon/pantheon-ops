import type { ModuleConfig } from '../core/router/types';
import { BizScopeModule } from './business/bizscope';
import { CmdbModule } from './business/cmdb';
import { DeployModule } from './business/deploy';
import { K8sModule } from './business/k8s';
import { ServiceModule } from './business/service';
// ArgoCD (CICD) module is disabled in v1.0 (shell implementation). It returns
// with the v1.2 real integration; see BUSINESS_ARGOCD_MODULE_DESIGN.md.
// import { CICDModule } from './business/cicd';
import { ObservabilityModule } from './business/observability';

const isObservabilityEnabled = import.meta.env.VITE_PANTHEON_EXPERIMENTAL_OBSERVABILITY === 'true';

export const overlayBusinessModules: ModuleConfig[] = [
  BizScopeModule,
  CmdbModule,
  DeployModule,
  K8sModule,
  ServiceModule,
  ...(isObservabilityEnabled ? [ObservabilityModule] : []),
];
