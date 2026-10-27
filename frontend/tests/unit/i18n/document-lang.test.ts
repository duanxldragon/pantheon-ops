import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../../src/modules/system/i18n/api', () => ({
  getLangPack: vi.fn(async () => ({})),
}));

vi.mock('../../../src/core/runtime/automationPolicy.ts', () => ({
  shouldFetchRemoteI18nPack: () => false,
}));

import i18n, { SUPPORTED_LOCALES, syncDocumentLang } from '../../../src/i18n';

describe('<html lang> follows the active locale', () => {
  beforeEach(() => {
    document.documentElement.lang = 'en';
  });

  it('writes every supported locale through to the document element', () => {
    for (const locale of SUPPORTED_LOCALES) {
      syncDocumentLang(locale);
      expect(document.documentElement.lang).toBe(locale);
    }
  });

  it('falls back to the app default for an unknown locale instead of leaking it', () => {
    syncDocumentLang('de-DE');
    expect(document.documentElement.lang).toBe('zh-CN');
    syncDocumentLang('');
    expect(document.documentElement.lang).toBe('zh-CN');
  });

  // The subscription is what keeps the attribute in step for locale changes made
  // anywhere, not only through the helpers in this module.
  it('stays in sync when i18next reports a language change', () => {
    i18n.emit('languageChanged', 'ja-JP');
    expect(document.documentElement.lang).toBe('ja-JP');
  });
});
