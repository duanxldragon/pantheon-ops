import type { ModuleConfig } from '../core/router/types';
import { BizScopeModule } from './business/bizscope';
import { CmdbModule } from './business/cmdb';
import { DeployModule } from './business/deploy';

export const overlayBusinessModules: ModuleConfig[] = [
  BizScopeModule,
  CmdbModule,
  DeployModule,
];
