/**
 * `-1` is the backend's unlimited sentinel (`tenant.UnlimitedQuota`), not a
 * very large number, so it must render as "no ceiling" rather than "-1".
 */
export const UNLIMITED_QUOTA = -1;

export function isUnlimitedQuota(limit: number): boolean {
  return limit < 0;
}

/** A ceiling spoken the way the console reads it: the sentinel is "no ceiling". */
export function formatQuotaLimit(limit: number, unlimitedText: string): string {
  return isUnlimitedQuota(limit) ? unlimitedText : String(limit);
}
