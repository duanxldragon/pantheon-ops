/**
 * WCAG contrast self-check for Pantheon design tokens.
 *
 * Fulfils the THEME_TOKENS_REFERENCE §12 "AA contrast" promise with an
 * executable guard: it parses the real token values from `src/index.css`
 * (light `:root` block and the `[data-color-mode='dark']` block) and asserts
 * that the key text/surface pairs meet WCAG AA.
 *
 * Body text (primary/secondary) and button/link text must reach 4.5:1.
 * Tertiary text (placeholders, watermarks) is reported at the 3:1 large-text
 * bar and only warns, matching WCAG's exemption for incidental text.
 */
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const cssPath = path.join(process.cwd(), 'src', 'index.css');
const css = fs.readFileSync(cssPath, 'utf8');

function extractBlock(selectorRegex) {
  const match = css.match(selectorRegex);
  if (!match) return {};
  const start = match.index + match[0].length;
  const end = css.indexOf('}', start);
  const body = css.slice(start, end);
  const vars = {};
  for (const line of body.split('\n')) {
    const m = line.match(/^\s*(--[\w-]+):\s*(#[0-9a-fA-F]{3,8})\s*;/);
    if (m) vars[m[1]] = m[2];
  }
  return vars;
}

// Light lives in the first `:root, :root[data-pantheon-theme='indigo']` block.
const light = extractBlock(/:root,\s*\n?\s*:root\[data-pantheon-theme='indigo'\]\s*\{/);
// Dark neutral overrides.
const darkNeutral = extractBlock(/:root\[data-color-mode='dark'\]\s*\{/);
// Dark indigo brand override.
const darkBrand = extractBlock(
  /:root\[data-color-mode='dark'\]\[data-pantheon-theme='indigo'\][^{]*\{/,
);
const dark = { ...light, ...darkNeutral, ...darkBrand };

function hexToRgb(hex) {
  let h = hex.replace('#', '');
  if (h.length === 3)
    h = h
      .split('')
      .map((c) => c + c)
      .join('');
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16));
}
function relLum([r, g, b]) {
  const f = (c) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
}
function contrast(a, b) {
  const l1 = relLum(hexToRgb(a));
  const l2 = relLum(hexToRgb(b));
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
}

// [label, fg token, bg token, threshold, hard(fail)|soft(warn)]
const PAIRS = [
  ['primary text / panel', '--text-primary', '--panel-bg-solid', 4.5, true],
  ['primary text / app bg', '--text-primary', '--app-bg', 4.5, true],
  ['secondary text / panel', '--text-secondary', '--panel-bg-solid', 4.5, true],
  ['brand link / panel', '--brand-primary', '--panel-bg-solid', 4.5, true],
  ['tertiary text / panel', '--text-tertiary', '--panel-bg-solid', 3.0, false],
];

let failed = 0;
for (const mode of [
  ['light', light],
  ['dark', dark],
]) {
  const [name, vars] = mode;
  console.log(`\n${name.toUpperCase()}`);
  for (const [label, fgKey, bgKey, threshold, hard] of PAIRS) {
    const fg = vars[fgKey];
    const bg = vars[bgKey];
    if (!fg || !bg) {
      console.log(`  ? ${label}: missing token (${fgKey}/${bgKey})`);
      if (hard) failed += 1;
      continue;
    }
    const ratio = contrast(fg, bg);
    const ok = ratio >= threshold;
    const tag = ok ? 'PASS' : hard ? 'FAIL' : 'warn';
    console.log(`  ${tag} ${label}: ${ratio.toFixed(2)}:1 (need ${threshold}) ${fg} on ${bg}`);
    if (!ok && hard) failed += 1;
  }
}

if (failed > 0) {
  console.error(`\nContrast check FAILED: ${failed} pair(s) below AA.`);
  process.exit(1);
}
console.log('\nContrast check passed (AA for body/link text, both modes).');
