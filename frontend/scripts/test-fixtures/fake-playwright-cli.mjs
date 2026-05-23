import fs from 'node:fs';
import process from 'node:process';

const markerPath = process.env.PANTHEON_FAKE_PLAYWRIGHT_MARKER;

if (markerPath) {
  fs.writeFileSync(
    markerPath,
    JSON.stringify({
      args: process.argv.slice(2),
      baseUrl: process.env.PANTHEON_WEB_BASE_URL ?? null,
      outputDir: process.env.PANTHEON_PLAYWRIGHT_OUTPUT_DIR ?? null,
    }),
  );
}

process.exit(0);
