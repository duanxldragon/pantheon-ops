import fs from 'node:fs';
import process from 'node:process';

const markerPath = process.env.PANTHEON_FAKE_SETUP_MARKER;
const action = process.argv[2] || 'up';

if (markerPath) {
  const current = fs.existsSync(markerPath)
    ? JSON.parse(fs.readFileSync(markerPath, 'utf8'))
    : [];
  current.push(action);
  fs.writeFileSync(markerPath, JSON.stringify(current));
}

process.exit(0);
