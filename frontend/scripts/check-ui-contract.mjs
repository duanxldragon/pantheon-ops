/**
 * Mechanical gate for the DESIGN.md §7 visual contract (§7.6 fonts, §7.7
 * tokens, §7.9 forbidden-pattern list).
 *
 * check-shell-visual-contract.mjs asserts structural rules on specific shell
 * files; this gate is the repo-wide sweep so business-module and generated
 * CSS cannot reintroduce patterns the shell already banned. All rules were at
 * zero occurrences when the gate landed — it is a ratchet, not a cleanup TODO.
 *
 * Escape hatch: append a `ui-contract-allow: <rule-id>` comment on the same
 * line (CSS or JSX) and record why in the PR body.
 */
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const srcRoot = path.join(frontendRoot, 'src');
const modulesRoot = path.join(srcRoot, 'modules');

const ALLOWED_FONT_WEIGHTS = new Set(['400', '500', '600', '700', 'normal', 'bold', 'inherit', 'unset']);
// Pure white/black are legitimate color-mix() anchors for tint generation.
const NEUTRAL_HEX = /^#(?:fff|ffffff|000|000000)$/i;

function checkFontWeightValue(rawValue) {
  const value = rawValue.trim().replace(/\s*!important\s*$/i, '').replace(/^['"]|['"]$/g, '');
  return ALLOWED_FONT_WEIGHTS.has(value) || value.startsWith('var(');
}

const RULES = [
  {
    id: 'no-radial-gradient',
    contract: 'DESIGN.md §7.9 — radial-gradient halo decoration is banned',
    extensions: ['.css', '.tsx', '.ts'],
    violates: (line) => /radial-gradient\s*\(/i.test(line),
  },
  {
    id: 'no-linear-gradient',
    contract: 'DESIGN.md §7.9 — broad linear-gradient surfaces are banned; use solid tokens',
    extensions: ['.css', '.tsx', '.ts'],
    violates: (line) => /linear-gradient\s*\(/i.test(line),
  },
  {
    id: 'standard-font-weight',
    contract: 'DESIGN.md §7.6 — font-weight must be 400/500/600/700 (or normal/bold/inherit/var())',
    extensions: ['.css'],
    violates: (line) => {
      const match = line.match(/font-weight\s*:\s*([^;}]+)/i);
      return match !== null && !checkFontWeightValue(match[1]);
    },
  },
  {
    id: 'standard-font-weight-inline',
    contract: 'DESIGN.md §7.6 — inline fontWeight must be 400/500/600/700 (or normal/bold)',
    extensions: ['.tsx', '.ts'],
    violates: (line) => {
      const match = line.match(/fontWeight\s*:\s*([^,}\n]+)/);
      return match !== null && !checkFontWeightValue(match[1]);
    },
  },
  {
    id: 'no-inter-font',
    contract: 'DESIGN.md §7.9 — Inter is not the product font; system-ui stack is the contract',
    extensions: ['.css', '.tsx', '.ts'],
    violates: (line) => /font(?:-f|F)amily[^;,}]*['"]Inter['"]/.test(line),
  },
  {
    id: 'no-raw-arco-token',
    contract: 'DESIGN.md §7.7 — raw Arco tokens (--color-text-1 etc.) are banned; use Pantheon tokens',
    extensions: ['.css', '.tsx', '.ts'],
    violates: (line) => /var\(\s*--color-(?:text|border|fill|bg)-\d/.test(line),
  },
  {
    id: 'no-module-hex-color',
    contract: 'DESIGN.md §7.7 — module CSS must reference Pantheon tokens, not hex literals',
    extensions: ['.css'],
    scope: modulesRoot,
    violates: (line) => {
      const matches = line.match(/#[0-9a-fA-F]{3,8}\b/g);
      return matches !== null && matches.some((hex) => !NEUTRAL_HEX.test(hex));
    },
  },
];

function walk(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(full));
    else out.push(full);
  }
  return out;
}

const files = walk(srcRoot);
const findings = [];

for (const file of files) {
  const extension = path.extname(file);
  const applicable = RULES.filter(
    (rule) =>
      rule.extensions.includes(extension) && (!rule.scope || file.startsWith(rule.scope + path.sep)),
  );
  if (applicable.length === 0) continue;
  const lines = fs.readFileSync(file, 'utf8').split(/\r?\n/);
  lines.forEach((line, index) => {
    for (const rule of applicable) {
      if (!rule.violates(line)) continue;
      if (line.includes(`ui-contract-allow: ${rule.id}`)) continue;
      findings.push({
        file: path.relative(frontendRoot, file).replaceAll(path.sep, '/'),
        line: index + 1,
        rule: rule.id,
        contract: rule.contract,
      });
    }
  });
}

if (findings.length === 0) {
  console.log(`UI contract check: 0 finding(s) across ${files.length} file(s)`);
  process.exitCode = 0;
} else {
  console.error(`UI contract check: ${findings.length} finding(s)`);
  for (const finding of findings) {
    console.error(`finding: ${finding.file}:${finding.line} [${finding.rule}]`);
    console.error(`  contract: ${finding.contract}`);
  }
  console.error(
    '\nFix the violation or, when a rule genuinely does not apply, append a `ui-contract-allow: <rule-id>` comment on the line and justify it in the PR body.',
  );
  process.exitCode = 1;
}
