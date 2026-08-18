import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const checker = path.join(repo, 'scripts', 'check-business-module-boundaries.mjs');

function fixture(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-boundary-'));
  for (const [relative, content] of Object.entries(files)) {
    const target = path.join(root, relative);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, content);
  }
  return root;
}

function run(root) {
  try {
    execFileSync(process.execPath, [checker, '--root', root], {
      cwd: repo,
      encoding: 'utf8',
      stdio: 'pipe',
    });
    return { passed: true, output: '' };
  } catch (error) {
    return {
      passed: false,
      output: `${error.stdout ?? ''}${error.stderr ?? ''}`,
    };
  }
}

test('fails forbidden cross-module request-path access', () => {
  const root = fixture({
    'backend/modules/business/deploy/deploy_service.go': 'package deploy\nconst table = "biz_cmdb_host"\n',
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /forbidden cross-module access biz_cmdb_host/);
});

test('fails raw scope table access in K8s and BizScope host mutation', () => {
  const root = fixture({
    'backend/modules/business/k8s/cluster_service.go': 'package k8s\nfunc x() { db.Table("biz_business_scope") }\n',
    'backend/modules/business/bizscope/bizscope_service.go': 'package bizscope\nfunc x() { db.Table("biz_cmdb_host") }\n',
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /k8s[\\/]cluster_service\.go/);
  assert.match(result.output, /bizscope[\\/]bizscope_service\.go/);
});

test('fails capability contracts that expose GORM types', () => {
  const root = fixture({
    'backend/modules/business/capability/capability.go': 'package capability\nimport "gorm.io/gorm"\ntype DTO struct { DB *gorm.DB }\n',
  });
  const result = run(root);
  assert.equal(result.passed, false);
  assert.match(result.output, /capability DTOs must not expose GORM types/);
});

test('passes owner access and explicitly allowlisted fixtures', () => {
  const root = fixture({
    'backend/modules/business/cmdb/host/host_service.go': 'package host\nfunc x() { db.Model(&Host{}) }\n',
    'backend/modules/business/cmdb/host/host_model.go': 'package host\nfunc (Host) TableName() string { return "biz_cmdb_host" }\n',
    'backend/modules/business/cmdb/host/host_service_test.go': 'package host\nconst fixture = "biz_business_scope"\n',
  });
  const result = run(root);
  assert.equal(result.passed, true, result.output);
});
