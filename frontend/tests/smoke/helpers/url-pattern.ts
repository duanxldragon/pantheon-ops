export function escapeRegexLiteral(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

export function buildUrlSuffixPattern(pathname: string) {
  return new RegExp(`${escapeRegexLiteral(pathname)}$`);
}
