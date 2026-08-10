import fs from 'node:fs';
import process from 'node:process';

const markerPath = process.env.PANTHEON_CLEANUP_MARKER;

if (markerPath) {
  const payload = JSON.stringify({
    args: process.argv.slice(2),
    preserve: process.env.PANTHEON_SMOKE_PRESERVE_FIXTURES ?? '',
  });
  fs.appendFileSync(markerPath, `${payload}\n`, 'utf8');
}

const failOn = process.env.PANTHEON_CLEANUP_FAIL_ON;
const kindIndex = process.argv.indexOf('--kind');
const phaseIndex = process.argv.indexOf('--phase');
const cleanupIdentity = `${process.argv[kindIndex + 1]}:${process.argv[phaseIndex + 1]}`;

process.exit(failOn === cleanupIdentity ? 7 : 0);
