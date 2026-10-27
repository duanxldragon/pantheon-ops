export type DeployFixedTemplateCode =
  | 'nginx_systemd'
  | 'mysql_systemd'
  | 'redis_systemd'
  | 'minio_systemd'
  | 'harbor_offline'
  | 'jdk_systemd'
  | 'prometheus_systemd';

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
      { key: 'version', labelKey: 'business.deploy.task.templateParams.version', defaultValue: '1.24.0' },
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
      { key: 'rootPassword', labelKey: 'business.deploy.task.templateParams.rootPassword', defaultValue: '', secret: true },
    ],
  },
  {
    code: 'redis_systemd',
    summaryKey: 'business.deploy.package.templateSummary.redis_systemd',
    parameters: [
      { key: 'version', labelKey: 'business.deploy.task.templateParams.version', defaultValue: '7.2.4' },
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
      { key: 'rootPassword', labelKey: 'business.deploy.task.templateParams.rootPassword', defaultValue: '', secret: true },
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
      { key: 'adminPassword', labelKey: 'business.deploy.task.templateParams.adminPassword', defaultValue: '', secret: true },
    ],
  },
  {
    code: 'jdk_systemd',
    summaryKey: 'business.deploy.package.templateSummary.jdk_systemd',
    parameters: [
      { key: 'installRoot', labelKey: 'business.deploy.task.templateParams.installRoot', defaultValue: '/data/jdk' },
      { key: 'serviceName', labelKey: 'business.deploy.task.templateParams.serviceName', defaultValue: 'jdk' },
      { key: 'mainClass', labelKey: 'business.deploy.task.templateParams.mainClass', defaultValue: '' },
      { key: 'appArgs', labelKey: 'business.deploy.task.templateParams.appArgs', defaultValue: '' },
      { key: 'javaOpts', labelKey: 'business.deploy.task.templateParams.javaOpts', defaultValue: '-Xms512m -Xmx512m' },
    ],
  },
  {
    code: 'prometheus_systemd',
    summaryKey: 'business.deploy.package.templateSummary.prometheus_systemd',
    parameters: [
      { key: 'installRoot', labelKey: 'business.deploy.task.templateParams.installRoot', defaultValue: '/data/prometheus' },
      { key: 'dataRoot', labelKey: 'business.deploy.task.templateParams.dataRoot', defaultValue: '/data/prometheus/data' },
      { key: 'serviceName', labelKey: 'business.deploy.task.templateParams.serviceName', defaultValue: 'prometheus' },
      { key: 'webPort', labelKey: 'business.deploy.task.templateParams.webPort', defaultValue: '9090' },
      { key: 'storageRetentionPolicy', labelKey: 'business.deploy.task.templateParams.storageRetentionPolicy', defaultValue: '30d' },
      { key: 'scrapeConfigFile', labelKey: 'business.deploy.task.templateParams.scrapeConfigFile', defaultValue: '' },
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

// Declarative component step types. These mirror the backend component registry
// keys (deploy_component.go) and map one-to-one onto a fixed template code.
export const deployComponentStepTypes = [
  'nginx',
  'redis',
  'jdk',
  'mysql',
  'minio',
  'harbor',
  'prometheus',
] as const;

export type DeployComponentStepType = (typeof deployComponentStepTypes)[number];

export const deployComponentTemplateCodeMap: Record<DeployComponentStepType, DeployFixedTemplateCode> = {
  nginx: 'nginx_systemd',
  redis: 'redis_systemd',
  jdk: 'jdk_systemd',
  mysql: 'mysql_systemd',
  minio: 'minio_systemd',
  harbor: 'harbor_offline',
  prometheus: 'prometheus_systemd',
};

export function isDeployComponentStepType(
  value: string | null | undefined,
): value is DeployComponentStepType {
  return Boolean(value && (deployComponentStepTypes as readonly string[]).includes(value));
}

export function getDeployComponentTemplateEntry(stepType?: string | null) {
  if (!isDeployComponentStepType(stepType)) {
    return null;
  }
  return getDeployFixedTemplateCatalogEntry(deployComponentTemplateCodeMap[stepType]);
}
