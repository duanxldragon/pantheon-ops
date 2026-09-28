import type { TFunction } from 'i18next';

const roleNameTranslationKeyPattern = /^role\.[a-z0-9_.-]+\.name$/i;

function normalizeRoleName(roleName: string | null | undefined): string {
  return String(roleName || '').trim();
}

export function isRoleNameTranslationKey(roleName: string | null | undefined): boolean {
  return roleNameTranslationKeyPattern.test(normalizeRoleName(roleName));
}

export function translateRoleName(roleName: string | null | undefined, t: TFunction): string {
  const normalized = normalizeRoleName(roleName);
  if (!normalized) {
    return '-';
  }
  if (isRoleNameTranslationKey(normalized)) {
    return t(normalized, { defaultValue: normalized });
  }
  return normalized;
}

export function resolveSubmittedRoleName(
  roleName: string | null | undefined,
  originalRoleName: string | null | undefined,
  t: TFunction,
): string {
  const normalized = normalizeRoleName(roleName);
  const originalNormalized = normalizeRoleName(originalRoleName);
  if (!normalized) {
    return normalized;
  }
  if (
    isRoleNameTranslationKey(originalNormalized) &&
    normalized === translateRoleName(originalNormalized, t)
  ) {
    return originalNormalized;
  }
  return normalized;
}
