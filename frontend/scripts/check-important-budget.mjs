/**
 * `!important` budget guard for P2-5.
 *
 * All historical `!important` declarations have been migrated to deletion,
 * component-native behavior, or scoped selectors. This zero-tolerance guard
 * prevents priority debt from returning; prefer supported component APIs, CSS
 * variables, or narrowly scoped selectors for future overrides.
 */
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const BUDGET = 0;

function walk(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(full));
    else if (entry.name.endsWith('.css')) out.push(full);
  }
  return out;
}

const root = path.join(process.cwd(), 'src');
let total = 0;
const perFile = [];
for (const file of walk(root)) {
  const count = (fs.readFileSync(file, 'utf8').match(/!important/g) || []).length;
  if (count > 0) {
    perFile.push([count, path.relative(process.cwd(), file)]);
    total += count;
  }
}

perFile.sort((a, b) => b[0] - a[0]);
for (const [count, file] of perFile) {
  console.log(`  ${String(count).padStart(4)}  ${file}`);
}
console.log(`\nTotal !important: ${total} / budget ${BUDGET}`);

if (total > BUDGET) {
  console.error(
    `\n!important budget exceeded (${total} > ${BUDGET}). ` +
      `Avoid new !important; prefer Arco CSS-variable customization.`,
  );
  process.exit(1);
}
if (total < BUDGET) {
  console.log(
    `Nice — ${BUDGET - total} below budget. Lower BUDGET in this script to ${total} to lock the win.`,
  );
}
console.log('!important budget check passed.');
