export function createCleanupFixtureCache() {
  let dictTypesPromise = null;
  const i18nListPromises = new Map();

  return {
    async getDictTypes(load) {
      if (!dictTypesPromise) {
        dictTypesPromise = Promise.resolve().then(load);
      }
      return dictTypesPromise;
    },
    clearDictTypes() {
      dictTypesPromise = null;
    },
    async getI18nList(queryKey, load) {
      if (!i18nListPromises.has(queryKey)) {
        i18nListPromises.set(queryKey, Promise.resolve().then(load));
      }
      return i18nListPromises.get(queryKey);
    },
    clearI18nList(queryKey) {
      if (typeof queryKey === 'string') {
        i18nListPromises.delete(queryKey);
        return;
      }
      i18nListPromises.clear();
    },
  };
}
