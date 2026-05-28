export type DeployFixedTemplateCode =
  | 'nginx_systemd'
  | 'mysql_systemd'
  | 'redis_systemd'
  | 'minio_systemd'
  | 'harbor_offline';

export type DeployTemplateParameterDefinition = {
  key: string;
  labelKey: string;
  defaultValue: string;
  secret?: boolean;
};

export type DeployTemplateCatalogEntry = {
  code: DeployFixedTemplateCode;
  summaryKey: string;
  parameters: DeployTemplateParameterDefinition[];
};

export const deployFixedTemplateCatalog: DeployTemplateCatalogEntry[] = [
  {
    code: 'nginx_systemd',
    summaryKey: 'business.deploy.package.templateSummary.nginx_systemd',
    parameters: [
      { key: 'installRoot', labelKey: 'business.deploy.task.templateParams.installRoot', defaultValue: '/data/nginx' },
      { key: 'serviceName', labelKey: 'business.deploy.task.templateParams.serviceName', defaultValue: 'nginx' },
    ],
  },
  {
    code: 'mysql_systemd',
    summaryKey: 'business.deploy.package.templateSummary.mysql_systemd',
    parameters: [
      { key: 'installRoot', labelKey: 'business.deploy.task.templateParams.installRoot', defaultValue: '/data/mysql' },
      { key: 'dataRoot', labelKey: 'business.deploy.task.templateParams.dataRoot', defaultValue: '/data/mysql/data' },
      { key: 'serviceName', labelKey: 'business.deploy.task.templateParams.serviceName', defaultValue: 'mysqld' },
      { key: 'port', labelKey: 'business.deploy.task.templateParams.port', defaultValue: '3306' },
      { key: 'rootPassword', labelKey: 'business.deploy.task.templateParams.rootPassword', defaultValue: 'ChangeMe_123', secret: true },
    ],
  },
  {
    code: 'redis_systemd',
    summaryKey: 'business.deploy.package.templateSummary.redis_systemd',
    parameters: [
      { key: 'installRoot', labelKey: 'business.deploy.task.templateParams.installRoot', defaultValue: '/data/redis' },
      { key: 'dataRoot', labelKey: 'business.deploy.task.templateParams.dataRoot', defaultValue: '/data/redis/data' },
      { key: 'serviceName', labelKey: 'business.deploy.task.templateParams.serviceName', defaultValue: 'redis' },
      { key: 'port', labelKey: 'business.deploy.task.templateParams.port', defaultValue: '6379' },
      { key: 'requirePassword', labelKey: 'business.deploy.task.templateParams.requirePassword', defaultValue: '' , secret: true},
    ],
  },
  {
    code: 'minio_systemd',
    summaryKey: 'business.deploy.package.templateSummary.minio_systemd',
    parameters: [
      { key: 'installRoot', labelKey: 'business.deploy.task.templateParams.installRoot', defaultValue: '/data/minio' },
      { key: 'dataRoot', labelKey: 'business.deploy.task.templateParams.dataRoot', defaultValue: '/data/minio/data' },
      { key: 'serviceName', labelKey: 'business.deploy.task.templateParams.serviceName', defaultValue: 'minio' },
      { key: 'apiPort', labelKey: 'business.deploy.task.templateParams.apiPort', defaultValue: '9000' },
      { key: 'consolePort', labelKey: 'business.deploy.task.templateParams.consolePort', defaultValue: '9001' },
      { key: 'rootUser', labelKey: 'business.deploy.task.templateParams.rootUser', defaultValue: 'minioadmin' },
      { key: 'rootPassword', labelKey: 'business.deploy.task.templateParams.rootPassword', defaultValue: 'ChangeMe_123', secret: true },
    ],
  },
  {
    code: 'harbor_offline',
    summaryKey: 'business.deploy.package.templateSummary.harbor_offline',
    parameters: [
      { key: 'installRoot', labelKey: 'business.deploy.task.templateParams.installRoot', defaultValue: '/data/harbor' },
      { key: 'dataRoot', labelKey: 'business.deploy.task.templateParams.dataRoot', defaultValue: '/data/harbor/data' },
      { key: 'hostname', labelKey: 'business.deploy.task.templateParams.hostname', defaultValue: 'harbor.local' },
      { key: 'httpPort', labelKey: 'business.deploy.task.templateParams.httpPort', defaultValue: '8088' },
      { key: 'adminPassword', labelKey: 'business.deploy.task.templateParams.adminPassword', defaultValue: 'ChangeMe_123', secret: true },
    ],
  },
];

export function getDeployFixedTemplateCatalogEntry(templateCode?: string | null) {
  return deployFixedTemplateCatalog.find((item) => item.code === templateCode) || null;
}

export function buildDeployTemplateDefaultParameters(templateCode?: string | null) {
  const entry = getDeployFixedTemplateCatalogEntry(templateCode);
  if (!entry) {
    return {};
  }
  return Object.fromEntries(entry.parameters.map((item) => [item.key, item.defaultValue]));
}
