import { type LazyExoticComponent, type ComponentType } from 'react';

interface RegistryEntry {
	component: LazyExoticComponent<ComponentType>;
	preload: ComponentLoader;
}

type ComponentLoader = () => Promise<{ default: ComponentType }>;

export const generatedComponentRegistry = {
} satisfies Record<string, RegistryEntry>;
