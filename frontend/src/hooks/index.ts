// Re-export useTheme from core/theme/theme.ts for backward compatibility
export {
  usePantheonTheme as useTheme,
  pantheonThemeOptions,
  getStoredPantheonTheme,
  applyPantheonTheme,
  applyPantheonDefaultTheme,
  initializePantheonTheme,
  clearPantheonThemePreference,
  type PantheonThemeKey,
} from '../core/theme/theme';

export { usePermission } from './usePermission';
export { useGovernanceRail } from './useGovernanceRail';
export { usePagination } from './usePagination';
export type { UsePaginationOptions, UsePaginationResult } from './usePagination';
export { useRequest } from './useRequest';
export type { UseRequestOptions, UseRequestResult } from './useRequest';
