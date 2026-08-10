import type { ComponentType, LazyExoticComponent } from 'react';

type ComponentLoader = () => Promise<{ default: ComponentType }>;

interface RegistryEntry {
	component: LazyExoticComponent<ComponentType>;
	preload: ComponentLoader;
}

export const generatedComponentRegistry = {
} satisfies Record<string, RegistryEntry>;
