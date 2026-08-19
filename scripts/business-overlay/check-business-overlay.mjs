import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const defaultRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const rootArgumentIndex = process.argv.indexOf('--root');
const root = rootArgumentIndex >= 0
  ? path.resolve(process.cwd(), process.argv[rootArgumentIndex + 1])
  : defaultRoot;
const manifest = JSON.parse(fs.readFileSync(path.join(root, 'business-overlay.json'), 'utf8'));
const isRebuiltTree = fs.existsSync(path.join(root, '.business-overlay-report.json'));
const forbiddenModule = isRebuiltTree ? manifest.sourceModule : manifest.base.module;
const required = isRebuiltTree ? [
  'backend/modules/business/business_overlay_registry.go',
  'frontend/src/modules/businessOverlay.ts',
  'frontend/src/core/router/businessOverlayComponentRegistry.ts',
  'backend/modules/system/iam/menu/business_overlay_component_registry.go',
] : [
  'backend/modules/business/generated_registry.go',
  'frontend/src/modules/generated/business.ts',
  'frontend/src/core/router/generatedComponentRegistry.ts',
  'backend/modules/system/iam/menu/generated_component_registry.go',
];
const missing = required.filter((file) => !fs.existsSync(path.join(root, file)));
const staleImports = [];
for (const declaredPath of manifest.businessPaths) {
  const absolute = path.join(root, declaredPath);
  if (!fs.existsSync(absolute)) missing.push(declaredPath);
}
const backendRoot = path.join(root, 'backend', 'modules', 'business');
const visit = (directory) => {
  if (!fs.existsSync(directory)) return;
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) visit(absolute);
    else if (entry.name.endsWith('.go') && fs.readFileSync(absolute, 'utf8').includes(`${forbiddenModule}/`)) {
      staleImports.push(path.relative(root, absolute).replaceAll('\\', '/'));
    }
  }
};
visit(backendRoot);
if (missing.length || staleImports.length) {
  for (const file of missing) console.error(`MISSING ${file}`);
  for (const file of staleImports) console.error(`STALE_IMPORT ${file}`);
  process.exitCode = 1;
} else {
  console.log(`OK business overlay contract (${manifest.businessPaths.length} declared paths, ${isRebuiltTree ? 'rebuilt' : 'source'} tree)`);
}
