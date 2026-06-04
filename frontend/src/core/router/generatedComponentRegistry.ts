import { type LazyExoticComponent, type ComponentType } from 'react';

type ComponentLoader = () => Promise<{ default: ComponentType }>;

interface RegistryEntry {
	component: LazyExoticComponent<ComponentType>;
	preload: ComponentLoader;
}

export const generatedComponentRegistry = {
} satisfies Record<string, RegistryEntry>;
