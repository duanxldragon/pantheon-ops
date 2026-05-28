import { spawn } from 'node:child_process';
import process from 'node:process';

const fixtureTableName = 'biz_cmdb_host';
const fixtureHostCodes = ['qa-host-001', 'qa-host-002'];

function parseArgs() {
  return {
    action: process.argv[2] || 'up',
  };
}

function parseDsn(dsn) {
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

function resolveMysqlConfig() {
  const parsedDsn = parseDsn(process.env.PANTHEON_DSN);
  return {
    host: process.env.PANTHEON_SMOKE_MYSQL_HOST || parsedDsn?.host || '127.0.0.1',
    port: Number(process.env.PANTHEON_SMOKE_MYSQL_PORT || parsedDsn?.port || 3306),
    username: process.env.PANTHEON_SMOKE_MYSQL_USER || parsedDsn?.username || 'root',
    password: process.env.PANTHEON_SMOKE_MYSQL_PASSWORD || parsedDsn?.password || '',
    database: process.env.PANTHEON_SMOKE_MYSQL_DATABASE || parsedDsn?.database || 'pantheon_base',
    mysqlBin: process.env.PANTHEON_SMOKE_MYSQL_BIN || 'mysql',
  };
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

DELETE FROM \`${fixtureTableName}\`
WHERE \`host_code\` IN (${fixtureHostCodes.map((value) => `'${value}'`).join(', ')});

INSERT INTO \`${fixtureTableName}\`
  (\`host_code\`, \`hostname\`, \`ip\`, \`os\`, \`environment\`, \`status\`, \`arch\`, \`provider\`, \`owner_name\`, \`ssh_port\`, \`remark\`, \`created_at\`, \`updated_at\`, \`deleted_at\`)
VALUES
  ('qa-host-001', 'qa-host-001.internal', '10.20.30.41', 'Ubuntu 24.04', 'test', 'active', 'x86_64', 'aliyun', 'QA Team', 22, 'database-import smoke fixture A', NOW(3), NOW(3), NULL),
  ('qa-host-002', 'qa-host-002.internal', '10.20.30.42', 'Rocky Linux 9', 'prod', 'inactive', 'arm64', 'tencent', 'Ops Team', 2222, 'database-import smoke fixture B', NOW(3), NOW(3), NULL);
`;
}

function buildCleanupSql() {
  return `
DELETE FROM \`${fixtureTableName}\`
WHERE \`host_code\` IN (${fixtureHostCodes.map((value) => `'${value}'`).join(', ')});
`;
}

function runMysql(sql, config) {
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
        stdio: 'inherit',
        shell: false,
        env: {
          ...process.env,
          MYSQL_PWD: config.password,
        },
      },
    );
    child.once('error', reject);
    child.once('exit', (code) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`mysql exited with code ${code ?? 'unknown'}`));
    });
  });
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

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
