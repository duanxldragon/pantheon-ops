const buildLocaleFromBase = (baseResource, translatedValues, localeName = 'derived-locale') => {
  const keys = Object.keys(baseResource);
  if (keys.length !== translatedValues.length) {
    throw new Error(
      `[i18n] ${localeName} value count mismatch: expected ${keys.length}, got ${translatedValues.length}`,
    );
  }

  return Object.fromEntries(keys.map((key, index) => [key, translatedValues[index]]));
};

export default buildLocaleFromBase;
