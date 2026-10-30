import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const checker = path.join(repo, 'scripts', 'check-gorm-sql-literal.mjs');

function fixture(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-gorm-sql-'));
  for (const [relative, content] of Object.entries(files)) {
    const target = path.join(root, relative);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, content);
  }
  return root;
}

function run(root) {
  try {
    const stdout = execFileSync(process.execPath, [checker, '--root', root], {
      cwd: repo,
      encoding: 'utf8',
      stdio: 'pipe',
    });
    return { passed: true, output: `${stdout}` };
  } catch (error) {
    return {
      passed: false,
      output: `${error.stdout ?? ''}${error.stderr ?? ''}`,
    };
  }
}

const business = (name) => `backend/modules/business/demo/${name}`;

test('flags request values concatenated into SQL text', () => {
  const root = fixture({
    [business('svc.go')]: `package demo

func (s *Svc) List(req Query) {
	s.db.Where("name = '" + req.Name + "'").Find(&rows)
}
`,
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Where\(\) SQL text or column name must be a literal/);
});

test('flags fmt.Sprintf SQL fragments', () => {
  const root = fixture({
    [business('svc.go')]: 'package demo\nfunc (s *Svc) List(req Query) {\n\ts.db.Where(fmt.Sprintf("name = \'%s\'", req.Name)).Find(&rows)\n}\n',
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Where\(\) SQL text or column name must be a literal/);
});

test('flags dynamic sort column in Order', () => {
  const root = fixture({
    [business('svc.go')]: 'package demo\nfunc (s *Svc) List(req Query) {\n\ts.db.Order(req.SortBy + " DESC").Find(&rows)\n}\n',
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Order\(\) SQL text or column name must be a literal/);
});

test('flags a map literal keyed by a variable column', () => {
  const root = fixture({
    [business('svc.go')]: 'package demo\nfunc (s *Svc) Patch(req Query) {\n\ts.db.Model(&Row{}).Updates(map[string]any{req.Field: req.Value})\n}\n',
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Updates\(\) column name must be a literal/);
});

test('flags a map variable keyed by a request-derived column', () => {
  const root = fixture({
    [business('svc.go')]: 'package demo\nfunc (s *Svc) Patch(req Query) {\n\tupdates := map[string]any{}\n\tupdates[req.Column] = req.Value\n\ts.db.Model(&Row{}).Updates(updates)\n}\n',
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /map updates is keyed by req\.Column/);
});

test('accepts bind parameters, literal maps and package constants', () => {
  const root = fixture({
    [business('svc.go')]: [
      'package demo',
      '',
      'const condNameEquals = "name = ? AND dept_id = ?"',
      '',
      'func (s *Svc) List(req Query) {',
      '\ts.db.Where("code LIKE ?", "%"+req.Code+"%").Order("created_at DESC").Find(&rows)',
      '\ts.db.Model(&Row{}).Where(condNameEquals, req.Name, req.DeptID).Updates(map[string]any{"status": "active"})',
      '\tupdates := map[string]any{}',
      '\tupdates["owner"] = req.Owner',
      '\ts.db.Model(&Row{}).Where("id = ?", id).Updates(updates)',
      '}',
      '',
    ].join('\n'),
  });
  const result = run(root);
  assert.equal(result.passed, true, result.output);
});

test('ignores same-named non-GORM methods', () => {
  const root = fixture({
    [business('svc.go')]: [
      'package demo',
      '',
      'func (s *Svc) Apply(ctx context.Context, item *appsv1.Deployment) {',
      '\ts.clientset.AppsV1().Deployments(ns).Update(ctx, item, metav1.UpdateOptions{})',
      '\ts.handler.Update(id, payload)',
      '}',
      '',
    ].join('\n'),
  });
  const result = run(root);
  assert.equal(result.passed, true, result.output);
});

test('reports line-scoped waivers so exceptions stay visible', () => {
  const root = fixture({
    [business('svc.go')]: [
      'package demo',
      '',
      'func (s *Svc) Filter(query *gorm.DB, inClause string, values []string) {',
      '\t// gorm-sql-literal:allow inClause is a package constant at every call site',
      '\tquery.Where(inClause, values)',
      '}',
      '',
    ].join('\n'),
  });
  const result = run(root);
  assert.equal(result.passed, true, result.output);
  assert.match(result.output, /waived .*: inClause is a package constant at every call site/);
});

test('rejects a waiver without a written justification', () => {
  const root = fixture({
    [business('svc.go')]: [
      'package demo',
      '',
      'func (s *Svc) Filter(query *gorm.DB, column string, values []string) {',
      '\t// gorm-sql-literal:allow',
      '\tquery.Where(column+" IN ?", values)',
      '}',
      '',
    ].join('\n'),
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Where\(\) SQL text or column name must be a literal/);
});

test('flags dynamic SQL text reaching Exec and a dynamic table name', () => {
  const root = fixture({
    [business('svc.go')]: [
      'package demo',
      '',
      'func (s *Svc) Purge(req Query) {',
      '\ts.db.Exec("DELETE FROM biz_row WHERE scope = \'" + req.Scope + "\'")',
      '\ts.db.Table(req.Table).Count(&count)',
      '}',
      '',
    ].join('\n'),
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Exec\(\) SQL text or column name must be a literal/);
  assert.match(result.output, /Table\(\) SQL text or column name must be a literal/);
});

test('does not treat a reassigned or parameter-shadowed name as a package constant', () => {
  const root = fixture({
    [business('a.go')]: [
      'package demo',
      '',
      'const orderCol = "id DESC"',
      'const condWhere = "name = ?"',
      '',
      'func (s *Svc) Sorted(db *gorm.DB, req Query) {',
      '\torderCol := req.Sort',
      '\tdb.Order(orderCol).Find(&rows)',
      '}',
      '',
    ].join('\n'),
    [business('b.go')]: [
      'package demo',
      '',
      'func (s *Svc) Filtered(db *gorm.DB, condWhere string, values []string) {',
      '\tdb.Where(condWhere, values)',
      '}',
      '',
    ].join('\n'),
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Order\(\) SQL text or column name must be a literal/);
  assert.match(result.output, /Where\(\) SQL text or column name must be a literal/);
});

test('recognizes database handles that are not literally named db', () => {
  const root = fixture({
    [business('svc.go')]: [
      'package demo',
      '',
      'func (s *Svc) Page(permissionQuery *gorm.DB, req Query) {',
      '\tpermissionQuery.Order(req.Sort).Find(&rows)',
      '}',
      '',
    ].join('\n'),
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /Order\(\) SQL text or column name must be a literal/);
});

test('passes the real business modules and documents the S3649 review', () => {
  const result = run(repo);
  assert.equal(result.passed, true, result.output);
});
