import fs from 'node:fs';
import { spawn } from 'node:child_process';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const fixtureTableName = 'biz_cmdb_host';
const documentedDevMysqlPasswords = ['DHCCroot@2025', 'dev_password_change_me'];
const scriptPath = fileURLToPath(import.meta.url);
const scriptDir = path.dirname(scriptPath);
const repoRoot = path.resolve(scriptDir, '../..');

function parseArgs() {
  return {
    action: process.argv[2] || 'up',
  };
}

function firstNonEmpty(...values) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim() !== '') {
      return value.trim();
    }
  }
  return '';
}

export function parseDsn(dsn) {
  const trimmed = String(dsn || '').trim();
  const marker = '@tcp(';
  const markerIndex = trimmed.indexOf(marker);
  if (markerIndex < 0) {
    return null;
  }
  const credentials = trimmed.slice(0, markerIndex);
  const separatorIndex = credentials.indexOf(':');
  if (separatorIndex < 0) {
    return null;
  }
  const username = credentials.slice(0, separatorIndex);
  const password = credentials.slice(separatorIndex + 1);
  const hostPortStart = markerIndex + marker.length;
  const hostPortEnd = trimmed.indexOf(')', hostPortStart);
  if (hostPortEnd < 0) {
    return null;
  }
  const hostPort = trimmed.slice(hostPortStart, hostPortEnd);
  const slashIndex = trimmed.indexOf('/', hostPortEnd);
  if (slashIndex < 0) {
    return null;
  }
  const queryIndex = trimmed.indexOf('?', slashIndex + 1);
  const database =
    queryIndex >= 0
      ? trimmed.slice(slashIndex + 1, queryIndex)
      : trimmed.slice(slashIndex + 1);
  const [host, portText] = hostPort.split(':');
  return {
    host: host || '127.0.0.1',
    port: Number(portText || '3306'),
    username,
    password,
    database,
  };
}

export function readEnvFile(envFilePath) {
  if (!envFilePath || !fs.existsSync(envFilePath)) {
    return {};
  }

  const content = fs.readFileSync(envFilePath, 'utf8');
  const values = {};
  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) {
      continue;
    }
    const separatorIndex = rawLine.indexOf('=');
    if (separatorIndex <= 0) {
      continue;
    }
    const key = rawLine.slice(0, separatorIndex).trim();
    let value = rawLine.slice(separatorIndex + 1).trim();
    if (
      (value.startsWith('"') && value.endsWith('"'))
      || (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }
  return values;
}

export function resolveMysqlConfig(env = process.env, options = {}) {
  const localEnv = options.localEnv ?? readEnvFile(path.join(options.repoRoot ?? repoRoot, '.env'));
  const parsedDsn = parseDsn(firstNonEmpty(env.PANTHEON_DSN, localEnv.PANTHEON_DSN));
  const explicitPassword = firstNonEmpty(
    env.PANTHEON_SMOKE_MYSQL_PASSWORD,
    env.MYSQL_ROOT_PASSWORD,
    localEnv.PANTHEON_SMOKE_MYSQL_PASSWORD,
    localEnv.MYSQL_ROOT_PASSWORD,
    parsedDsn?.password,
  );
  return {
    host: firstNonEmpty(
      env.PANTHEON_SMOKE_MYSQL_HOST,
      localEnv.PANTHEON_SMOKE_MYSQL_HOST,
      parsedDsn?.host,
      '127.0.0.1',
    ),
    port: Number(
      firstNonEmpty(
        env.PANTHEON_SMOKE_MYSQL_PORT,
        localEnv.PANTHEON_SMOKE_MYSQL_PORT,
        parsedDsn?.port ? String(parsedDsn.port) : '',
        '3306',
      ),
    ),
    username: firstNonEmpty(
      env.PANTHEON_SMOKE_MYSQL_USER,
      localEnv.PANTHEON_SMOKE_MYSQL_USER,
      parsedDsn?.username,
      'root',
    ),
    password: explicitPassword,
    database: firstNonEmpty(
      env.PANTHEON_SMOKE_MYSQL_DATABASE,
      localEnv.PANTHEON_SMOKE_MYSQL_DATABASE,
      parsedDsn?.database,
      'pantheon_base',
    ),
    mysqlBin: firstNonEmpty(
      env.PANTHEON_SMOKE_MYSQL_BIN,
      localEnv.PANTHEON_SMOKE_MYSQL_BIN,
      'mysql',
    ),
    hasExplicitPassword: explicitPassword !== '',
  };
}

export function buildMysqlPasswordCandidates(config) {
  if (config.hasExplicitPassword) {
    return [config.password];
  }

  return Array.from(new Set([...documentedDevMysqlPasswords, '']));
}

function buildSetupSql() {
  return `
CREATE TABLE IF NOT EXISTS \`${fixtureTableName}\` (
  \`id\` bigint unsigned NOT NULL AUTO_INCREMENT,
  \`host_code\` varchar(64) NOT NULL COMMENT '主机编码',
  \`hostname\` varchar(128) NOT NULL COMMENT '主机名',
  \`ip\` varchar(64) NOT NULL COMMENT 'IP',
  \`os\` varchar(128) NOT NULL COMMENT '操作系统',
  \`environment\` varchar(32) NOT NULL COMMENT '环境',
  \`status\` varchar(32) NOT NULL COMMENT '状态',
  \`arch\` varchar(32) DEFAULT NULL COMMENT '架构',
  \`provider\` varchar(64) DEFAULT NULL COMMENT '云厂商',
  \`owner_name\` varchar(64) DEFAULT NULL COMMENT '负责人',
  \`ssh_port\` int DEFAULT NULL COMMENT 'SSH 端口',
  \`remark\` text COMMENT '备注',
  \`created_at\` datetime(3) DEFAULT NULL,
  \`updated_at\` datetime(3) DEFAULT NULL,
  \`deleted_at\` datetime(3) DEFAULT NULL,
  PRIMARY KEY (\`id\`),
  UNIQUE KEY \`uidx_biz_cmdb_host_host_code\` (\`host_code\`),
  UNIQUE KEY \`uidx_biz_cmdb_host_hostname\` (\`hostname\`),
  KEY \`idx_biz_cmdb_host_deleted_at\` (\`deleted_at\`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CMDB 主机 QA 导入源表';
`;
}

function buildCleanupSql() {
  return `
SELECT 1;
`;
}

function isAuthFailure(result) {
  return /ERROR 1045|Access denied/i.test(result.stderr);
}

function executeMysql(sql, config, password) {
  return new Promise((resolve, reject) => {
    const child = spawn(
      config.mysqlBin,
      [
        `--host=${config.host}`,
        `--port=${String(config.port)}`,
        `--user=${config.username}`,
        '--default-character-set=utf8mb4',
        '--protocol=TCP',
        config.database,
        '-e',
        sql,
      ],
      {
        stdio: ['ignore', 'pipe', 'pipe'],
        shell: false,
        env: password === '' ? process.env : { ...process.env, MYSQL_PWD: password },
      },
    );
    let stdout = '';
    let stderr = '';
    child.stdout?.setEncoding('utf8');
    child.stderr?.setEncoding('utf8');
    child.stdout?.on('data', (chunk) => {
      stdout += chunk;
    });
    child.stderr?.on('data', (chunk) => {
      stderr += chunk;
    });
    child.once('error', reject);
    child.once('exit', (code) => {
      resolve({
        code: code ?? 0,
        stdout,
        stderr,
      });
    });
  });
}

async function runMysql(sql, config) {
  let lastResult = null;
  for (const password of buildMysqlPasswordCandidates(config)) {
    const result = await executeMysql(sql, config, password);
    if (result.code === 0) {
      if (result.stdout.trim()) {
        process.stdout.write(result.stdout);
      }
      return;
    }
    lastResult = result;
    if (!isAuthFailure(result)) {
      break;
    }
  }

  if (lastResult?.stderr.trim()) {
    process.stderr.write(lastResult.stderr);
  }
  throw new Error(`mysql exited with code ${lastResult?.code ?? 'unknown'}`);
}

async function main() {
  const { action } = parseArgs();
  const mysqlConfig = resolveMysqlConfig();
  if (action === 'down') {
    await runMysql(buildCleanupSql(), mysqlConfig);
    return;
  }
  await runMysql(buildSetupSql(), mysqlConfig);
}

if (process.argv[1] && path.resolve(process.argv[1]) === scriptPath) {
  main().catch((error) => {
    console.error(error);
    process.exit(1);
  });
}
