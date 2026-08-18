import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const defaultRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const rootArgumentIndex = process.argv.indexOf('--root');
const root = rootArgumentIndex >= 0
  ? path.resolve(process.cwd(), process.argv[rootArgumentIndex + 1])
  : defaultRoot;

// These tokens describe owner boundaries, not implementation details. Keep the
// list narrow and review every addition with the owning module.
const boundaryRules = [
  {
    module: 'bizscope',
    forbidden: [
      'biz_cmdb_host',
      'modules/business/cmdb/host',
      'cmdbhost.Host',
    ],
  },
  {
    module: 'cmdb',
    forbidden: [
      'biz_business_scope',
      'modules/business/bizscope/bizscope_model',
      'bizscope.BizScope',
    ],
  },
  {
    module: 'deploy',
    forbidden: [
      'biz_business_scope',
      'biz_cmdb_host',
      'cmdbhost.Host',
      'bizscope.BizScope',
    ],
  },
  {
    module: 'k8s',
    forbidden: [
      'biz_business_scope',
      'bizscope.BizScope',
      'modules/business/bizscope/bizscope_model',
    ],
  },
];

const allowlist = [
  {
    suffix: '_model.go',
    owner: 'module owner',
    reason: 'owner model declares its own table name',
    review: 'review when table ownership changes',
  },
  {
    suffix: '_seed.go',
    owner: 'module owner',
    reason: 'seed/bootstrap data is not request-path access',
    review: 'review when seed wiring changes',
  },
  {
    suffix: '_test.go',
    owner: 'test owner',
    reason: 'isolated fixtures may create cross-module test tables',
    review: 'review when fixture reaches production wiring',
  },
];

function normalize(value) {
  return value.replaceAll('\\', '/');
}

function goFiles(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) return goFiles(file);
    return entry.name.endsWith('.go') ? [file] : [];
  });
}

function isAllowlisted(file) {
  const name = path.basename(file);
  return allowlist.find((entry) => name.endsWith(entry.suffix));
}

function scanBoundaryRules() {
  const violations = [];
  for (const rule of boundaryRules) {
    const dir = path.join(root, 'backend', 'modules', 'business', rule.module);
    for (const file of goFiles(dir)) {
      if (isAllowlisted(file)) continue;
      const lines = fs.readFileSync(file, 'utf8').split(/\r?\n/);
      lines.forEach((line, index) => {
        for (const token of rule.forbidden) {
          if (line.includes(token)) {
            violations.push(
              `${normalize(path.relative(root, file))}:${index + 1}: forbidden cross-module access ${token}`,
            );
          }
        }
      });
    }
  }
  return violations;
}

function scanCapabilityContracts() {
  const violations = [];
  const capabilityRoot = path.join(root, 'backend', 'modules', 'business', 'capability');
  for (const file of goFiles(capabilityRoot)) {
    if (isAllowlisted(file)) continue;
    const lines = fs.readFileSync(file, 'utf8').split(/\r?\n/);
    lines.forEach((line, index) => {
      if (line.includes('gorm.io/gorm') || line.includes('*gorm.DB') || line.includes('gorm.Model')) {
        violations.push(
          `${normalize(path.relative(root, file))}:${index + 1}: capability DTOs must not expose GORM types`,
        );
      }
    });
  }
  return violations;
}

const violations = [...scanBoundaryRules(), ...scanCapabilityContracts()];
if (violations.length > 0) {
  console.error(violations.join('\n'));
  process.exit(1);
}

console.log('business module boundary check passed');
