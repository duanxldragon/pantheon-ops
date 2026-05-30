import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const DOC_ROOT = 'docs';
const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_REPO_ROOT = path.resolve(SCRIPT_DIR, '..');

const REQUIRED_BASE_FIELDS = ['title', 'doc_type', 'layer', 'status', 'updated_at'];
const REQUIRED_RETAINED_FIELDS = ['index_group', 'retention_reason', 'linked_contracts'];
const DOC_TYPES_REQUIRING_CONTRACTS = new Set([
  'Design',
  'Assessment',
  'Remediation',
  'Acceptance',
]);
const ALLOWED_DOC_TYPES = new Set([
  'Contract',
  'Design',
  'Assessment',
  'Remediation',
  'Acceptance',
]);
const ALLOWED_STATUSES = new Set([
  'Draft',
  'Active',
  'Approved',
  'Superseded',
  'Archived',
]);
const CONTRACT_RELATION_FIELDS = [
  { heading: '关联设计：', field: 'related_designs' },
  { heading: '关联评估：', field: 'related_assessments' },
  { heading: '关联整改：', field: 'related_remediations' },
  { heading: '关联验收：', field: 'related_acceptances' },
];

function walkMarkdownFiles(dirPath) {
  if (!fs.existsSync(dirPath)) return [];

  const entries = fs.readdirSync(dirPath, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const fullPath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) {
      files.push(...walkMarkdownFiles(fullPath));
      continue;
    }
    if (entry.isFile() && entry.name.endsWith('.md')) {
      files.push(fullPath);
    }
  }
  return files;
}

export function hasLegacyMetadata(source) {
  return /^(更新时间：|类型：|归属层：|主层：|状态：)/m.test(source);
}

export function parseFrontmatter(source) {
  const lines = source.split(/\r?\n/);
  if (lines[0] !== '---') {
    return { data: null, body: source, hasFrontmatter: false };
  }

  const closingIndex = lines.findIndex((line, index) => index > 0 && line === '---');
  if (closingIndex === -1) {
    return { data: null, body: source, hasFrontmatter: false };
  }

  const frontmatterLines = lines.slice(1, closingIndex);
  const data = {};
  let currentArrayKey = null;

  for (const line of frontmatterLines) {
    if (!line.trim()) continue;

    const arrayItemMatch = line.match(/^\s*-\s+(.*)$/);
    if (arrayItemMatch) {
      if (!currentArrayKey) {
        throw new Error(`Array item found before array key: ${line}`);
      }
      data[currentArrayKey].push(arrayItemMatch[1].trim());
      continue;
    }

    const keyMatch = line.match(/^([A-Za-z0-9_]+):\s*(.*)$/);
    if (!keyMatch) {
      throw new Error(`Unsupported frontmatter line: ${line}`);
    }

    const [, key, rawValue] = keyMatch;
    if (rawValue === '') {
      data[key] = [];
      currentArrayKey = key;
      continue;
    }

    data[key] = rawValue.trim();
    currentArrayKey = null;
  }

  return {
    data,
    body: lines.slice(closingIndex + 1).join('\n'),
    hasFrontmatter: true,
  };
}

function expectedIndexGroup(filePath) {
  const normalized = filePath.replace(/\\/g, '/');
  if (normalized.startsWith('docs/superpowers/specs/')) return 'superpowers-specs';
  if (normalized.startsWith('docs/archive/examples/')) return 'archive/examples';
  if (normalized.startsWith('docs/archive/baselines/')) return 'archive/baselines';
  if (normalized.startsWith('docs/archive/upgrade/')) return 'archive/upgrade';
  return null;
}

function expectedStatuses(filePath) {
  const normalized = filePath.replace(/\\/g, '/');
  if (normalized.startsWith('docs/superpowers/specs/')) return new Set(['Approved', 'Superseded']);
  if (normalized.startsWith('docs/archive/examples/')) return new Set(['Archived']);
  if (normalized.startsWith('docs/archive/baselines/')) return new Set(['Archived', 'Superseded']);
  if (normalized.startsWith('docs/archive/upgrade/')) return new Set(['Archived']);
  return null;
}

function isNonEmptyString(value) {
  return typeof value === 'string' && value.trim().length > 0;
}

function isNonEmptyArray(value) {
  return Array.isArray(value) && value.length > 0 && value.every(isNonEmptyString);
}

function normalizeSlash(p) {
  return p.replace(/\\/g, '/');
}

function buildDocsIndex(repoRoot) {
  const files = walkMarkdownFiles(path.resolve(repoRoot, DOC_ROOT));
  const byPath = new Map();
  const byBasename = new Map();

  for (const filePath of files) {
    const relativePath = normalizeSlash(path.relative(repoRoot, filePath));
    const source = fs.readFileSync(filePath, 'utf8');
    let parsed;
    try {
      parsed = parseFrontmatter(source);
    } catch {
      parsed = { hasFrontmatter: false, data: null, body: source };
    }

    const entry = {
      path: relativePath,
      basename: path.basename(relativePath),
      source,
      parsed,
    };
    byPath.set(relativePath, entry);
    const existing = byBasename.get(entry.basename) ?? [];
    existing.push(entry);
    byBasename.set(entry.basename, existing);
  }

  return { files, byPath, byBasename };
}

function resolveContractBodyReference(ref, docsIndex) {
  if (ref.startsWith('docs/')) return docsIndex.byPath.get(ref) ?? null;
  const basenameMatches = docsIndex.byBasename.get(ref) ?? [];
  if (basenameMatches.length === 1) return basenameMatches[0];
  return null;
}

export function parseContractBodyReferences(source) {
  const refs = [];
  const byField = {
    related_designs: [],
    related_assessments: [],
    related_remediations: [],
    related_acceptances: [],
  };
  const lines = source.split(/\r?\n/);

  for (let i = 0; i < lines.length; i += 1) {
    const relation = CONTRACT_RELATION_FIELDS.find(({ heading }) => heading === lines[i].trim());
    if (!relation) continue;
    for (let j = i + 1; j < lines.length; j += 1) {
      const line = lines[j].trim();
      if (!line) break;
      const match = line.match(/^-\s+`([^`]+)`$/);
      if (!match) break;
      refs.push(match[1]);
      byField[relation.field].push(match[1]);
    }
  }

  return { all: refs, byField };
}

export function extractReadmeDocLinks(source) {
  const links = [];
  const regex = /\[[^\]]+\]\(([^)]+)\)/g;
  let match;
  while ((match = regex.exec(source)) !== null) {
    const href = match[1].trim();
    if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('#')) continue;
    links.push(href);
  }
  return links;
}

export function extractReadmeMainEntryLinks(source) {
  const lines = source.split(/\r?\n/);
  const captured = [];
  let inMainEntry = false;

  for (const line of lines) {
    if (/^## 2\./.test(line) || /^## 3\./.test(line) || /^## 4\./.test(line)) {
      inMainEntry = true;
    } else if (/^## 5\./.test(line) || /^## 6\./.test(line)) {
      inMainEntry = false;
    }
    if (!inMainEntry) continue;
    for (const href of extractReadmeDocLinks(line)) {
      captured.push(href);
    }
  }

  return captured;
}

export function validateDoc({ filePath, data, repoRoot }) {
  const errors = [];

  for (const field of REQUIRED_BASE_FIELDS) {
    if (!isNonEmptyString(data[field])) {
      errors.push(`${filePath}: missing or empty required field "${field}"`);
    }
  }

  if (data.doc_type && !ALLOWED_DOC_TYPES.has(data.doc_type)) {
    errors.push(`${filePath}: doc_type "${data.doc_type}" is not in the allowed set`);
  }

  if (data.status && !ALLOWED_STATUSES.has(data.status)) {
    errors.push(`${filePath}: status "${data.status}" is not in the allowed set`);
  }

  const expectedGroup = expectedIndexGroup(filePath);
  if (expectedGroup) {
    for (const field of REQUIRED_RETAINED_FIELDS) {
      if (field === 'linked_contracts') {
        if (!isNonEmptyArray(data[field])) {
          errors.push(`${filePath}: missing or empty required field "${field}"`);
        }
      } else if (!isNonEmptyString(data[field])) {
        errors.push(`${filePath}: missing or empty required field "${field}"`);
      }
    }

    if (data.index_group && data.index_group !== expectedGroup) {
      errors.push(`${filePath}: index_group "${data.index_group}" does not match expected "${expectedGroup}"`);
    }

    const allowedStatuses = expectedStatuses(filePath);
    if (data.status && allowedStatuses && !allowedStatuses.has(data.status)) {
      errors.push(`${filePath}: status "${data.status}" is not allowed for this directory`);
    }
  }

  if (data.linked_contracts !== undefined) {
    if (!Array.isArray(data.linked_contracts)) {
      errors.push(`${filePath}: linked_contracts must be a YAML array`);
    } else {
      for (const linkedPath of data.linked_contracts) {
        const absolute = path.resolve(repoRoot, linkedPath);
        if (!fs.existsSync(absolute)) {
          errors.push(`${filePath}: linked contract path does not exist: ${linkedPath}`);
        }
      }
    }
  }

  if (DOC_TYPES_REQUIRING_CONTRACTS.has(data.doc_type) && !isNonEmptyArray(data.linked_contracts)) {
    errors.push(`${filePath}: doc_type "${data.doc_type}" requires non-empty linked_contracts`);
  }

  if (data.status === 'Superseded' && !isNonEmptyString(data.superseded_by)) {
    errors.push(`${filePath}: status "Superseded" requires non-empty superseded_by`);
  }

  return {
    ok: errors.length === 0,
    errors,
  };
}

function checkFile(filePath, repoRoot) {
  const source = fs.readFileSync(filePath, 'utf8');
  const relativePath = path.relative(repoRoot, filePath).replace(/\\/g, '/');

  let parsed;
  try {
    parsed = parseFrontmatter(source);
  } catch (error) {
    return {
      ok: false,
      errors: [`${relativePath}: failed to parse frontmatter: ${error.message}`],
    };
  }

  if (!parsed.hasFrontmatter) {
    const expectedGroup = expectedIndexGroup(relativePath);
    if (expectedGroup) {
      return {
        ok: false,
        errors: [`${relativePath}: missing YAML frontmatter block`],
        legacy: hasLegacyMetadata(source),
      };
    }

    return {
      ok: true,
      errors: [],
      legacy: hasLegacyMetadata(source),
    };
  }

  const validation = validateDoc({
    filePath: relativePath,
    data: parsed.data ?? {},
    repoRoot,
  });

  return {
    ...validation,
    legacy: false,
  };
}

export function runCheck(repoRoot = DEFAULT_REPO_ROOT, options = {}) {
  const docsIndex = buildDocsIndex(repoRoot);
  const files = docsIndex.files;
  const errors = [];
  const legacyFiles = [];
  let frontmatterFiles = 0;

  for (const filePath of files) {
    const result = checkFile(filePath, repoRoot);
    const relativePath = normalizeSlash(path.relative(repoRoot, filePath));
    const source = docsIndex.byPath.get(relativePath)?.source ?? fs.readFileSync(filePath, 'utf8');
    const parsed = docsIndex.byPath.get(relativePath)?.parsed ?? parseFrontmatter(source);
    if (parsed.hasFrontmatter) {
      frontmatterFiles += 1;
    }
    if (!result.ok) {
      errors.push(...result.errors);
    }
    if (result.legacy) {
      legacyFiles.push(path.relative(repoRoot, filePath).replace(/\\/g, '/'));
    }
  }

  for (const [relativePath, entry] of docsIndex.byPath.entries()) {
    if (!entry.parsed.hasFrontmatter) continue;
    if (entry.parsed.data?.doc_type !== 'Contract') continue;
    if (entry.parsed.data?.status === 'Draft') continue;

    const relationRefs = parseContractBodyReferences(entry.source);
    for (const { field } of CONTRACT_RELATION_FIELDS) {
      const bodyRefs = relationRefs.byField[field].filter((ref) => ref !== 'TBD');
      const fmRefs = Array.isArray(entry.parsed.data?.[field]) ? entry.parsed.data[field] : [];
      const resolvedBody = bodyRefs
        .map((ref) => resolveContractBodyReference(ref, docsIndex)?.path ?? null)
        .filter(Boolean)
        .sort((a, b) => a.localeCompare(b));
      const resolvedFm = fmRefs
        .map((ref) => resolveContractBodyReference(ref, docsIndex)?.path ?? ref)
        .filter(Boolean)
        .sort((a, b) => a.localeCompare(b));
      if (resolvedBody.join('|') !== resolvedFm.join('|')) {
        errors.push(`${relativePath}: frontmatter field "${field}" does not match contract body references`);
      }
    }

    for (const ref of relationRefs.all) {
      if (ref === 'TBD') continue;
      const target = resolveContractBodyReference(ref, docsIndex);
      if (!target) {
        errors.push(`${relativePath}: contract reference cannot be resolved: ${ref}`);
        continue;
      }
      if (!target.parsed.hasFrontmatter) {
        errors.push(`${relativePath}: referenced doc has no frontmatter: ${target.path}`);
        continue;
      }
      const linked = target.parsed.data?.linked_contracts;
      if (!Array.isArray(linked) || !linked.includes(relativePath)) {
        errors.push(`${relativePath}: referenced doc does not link back via linked_contracts: ${target.path}`);
      }
    }
  }

  const readmeEntry = docsIndex.byPath.get('docs/README.md');
  if (readmeEntry) {
    for (const href of extractReadmeDocLinks(readmeEntry.source)) {
      const baseDir = path.resolve(repoRoot, 'docs');
      const resolved = normalizeSlash(path.relative(repoRoot, path.resolve(baseDir, href)));
      if (!docsIndex.byPath.has(resolved) && !fs.existsSync(path.resolve(repoRoot, resolved))) {
        errors.push(`docs/README.md: link target does not exist: ${href}`);
      }
    }

    for (const href of extractReadmeMainEntryLinks(readmeEntry.source)) {
      const baseDir = path.resolve(repoRoot, 'docs');
      const resolved = normalizeSlash(path.relative(repoRoot, path.resolve(baseDir, href)));
      const target = docsIndex.byPath.get(resolved);
      if (!target?.parsed?.hasFrontmatter) continue;
      const status = target.parsed.data?.status;
      if (status && status !== 'Active') {
        errors.push(`docs/README.md: main entry link must target Active docs only: ${href} (${status})`);
      }
    }
  }

  return {
    ok: errors.length === 0,
    checkedFiles: files.length,
    frontmatterFiles,
    legacyFiles,
    errors,
  };
}

const isDirectRun =
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url);

if (isDirectRun) {
  const showLegacy = process.argv.includes('--report-legacy');
  const result = runCheck();
  if (!result.ok) {
    console.error(`Frontmatter check failed. Checked ${result.checkedFiles} docs, ${result.frontmatterFiles} with frontmatter.`);
    for (const error of result.errors) {
      console.error(`- ${error}`);
    }
    if (showLegacy && result.legacyFiles.length > 0) {
      console.error('Legacy metadata docs not yet migrated:');
      for (const file of result.legacyFiles) {
        console.error(`- ${file}`);
      }
    }
    process.exit(1);
  }

  console.log(`Frontmatter check passed. Checked ${result.checkedFiles} docs, ${result.frontmatterFiles} with frontmatter.`);
  if (showLegacy) {
    console.log(`Legacy metadata docs not yet migrated: ${result.legacyFiles.length}`);
    for (const file of result.legacyFiles) {
      console.log(`- ${file}`);
    }
  }
}
