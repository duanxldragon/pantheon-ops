import { type LazyExoticComponent, type ComponentType } from 'react';

interface RegistryEntry {
	component: LazyExoticComponent<ComponentType>;
	preload: ComponentLoader;
}

type ComponentLoader = () => Promise<{ default: ComponentType }>;

export const generatedComponentRegistry = {
	'business/bizscope/BizScopeList': defineRegistryEntry(() => import('../../modules/business/bizscope/BizScopeList')),
	'business/bizscope/BizScopeDetail': defineRegistryEntry(() => import('../../modules/business/bizscope/BizScopeDetail')),
} satisfies Record<string, RegistryEntry>;
