type FallbackResourceMap = Record<string, string>;

declare function buildLocaleFromBase(
  baseResource: FallbackResourceMap,
  translatedValues: readonly string[],
  localeName?: string,
): FallbackResourceMap;

export default buildLocaleFromBase;
