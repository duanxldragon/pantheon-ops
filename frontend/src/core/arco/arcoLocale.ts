import zhCN from '@arco-design/web-react/es/locale/zh-CN';
import enUS from '@arco-design/web-react/es/locale/en-US';
import jaJP from '@arco-design/web-react/es/locale/ja-JP';
import koKR from '@arco-design/web-react/es/locale/ko-KR';
import frFR from '@arco-design/web-react/es/locale/fr-FR';
import { SUPPORTED_LOCALES, type SupportedLocale } from '../../i18n';

// Arco's per-locale bundle types differ slightly (some locales omit newer keys
// like Form/ColorPicker), so widen to the zh-CN shape which ConfigProvider accepts.
type ArcoLocale = typeof zhCN;

const ARCO_LOCALE_MAP: Record<SupportedLocale, ArcoLocale> = {
  'zh-CN': zhCN,
  'en-US': enUS as ArcoLocale,
  'ja-JP': jaJP as ArcoLocale,
  'ko-KR': koKR as ArcoLocale,
  'fr-FR': frFR as ArcoLocale,
};

/**
 * Resolve the Arco Design locale bundle for a given app language. Falls back to
 * zh-CN for anything outside the supported set. Used by the root ConfigProvider
 * so Arco's built-in strings (pagination "X / page", DatePicker, Empty, Table
 * sort hints, etc.) follow the app language instead of staying Chinese.
 */
export function resolveArcoLocale(language: string | null | undefined): ArcoLocale {
  const normalized = (language ?? '').trim() as SupportedLocale;
  if (SUPPORTED_LOCALES.includes(normalized)) {
    return ARCO_LOCALE_MAP[normalized];
  }
  return zhCN;
}
