import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { after, test } from 'node:test';
import {
  buildMysqlPasswordCandidates,
  parseDsn,
  readEnvFile,
  resolveMysqlConfig,
} from './database-import-qa-setup.mjs';

const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-db-import-setup-'));

after(() => {
  fs.rmSync(tmpRoot, { recursive: true, force: true });
});

test('parseDsn extracts mysql credentials and target database', () => {
  assert.deepEqual(
    parseDsn('root:secret@tcp(127.0.0.1:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local'),
    {
      host: '127.0.0.1',
      port: 3306,
      username: 'root',
      password: 'secret',
      database: 'pantheon_base',
    },
  );
});

test('readEnvFile parses key-value pairs and ignores comments', () => {
  const envPath = path.join(tmpRoot, 'sample.env');
  fs.writeFileSync(
    envPath,
    [
      '# comment',
      'MYSQL_ROOT_PASSWORD=dev_password_change_me',
      'PANTHEON_DSN="root:quoted@tcp(localhost:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local"',
      '',
    ].join('\n'),
    'utf8',
  );

  assert.deepEqual(readEnvFile(envPath), {
    MYSQL_ROOT_PASSWORD: 'dev_password_change_me',
    PANTHEON_DSN:
      'root:quoted@tcp(localhost:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local',
  });
});

test('resolveMysqlConfig falls back to local env root password when runtime env is absent', () => {
  const repoRoot = path.join(tmpRoot, 'repo');
  fs.mkdirSync(repoRoot, { recursive: true });
  fs.writeFileSync(
    path.join(repoRoot, '.env'),
    'MYSQL_ROOT_PASSWORD=dev_password_change_me\n',
    'utf8',
  );

  const config = resolveMysqlConfig({}, { repoRoot });
  assert.equal(config.host, '127.0.0.1');
  assert.equal(config.port, 3306);
  assert.equal(config.username, 'root');
  assert.equal(config.database, 'pantheon_base');
  assert.equal(config.password, 'dev_password_change_me');
  assert.equal(config.hasExplicitPassword, true);
});

test('resolveMysqlConfig prefers explicit smoke env over local defaults', () => {
  const repoRoot = path.join(tmpRoot, 'repo-explicit');
  fs.mkdirSync(repoRoot, { recursive: true });
  fs.writeFileSync(path.join(repoRoot, '.env'), 'MYSQL_ROOT_PASSWORD=dev_password_change_me\n', 'utf8');

  const config = resolveMysqlConfig(
    {
      PANTHEON_SMOKE_MYSQL_HOST: 'db.internal',
      PANTHEON_SMOKE_MYSQL_PORT: '3307',
      PANTHEON_SMOKE_MYSQL_USER: 'smoke',
      PANTHEON_SMOKE_MYSQL_PASSWORD: 'override-secret',
      PANTHEON_SMOKE_MYSQL_DATABASE: 'pantheon_smoke',
      PANTHEON_SMOKE_MYSQL_BIN: 'custom-mysql',
    },
    { repoRoot },
  );

  assert.equal(config.host, 'db.internal');
  assert.equal(config.port, 3307);
  assert.equal(config.username, 'smoke');
  assert.equal(config.password, 'override-secret');
  assert.equal(config.database, 'pantheon_smoke');
  assert.equal(config.mysqlBin, 'custom-mysql');
  assert.equal(config.hasExplicitPassword, true);
});

test('buildMysqlPasswordCandidates falls back to documented dev passwords when no password is configured', () => {
  assert.deepEqual(
    buildMysqlPasswordCandidates({
      hasExplicitPassword: false,
      password: '',
    }),
    ['DHCCroot@2025', 'dev_password_change_me', ''],
  );
});
