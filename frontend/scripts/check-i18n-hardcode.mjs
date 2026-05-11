import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const sourceRoot = path.join(frontendRoot, 'src');

const TARGET_EXTENSIONS = new Set(['.ts', '.tsx']);
const EXCLUDED_SEGMENTS = [
  `${path.sep}i18n${path.sep}resources${path.sep}`,
];

const PROP_PATTERN = /\b(title|label|placeholder|content|description|message|okText|cancelText|header|footer|emptyText)\s*:\s*(['"`])([\s\S]*?)\2/g;
const DEFAULT_VALUE_PATTERN = /\bdefaultValue\s*:\s*(['"`])([\s\S]*?)\1/g;
const LINE_PUSH_PATTERN = /\b(lines\.push|Message\.(?:error|success|warning|info)|Notification\.(?:error|success|warning|info))\s*\(\s*(['"`])([\s\S]*?)\2/g;
const TRANSLATION_KEY_PATTERN = /^[a-z0-9_]+(?:\.[a-z0-9_]+)+$/i;
const FILE_NAME_PATTERN = /^[a-z0-9._/-]+\.(csv|zip|json|txt)$/i;
const TECHNICAL_TOKEN_PATTERN = /^[a-z0-9:_./-]+$/i;

function walkFiles(dirPath, bucket = []) {
  for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
    const nextPath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) {
      walkFiles(nextPath, bucket);
      continue;
    }
    if (TARGET_EXTENSIONS.has(path.extname(entry.name))) {
      bucket.push(nextPath);
    }
  }
  return bucket;
}

function shouldSkip(filePath) {
  return EXCLUDED_SEGMENTS.some((segment) => filePath.includes(segment));
}

function isNaturalLanguage(text) {
  const value = String(text).trim();
  if (!value) {
    return false;
  }
  if (TRANSLATION_KEY_PATTERN.test(value)) {
    return false;
  }
  if (FILE_NAME_PATTERN.test(value)) {
    return false;
  }
  if (TECHNICAL_TOKEN_PATTERN.test(value) && !/[\p{Script=Han} ]/u.test(value)) {
    return false;
  }
  if (/^(system|business|auth|platform):/i.test(value)) {
    return false;
  }
  if (/^\/[a-z0-9/_:-]+$/i.test(value)) {
    return false;
  }
  return /[\p{Script=Han}]|[A-Za-z]{3,}\s+[A-Za-z]{2,}/u.test(value);
}

function lineNumberAt(source, index) {
  return source.slice(0, index).split('\n').length;
}

function pushFinding(findings, filePath, source, index, text, kind) {
  findings.push({
    filePath,
    line: lineNumberAt(source, index),
    kind,
    text: String(text).trim(),
  });
}

function collectFindings(filePath) {
  const source = fs.readFileSync(filePath, 'utf8');
  const findings = [];

  for (const match of source.matchAll(PROP_PATTERN)) {
    const [, propName, , literal] = match;
    if (isNaturalLanguage(literal)) {
      pushFinding(findings, filePath, source, match.index ?? 0, `${propName}: ${literal}`, 'prop');
    }
  }

  for (const match of source.matchAll(DEFAULT_VALUE_PATTERN)) {
    const [, literal] = match;
    if (isNaturalLanguage(literal)) {
      pushFinding(findings, filePath, source, match.index ?? 0, `defaultValue: ${literal}`, 'defaultValue');
    }
  }

  for (const match of source.matchAll(LINE_PUSH_PATTERN)) {
    const [, callee, , literal] = match;
    if (isNaturalLanguage(literal)) {
      pushFinding(findings, filePath, source, match.index ?? 0, `${callee}(${literal})`, 'runtime-message');
    }
  }

  return findings;
}

const files = walkFiles(sourceRoot).filter((filePath) => !shouldSkip(filePath));
const findings = files.flatMap((filePath) => collectFindings(filePath));

if (findings.length > 0) {
  console.error('❌ 检测到疑似展示型硬编码文案，请改为 i18n key 或 t() 调用：');
  for (const finding of findings) {
    const relativePath = path.relative(frontendRoot, finding.filePath).replaceAll('\\', '/');
    console.error(`- ${relativePath}:${finding.line} [${finding.kind}] ${finding.text}`);
  }
  process.exit(1);
}

console.log(`✅ 展示型硬编码扫描通过（检查 ${files.length} 个文件）`);
