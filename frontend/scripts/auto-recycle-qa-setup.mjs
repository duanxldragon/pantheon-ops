import {
  executeMysql,
  getOperationToken,
  loginWithOptionalMfa,
} from './smoke-auth.mjs';
import {
  createSmokeConfig,
  finalizeSchema,
  generateModule,
  purgeModule,
} from './qa-dynamic-module-utils.mjs';

const { apiBaseUrl, adminUsername, adminPassword } = createSmokeConfig();

const moduleName = 'autorecycleqa';
const moduleKey = `business.${moduleName}`;
const tableName = 'biz_auto_recycle_qa';

function buildSchema() {
  return finalizeSchema({
    name: moduleName,
    templateVersion: 'v1',
    displayName: '自动回收 QA 模块',
    displayNameEn: 'Auto Recycle QA Module',
    description: 'QA module for lifecycle purge verification',
    scope: 'business',
    parentMenu: '',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'detail'],
    metadata: {
      businessContext: 'qa',
      businessContextTitle: 'QA',
      businessContextTitleEn: 'QA',
      tableRole: 'main',
      autoRecycle: true,
      boundedContext: 'qa',
      owner: 'codex',
      summary: '验证临时 QA 表自动回收',
      sourceMode: 'manual',
      sourceTable: '',
    },
    model: {
      tableName,
      modelName: 'Autorecycleqa',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '名称',
          labelEn: 'Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入名称',
          placeholderEn: 'Enter name',
        },
        {
          name: 'status',
          type: 'enum',
          label: '状态',
          labelEn: 'Status',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请选择状态',
          placeholderEn: 'Select status',
          enumOptions: [
            { value: 'draft', label: '草稿', labelEn: 'Draft' },
            { value: 'active', label: '启用', labelEn: 'Active' },
          ],
        },
      ],
    },
    dependencies: [],
    relations: [],
    menus: [],
    permissions: [],
    i18n: { namespace: '', translations: { zh: {}, en: {} } },
  });
}

async function ensureManagedTable() {
  await executeMysql(`
CREATE TABLE IF NOT EXISTS ${tableName} (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_${tableName}_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`);
}

async function main() {
  const action = process.argv[2] || 'up';
  const loginData = await loginWithOptionalMfa(apiBaseUrl, {
    username: adminUsername,
    password: adminPassword,
  });
  const accessToken = loginData.accessToken;
  const csrfToken = loginData.csrfToken;
  const operationToken = await getOperationToken(apiBaseUrl, loginData, adminPassword);

  if (action === 'down') {
    await purgeModule(apiBaseUrl, moduleKey, accessToken, csrfToken, operationToken);
    return;
  }

  await purgeModule(apiBaseUrl, moduleKey, accessToken, csrfToken, operationToken);
  await generateModule(apiBaseUrl, buildSchema(), accessToken, csrfToken, operationToken);
  await ensureManagedTable();
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
