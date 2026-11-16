import { describe, expect, it } from 'vitest';
import zhCN from '../../../../src/i18n/resources/business/zh-CN';
import enUS from '../../../../src/i18n/resources/business/en-US';
import jaJP from '../../../../src/i18n/resources/business/ja-JP';
import koKR from '../../../../src/i18n/resources/business/ko-KR';
import frFR from '../../../../src/i18n/resources/business/fr-FR';
const canonicalLocales = import.meta.glob<Record<string, string>>('../../../../src/modules/business/observability/locales/*.json', { eager: true, import: 'default' });
const prefix = 'business.observability.workspace.';
const permissionKeys = ['queryHistoryView', 'queryHistoryDelete', 'queryHistoryRestore', 'savedViewView', 'savedViewCreate', 'savedViewUpdate', 'savedViewDelete', 'savedViewRestore'].map((key) => 'business.observability.permission.' + key);
describe('supported observability workspace locale contract', () => {
  it.each(Object.entries({ 'zh-CN': zhCN, 'en-US': enUS, 'ja-JP': jaJP, 'ko-KR': koKR, 'fr-FR': frFR }))('%s provides every workspace and permission label from canonical business JSON', (locale, resources) => {
    const translations = resources as Record<string, string>;
    const canonical = canonicalLocales[`../../../../src/modules/business/observability/locales/${locale}.json`];
    expect(canonical).toBeDefined();
    const required = Object.keys(zhCN).filter((key) => key.startsWith(prefix)).concat(permissionKeys);
    expect(required).toHaveLength(62);
    for (const key of required) {
      expect(typeof translations[key]).toBe('string');
      expect(translations[key].trim().length).toBeGreaterThan(0);
      expect(translations[key]).not.toBe(key);
      expect(canonical[key]).toBe(translations[key]);
    }
  });
});
