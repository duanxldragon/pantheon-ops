import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { getLangPack } from '../modules/system/i18n/api';
import { shouldFetchRemoteI18nPack } from '../core/runtime/automationPolicy.ts';

export const SUPPORTED_LOCALES = ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'] as const;

export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

type FallbackResourceMap = Record<string, string>;

const fallbackLoaders: Record<SupportedLocale, () => Promise<FallbackResourceMap>> = {
  'zh-CN': async () => (await import('./resources/zh-CN')).default,
  'en-US': async () => (await import('./resources/en-US')).default,
  'ja-JP': async () => (await import('./resources/ja-JP')).default,
  'ko-KR': async () => (await import('./resources/ko-KR')).default,
  'fr-FR': async () => (await import('./resources/fr-FR')).default,
};

const overrideFallbackLoaders: Record<SupportedLocale, () => Promise<FallbackResourceMap>> = {
  'zh-CN': async () => ({}),
  'en-US': async () => ({}),
  'ja-JP': async () => (await import('./resources/overrides/ja-JP')).default,
  'ko-KR': async () => (await import('./resources/overrides/ko-KR')).default,
  'fr-FR': async () => (await import('./resources/overrides/fr-FR')).default,
};

const generatedFallbackLoaders: Record<SupportedLocale, () => Promise<FallbackResourceMap>> = {
  'zh-CN': async () => (await import('./resources/generated/zh-CN')).default,
  'en-US': async () => (await import('./resources/generated/en-US')).default,
  'ja-JP': async () => (await import('./resources/generated/ja-JP')).default,
  'ko-KR': async () => (await import('./resources/generated/ko-KR')).default,
  'fr-FR': async () => (await import('./resources/generated/fr-FR')).default,
};
const localeResourceTasks = new Map<SupportedLocale, Promise<void>>();

function normalizeLocale(locale: string | null | undefined): SupportedLocale {
  const normalized = (locale || '').trim() as SupportedLocale;
  return SUPPORTED_LOCALES.includes(normalized) ? normalized : 'zh-CN';
}

async function loadFallbackResources(locale: SupportedLocale) {
  const [baseResources, overrideResources, generatedResources] = await Promise.all([
    fallbackLoaders[locale](),
    overrideFallbackLoaders[locale](),
    generatedFallbackLoaders[locale](),
  ]);
  return {
    ...baseResources,
    ...overrideResources,
    ...generatedResources,
  };
}

// 获取全量语言包 API
async function fetchLangPack(locale: string) {
  if (!shouldFetchRemoteI18nPack()) {
    return {};
  }
  try {
    const data = await getLangPack(locale);
    return data;
  } catch (error) {
    console.error('Failed to load i18n pack', error);
    return {};
  }
}

async function buildLocaleResources(locale: SupportedLocale) {
  const [localResources, remoteResources] = await Promise.all([
    loadFallbackResources(locale),
    fetchLangPack(locale),
  ]);

  return {
    translation: {
      ...localResources,
      ...remoteResources,
    },
  };
}

function hasLocaleResources(locale: SupportedLocale) {
  return i18n.hasResourceBundle(locale, 'translation');
}

async function ensureSingleLocaleResources(locale: SupportedLocale) {
  if (hasLocaleResources(locale)) {
    return;
  }
  const currentTask = localeResourceTasks.get(locale);
  if (currentTask) {
    return currentTask;
  }

  const nextTask = buildLocaleResources(locale)
    .then((resources) => {
      i18n.addResourceBundle(locale, 'translation', resources.translation, true, true);
    })
    .finally(() => {
      localeResourceTasks.delete(locale);
    });
  localeResourceTasks.set(locale, nextTask);
  return nextTask;
}

async function ensureLocaleResources(locale: SupportedLocale) {
  const fallbackLocale: SupportedLocale = 'zh-CN';
  const localesToLoad = locale === fallbackLocale ? [fallbackLocale] : [fallbackLocale, locale];

  await Promise.all(localesToLoad.map((item) => ensureSingleLocaleResources(item)));
}

// 初始化 i18n
export async function initI18n() {
  const currentLang = normalizeLocale(localStorage.getItem('pantheon_lang'));
  const resources: Record<string, { translation: Record<string, string> }> = {};
  const fallbackLocale: SupportedLocale = 'zh-CN';

  resources[fallbackLocale] = await buildLocaleResources(fallbackLocale);
  if (currentLang !== fallbackLocale) {
    resources[currentLang] = await buildLocaleResources(currentLang);
  }

  await i18n.use(initReactI18next).init({
    resources,
    lng: currentLang,
    fallbackLng: 'zh-CN',
    interpolation: {
      escapeValue: false,
    },
  });
}

export async function switchI18nLanguage(locale: string) {
  const nextLocale = normalizeLocale(locale);
  if (i18n.language === nextLocale && hasLocaleResources(nextLocale)) {
    return;
  }
  await ensureLocaleResources(nextLocale);
  if (i18n.language !== nextLocale) {
    await i18n.changeLanguage(nextLocale);
  }
}

// 供页面手动刷新翻译资源使用
export async function reloadI18nResources() {
  const currentLang = normalizeLocale(i18n.language);
  await ensureLocaleResources(currentLang);
  const remoteResources = await fetchLangPack(currentLang);
  i18n.addResourceBundle(currentLang, 'translation', remoteResources, true, true);
}

export default i18n;
