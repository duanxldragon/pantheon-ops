const fs = require('node:fs');

// Append ops-only legacy business i18n keys (living in the base-owned
// resources/zh-CN.ts + en-US.ts of the ops source tree) into the v12locales
// module JSONs so the rebuild reproduces the complete key set.
const outDir = 'frontend/src/modules/business/v12locales';

for (const locale of ['zh-CN', 'en-US']) {
  const src = fs.readFileSync(`frontend/src/i18n/resources/${locale}.ts`, 'utf8');
  const out = JSON.parse(fs.readFileSync(`${outDir}/${locale}.json`, 'utf8'));
  const re = /'((?:[^'\\]|\\.)+)':\s*'((?:[^'\\]|\\.)*)'/g;
  const legacy = {};
  let m;
  while ((m = re.exec(src)) !== null) legacy[m[1]] = m[2];
  let added = 0;
  for (const [k, v] of Object.entries(legacy)) {
    if (k.startsWith('business.') && !(k in out)) {
      out[k] = v;
      added++;
    }
  }
  fs.writeFileSync(`${outDir}/${locale}.json`, JSON.stringify(out, null, 2) + '\n');
  console.log(`${locale}: added ${added} legacy keys, total ${Object.keys(out).length}`);
}
