/**
 * Fingerprint baseline for pattern checkers.
 *
 * Why: `check-security-patterns.mjs` never really ran (its ripgrep dependency
 * failed silently), so the rules' legacy hits were never triaged. Turning the
 * scanner on surfaces 46 production hits on day one. Failing on all of them
 * would make a blocking gate red for debt nobody has decided to fix; ignoring
 * them would hide new ones. A baseline encodes "these exact lines are accepted,
 * anything else fails" — the same ratchet shape as config/boundary-baseline.json.
 *
 * Fingerprint = description + file + trimmed line text. It survives line moves
 * and formatting of neighbouring code, and changes as soon as the matched line
 * itself changes, so an edited call re-enters review instead of hiding.
 *
 * `reviewBy` exists so the baseline cannot silently become permanent: the
 * checker prints it, and a past date is the reviewer's cue to re-triage.
 */

import fs from 'node:fs';
import path from 'node:path';

export function fingerprintOf(description, match) {
  return `${description}::${match.file}::${match.text}`;
}

/** loadFingerprints returns a Set, or null when no baseline file exists. */
export function loadFingerprints(filePath) {
  if (!fs.existsSync(filePath)) {
    return null;
  }
  const payload = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  if (!Array.isArray(payload.fingerprints)) {
    throw new Error(`${filePath}: "fingerprints" must be an array`);
  }
  return new Set(payload.fingerprints);
}

/** saveFingerprints writes a sorted, de-duplicated baseline. */
export function saveFingerprints(filePath, fingerprints, reviewBy) {
  const unique = [...new Set(fingerprints)].sort((left, right) => left.localeCompare(right));
  const payload = {
    reviewBy,
    note:
      'Accepted findings for a pattern checker. Regenerate with --write-baseline after triage. ' +
      'A finding not listed here fails the gate.',
    fingerprints: unique,
  };
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(payload, null, 2)}\n`);
  return unique.length;
}

/** reviewByDate returns today + the given number of days as YYYY-MM-DD. */
export function reviewByDate(days = 90) {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}
