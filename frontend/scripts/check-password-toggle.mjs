/**
 * Mechanical gate for the password-visibility accessible-name contract (F-07/F-08).
 *
 * Arco's `Input.Password` renders its built-in toggle as a focusable <svg> with
 * `tabIndex: 0` and no accessible name, and exposes no prop to supply one. Every
 * password field therefore has to go through the shared `PasswordInput`, which
 * replaces it with a real <button aria-label> and stops the click from bubbling
 * to Arco's wrapper (that wrapper refocuses the <input>, which strands keyboard
 * users after one activation).
 *
 * All sites were migrated when this gate landed — it is a ratchet, not a TODO.
 *
 * Escape hatch: append a `password-toggle-allow: <reason>` comment on the line
 * and record why in the PR body.
 */
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const srcRoot = path.join(frontendRoot, 'src');
const ownerFile = path.join(srcRoot, 'components', 'patterns', 'forms', 'PasswordInput.tsx');

function walk(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(full));
    else if (/\.tsx?$/.test(entry.name)) out.push(full);
  }
  return out;
}

const findings = [];
const files = walk(srcRoot);

for (const file of files) {
  if (file === ownerFile) continue;
  const lines = fs.readFileSync(file, 'utf8').split(/\r?\n/);
  lines.forEach((line, index) => {
    if (!/Input\.Password\b/.test(line)) return;
    if (line.includes('password-toggle-allow:')) return;
    findings.push({
      file: path.relative(frontendRoot, file).replaceAll(path.sep, '/'),
      line: index + 1,
      text: line.trim(),
    });
  });
}

if (findings.length === 0) {
  console.log(`Password toggle contract check: 0 finding(s) across ${files.length} file(s)`);
  process.exitCode = 0;
} else {
  console.error(`Password toggle contract check: ${findings.length} finding(s)`);
  for (const finding of findings) {
    console.error(`finding: ${finding.file}:${finding.line}  ${finding.text}`);
  }
  console.error(
    '\nUse the shared `components/patterns/forms/PasswordInput` instead of `Input.Password`: Arco\'s own toggle has no accessible name. When a line genuinely must keep it, append a `password-toggle-allow: <reason>` comment and justify it in the PR body.',
  );
  process.exitCode = 1;
}
