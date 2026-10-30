import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const checker = path.join(repo, 'scripts', 'check-gorm-sql-literal.mjs');

// Each case is a package directory plus the expectation on the checker output. `sources` is
// keyed by file name; all files land in one demo package so cross-file shadowing is exercised.
const CASES = [
  {
    title: 'flags request values concatenated into SQL text',
    sources: {
      'svc.go': `package demo

func (s *Svc) List(req Query) {
	s.db.Where("name = '" + req.Name + "'").Find(&rows)
}
`,
    },
    passed: false,
    output: [/Where\(\) SQL text or column name must be a literal/],
  },
  {
    title: 'flags fmt.Sprintf SQL fragments',
    sources: {
      'svc.go': `package demo

func (s *Svc) List(req Query) {
	s.db.Where(fmt.Sprintf("name = '%s'", req.Name)).Find(&rows)
}
`,
    },
    passed: false,
    output: [/Where\(\) SQL text or column name must be a literal/],
  },
  {
    title: 'flags dynamic sort column in Order',
    sources: {
      'svc.go': `package demo

func (s *Svc) List(req Query) {
	s.db.Order(req.SortBy + " DESC").Find(&rows)
}
`,
    },
    passed: false,
    output: [/Order\(\) SQL text or column name must be a literal/],
  },
  {
    title: 'flags a map literal keyed by a variable column',
    sources: {
      'svc.go': `package demo

func (s *Svc) Patch(req Query) {
	s.db.Model(&Row{}).Updates(map[string]any{req.Field: req.Value})
}
`,
    },
    passed: false,
    output: [/Updates\(\) column name must be a literal/],
  },
  {
    title: 'flags a map variable keyed by a request-derived column',
    sources: {
      'svc.go': `package demo

func (s *Svc) Patch(req Query) {
	updates := map[string]any{}
	updates[req.Column] = req.Value
	s.db.Model(&Row{}).Updates(updates)
}
`,
    },
    passed: false,
    output: [/map updates is keyed by req\.Column/],
  },
  {
    title: 'flags dynamic SQL text reaching Exec and a dynamic table name',
    sources: {
      'svc.go': `package demo

func (s *Svc) Purge(req Query) {
	s.db.Exec("DELETE FROM biz_row WHERE scope = '" + req.Scope + "'")
	s.db.Table(req.Table).Count(&count)
}
`,
    },
    passed: false,
    output: [
      /Exec\(\) SQL text or column name must be a literal/,
      /Table\(\) SQL text or column name must be a literal/,
    ],
  },
  {
    title: 'recognizes database handles that are not literally named db',
    sources: {
      'svc.go': `package demo

func (s *Svc) Page(permissionQuery *gorm.DB, req Query) {
	permissionQuery.Order(req.Sort).Find(&rows)
}
`,
    },
    passed: false,
    output: [/Order\(\) SQL text or column name must be a literal/],
  },
  {
    title: 'does not treat a reassigned or parameter-shadowed name as a package constant',
    sources: {
      'a.go': `package demo

const orderCol = "id DESC"
const condWhere = "name = ?"

func (s *Svc) Sorted(db *gorm.DB, req Query) {
	orderCol := req.Sort
	db.Order(orderCol).Find(&rows)
}
`,
      'b.go': `package demo

func (s *Svc) Filtered(db *gorm.DB, condWhere string, values []string) {
	db.Where(condWhere, values)
}
`,
    },
    passed: false,
    output: [
      /Order\(\) SQL text or column name must be a literal/,
      /Where\(\) SQL text or column name must be a literal/,
    ],
  },
  {
    title: 'rejects a waiver without a written justification',
    sources: {
      'svc.go': `package demo

func (s *Svc) Filter(query *gorm.DB, column string, values []string) {
	// gorm-sql-literal:allow
	query.Where(column+" IN ?", values)
}
`,
    },
    passed: false,
    output: [/Where\(\) SQL text or column name must be a literal/],
  },
  {
    title: 'accepts bind parameters, literal maps and package constants',
    sources: {
      'svc.go': `package demo

const condNameEquals = "name = ? AND dept_id = ?"

func (s *Svc) List(req Query) {
	s.db.Where("code LIKE ?", "%"+req.Code+"%").Order("created_at DESC").Find(&rows)
	s.db.Model(&Row{}).Where(condNameEquals, req.Name, req.DeptID).Updates(map[string]any{"status": "active"})
	updates := map[string]any{}
	updates["owner"] = req.Owner
	s.db.Model(&Row{}).Where("id = ?", id).Updates(updates)
}
`,
    },
    passed: true,
  },
  {
    title: 'ignores same-named non-GORM methods',
    sources: {
      'svc.go': `package demo

func (s *Svc) Apply(ctx context.Context, item *appsv1.Deployment) {
	s.clientset.AppsV1().Deployments(ns).Update(ctx, item, metav1.UpdateOptions{})
	s.handler.Update(id, payload)
}
`,
    },
    passed: true,
  },
  {
    title: 'reports line-scoped waivers so exceptions stay visible',
    sources: {
      'svc.go': `package demo

func (s *Svc) Filter(query *gorm.DB, inClause string, values []string) {
	// gorm-sql-literal:allow inClause is a package constant at every call site
	query.Where(inClause, values)
}
`,
    },
    passed: true,
    output: [/waived .*: inClause is a package constant at every call site/],
  },
];

function writePackage(sources) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-gorm-sql-'));
  const dir = path.join(root, 'backend', 'modules', 'business', 'demo');
  fs.mkdirSync(dir, { recursive: true });
  for (const [name, content] of Object.entries(sources)) {
    fs.writeFileSync(path.join(dir, name), content);
  }
  return root;
}

function runChecker(root) {
  try {
    const stdout = execFileSync(process.execPath, [checker, '--root', root], {
      cwd: repo,
      encoding: 'utf8',
      stdio: 'pipe',
    });
    return { passed: true, output: `${stdout}` };
  } catch (error) {
    return { passed: false, output: `${error.stdout ?? ''}${error.stderr ?? ''}` };
  }
}

for (const { title, sources, passed, output = [] } of CASES) {
  test(title, () => {
    const result = runChecker(writePackage(sources));
    assert.equal(result.passed, passed, result.output);
    for (const pattern of output) assert.match(result.output, pattern);
  });
}

test('passes the real business modules and documents the S3649 review', () => {
  const result = runChecker(repo);
  assert.equal(result.passed, true, result.output);
});
