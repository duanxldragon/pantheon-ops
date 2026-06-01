import { getOperationToken, loginWithOptionalMfa } from './smoke-auth.mjs';
import {
  createSmokeConfig,
  finalizeSchema,
  generateModule,
  purgeModule,
} from './qa-dynamic-module-utils.mjs';

const { apiBaseUrl, adminUsername, adminPassword } = createSmokeConfig();

const ownerModuleName = 'm2mqaasset';
const targetModuleName = 'm2mqatag';
const ownerModuleKey = `business.${ownerModuleName}`;
const targetModuleKey = `business.${targetModuleName}`;
const ownerTableName = 'biz_m2mqa_asset';
const targetTableName = 'biz_m2mqa_tag';

function buildOwnerSchema() {
  return finalizeSchema({
    name: ownerModuleName,
    templateVersion: 'v1',
    displayName: '多对多资产',
    displayNameEn: 'Many To Many Asset',
    description: 'QA owner module for many-to-many runtime flow',
    scope: 'business',
    parentMenu: '',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'detail'],
    dependencies: [{ module: targetModuleName, required: true, reason: '绑定标签' }],
    relations: [
      {
        name: 'assetTags',
        type: 'manyToMany',
        targetModule: targetModuleName,
        localField: 'id',
        targetField: 'id',
        targetLabelField: 'name',
        lookupApi: `/business/${targetModuleName}/options`,
        lookupValueField: 'id',
        junctionTable: 'biz_m2mqa_asset_tag_rel',
      },
    ],
    metadata: {
      businessContext: 'qa',
      businessContextTitle: 'QA',
      businessContextTitleEn: 'QA',
      tableRole: 'main',
      boundedContext: 'qa',
      owner: 'codex',
      summary: '多对多详情绑定链路验证',
      sourceMode: 'manual',
      sourceTable: '',
      autoRecycle: true,
    },
    model: {
      tableName: ownerTableName,
      modelName: 'M2mqaasset',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '资产名称',
          labelEn: 'Asset Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入资产名称',
          placeholderEn: 'Enter asset name',
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
    menus: [],
    permissions: [],
    i18n: { namespace: '', translations: { zh: {}, en: {} } },
  });
}

function buildTargetSchema() {
  return finalizeSchema({
    name: targetModuleName,
    templateVersion: 'v1',
    displayName: '多对多标签',
    displayNameEn: 'Many To Many Tag',
    description: 'QA target module for many-to-many runtime flow',
    scope: 'business',
    parentMenu: '',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'detail'],
    dependencies: [],
    relations: [],
    metadata: {
      businessContext: 'qa',
      businessContextTitle: 'QA',
      businessContextTitleEn: 'QA',
      tableRole: 'dictionary',
      boundedContext: 'qa',
      owner: 'codex',
      summary: '多对多标签字典',
      sourceMode: 'manual',
      sourceTable: '',
      autoRecycle: true,
    },
    model: {
      tableName: targetTableName,
      modelName: 'M2mqatag',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '标签名称',
          labelEn: 'Tag Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入标签名称',
          placeholderEn: 'Enter tag name',
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
            { value: 'active', label: '启用', labelEn: 'Active' },
            { value: 'disabled', label: '停用', labelEn: 'Disabled' },
          ],
        },
      ],
    },
    menus: [],
    permissions: [],
    i18n: { namespace: '', translations: { zh: {}, en: {} } },
  });
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
    await purgeModule(apiBaseUrl, ownerModuleKey, accessToken, csrfToken, operationToken);
    await purgeModule(apiBaseUrl, targetModuleKey, accessToken, csrfToken, operationToken);
    return;
  }

  await purgeModule(apiBaseUrl, ownerModuleKey, accessToken, csrfToken, operationToken);
  await purgeModule(apiBaseUrl, targetModuleKey, accessToken, csrfToken, operationToken);
  await generateModule(apiBaseUrl, buildTargetSchema(), accessToken, csrfToken, operationToken);
  await generateModule(apiBaseUrl, buildOwnerSchema(), accessToken, csrfToken, operationToken);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
