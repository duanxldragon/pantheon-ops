export function buildCleanupI18nQueries(pattern) {
  const prefixes = Array.isArray(pattern?.prefixes) ? pattern.prefixes : [];
  const exacts = Array.isArray(pattern?.exacts) ? pattern.exacts : [];
  const queries = [];

  if (prefixes.some((value) => value.startsWith('i18n.'))) {
    queries.push('i18n.');
  }
  if (prefixes.some((value) => value.startsWith('dict.smoke'))) {
    queries.push('dict.smoke');
  }
  queries.push(...exacts);

  return Array.from(new Set(queries));
}
