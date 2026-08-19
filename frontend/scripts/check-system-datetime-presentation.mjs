import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const frontendRoot = path.resolve(__dirname, '..');
const scanRoots = [
  path.resolve(frontendRoot, 'src/modules/system'),
  path.resolve(frontendRoot, 'src/modules/auth'),
];

const timeFields = [
  'createdAt',
  'updatedAt',
  'operTime',
  'loginTime',
  'logoutTime',
  'lastActiveAt',
  'lastActivityAt',
  'lastRefreshAt',
  'refreshExpiresAt',
  'expiresAt',
  'expireTime',
  'acknowledgedAt',
  'lastItemUpdatedAt',
  'lifecycleMarkedAt',
];

const timeFieldPattern = new RegExp(`\\b(${timeFields.join('|')})\\b`);
const propertyAccess = String.raw`[A-Za-z_$][\w$]*(?:\?\.)?(?:\.[A-Za-z_$][\w$]*)*`;
const directValuePattern = new RegExp(
  String.raw`value:\s*(?:${propertyAccess}\.)?(${timeFields.join('|')})(?:\s*\|\|[^,\n]+)?`,
);
const directInlinePattern = new RegExp(
  String.raw`:\s*\{(?:${propertyAccess}\.)?(${timeFields.join('|')})(?:\s*\|\|[^}\n]+)?\}`,
);
const allowedFormatterPattern = /formatDateTime\(|formatDate\(|DateTimeMeta|renderActivityTime|renderDateTime|renderDateCell/;

function walk(dirPath, files = []) {
  for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
    const absolutePath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) {
      walk(absolutePath, files);
      continue;
    }
    if (entry.isFile() && absolutePath.endsWith('.tsx')) {
      files.push(absolutePath);
    }
  }
  return files;
}

function toRelative(absolutePath) {
  return path.relative(frontendRoot, absolutePath).replaceAll('\\', '/');
}

function scanFile(absolutePath) {
  const source = fs.readFileSync(absolutePath, 'utf8');
  const lines = source.split(/\r?\n/);
  const issues = [];

  lines.forEach((line, index) => {
    if (!timeFieldPattern.test(line) || allowedFormatterPattern.test(line)) {
      return;
    }

    if (/dataIndex:\s*'[^']+'/.test(line)) {
      const block = lines.slice(index, index + 8).join('\n');
      if (!/render\s*:/.test(block)) {
        issues.push(
          `${toRelative(absolutePath)}:${index + 1} time-like table column is missing a render formatter`,
        );
      }
      return;
    }

    if (directValuePattern.test(line) || directInlinePattern.test(line)) {
      issues.push(
        `${toRelative(absolutePath)}:${index + 1} time-like value is rendered directly without a shared formatter`,
      );
    }
  });

  return issues;
}

function main() {
  const files = scanRoots.flatMap((rootPath) => walk(rootPath));
  const issues = files.flatMap((filePath) => scanFile(filePath));

  if (issues.length > 0) {
    throw new Error(`system datetime presentation check failed:\n${issues.join('\n')}`);
  }

  console.log(`system datetime presentation check passed for ${files.length} files`);
}

main();
