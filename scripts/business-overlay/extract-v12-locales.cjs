const fs = require('node:fs');

// Extract the hand-maintained V1.2 i18n keys from the generated business
// fallback TS files into a module locale JSON so the rebuild's
// generateBusinessFallbacks step reproduces the full key set.
const businessRoot = 'frontend/src/modules/business';
const outDir = businessRoot + '/v12locales';
fs.mkdirSync(outDir, { recursive: true });

for (const locale of ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR']) {
  const src = `frontend/src/i18n/resources/business/${locale}.ts`;
  if (!fs.existsSync(src)) continue;
  const content = fs.readFileSync(src, 'utf8');
  // Keys are quoted with single quotes; values are single-quoted strings
  // (may contain escaped quotes) or primitives.
  const re = /'((?:[^'\\]|\\.)+)':\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|true|false|null|-?[\d.]+),?/g;
  const obj = {};
  let m;
  while ((m = re.exec(content)) !== null) {
    let v = m[2];
    if (v.startsWith("'") || v.startsWith('"')) {
      v = v.slice(1, -1);
      v = v.replace(/\\'/g, "'").replace(/\\"/g, '"').replace(/\\\\/g, '\\');
    }
    obj[m[1]] = v;
  }
  fs.writeFileSync(`${outDir}/${locale}.json`, JSON.stringify(obj, null, 2) + '\n');
  console.log(`${locale}: ${Object.keys(obj).length} keys`);
}
