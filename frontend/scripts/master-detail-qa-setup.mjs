import { getOperationToken, loginWithOptionalMfa } from './smoke-auth.mjs';
import {
  createSmokeConfig,
  finalizeSchema,
  generateModule,
  purgeModule,
} from './qa-dynamic-module-utils.mjs';

const { apiBaseUrl, adminUsername, adminPassword } = createSmokeConfig();

const masterModuleName = 'mdqaorder';
const detailModuleName = 'mdqaorderitem';
const masterModuleKey = `business.${masterModuleName}`;
const detailModuleKey = `business.${detailModuleName}`;
const masterTableName = 'biz_mdqa_order';
const detailTableName = 'biz_mdqa_order_item';

function buildMasterSchema() {
  return finalizeSchema({
    name: masterModuleName,
    templateVersion: 'v1',
    displayName: '主从订单',
    displayNameEn: 'Master Detail Order',
    description: 'QA master module for low-code child table flow',
    scope: 'business',
    parentMenu: '',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'detail'],
    dependencies: [],
    relations: [
      {
        name: 'orderItems',
        type: 'oneToMany',
        targetModule: detailModuleName,
        localField: 'id',
        targetField: 'orderId',
        targetLabelField: 'itemName',
      },
    ],
    metadata: {
      businessContext: 'qa',
      businessContextTitle: 'QA',
      businessContextTitleEn: 'QA',
      tableRole: 'main',
      boundedContext: 'qa',
      owner: 'codex',
      summary: '主从表详情编辑链路验证',
      sourceMode: 'manual',
      sourceTable: '',
      autoRecycle: true,
    },
    model: {
      tableName: masterTableName,
      modelName: 'Mdqaorder',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '订单名称',
          labelEn: 'Order Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入订单名称',
          placeholderEn: 'Enter order name',
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
            { value: 'active', label: '生效', labelEn: 'Active' },
          ],
        },
      ],
    },
    menus: [],
    permissions: [],
    i18n: { namespace: '', translations: { zh: {}, en: {} } },
  });
}

function buildDetailSchema() {
  return finalizeSchema({
    name: detailModuleName,
    templateVersion: 'v1',
    displayName: '订单明细',
    displayNameEn: 'Order Item',
    description: 'QA detail module for low-code child table flow',
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
      tableRole: 'detail',
      primaryTable: masterTableName,
      relationFromField: 'orderId',
      relationToField: 'id',
      boundedContext: 'qa',
      owner: 'codex',
      summary: '主从子表编辑链路验证',
      sourceMode: 'manual',
      sourceTable: '',
      autoRecycle: true,
    },
    model: {
      tableName: detailTableName,
      modelName: 'Mdqaorderitem',
      fields: [
        {
          name: 'itemName',
          type: 'string',
          label: '明细名称',
          labelEn: 'Item Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入明细名称',
          placeholderEn: 'Enter item name',
        },
        {
          name: 'quantity',
          type: 'int',
          label: '数量',
          labelEn: 'Quantity',
          required: true,
          searchable: false,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入数量',
          placeholderEn: 'Enter quantity',
        },
        {
          name: 'enabled',
          type: 'bool',
          label: '启用',
          labelEn: 'Enabled',
          required: false,
          searchable: false,
          sortable: false,
          visibleInList: true,
          visibleInForm: true,
        },
        {
          name: 'remark',
          type: 'text',
          label: '备注',
          labelEn: 'Remark',
          required: false,
          searchable: false,
          sortable: false,
          visibleInList: false,
          visibleInForm: true,
          placeholder: '请输入备注',
          placeholderEn: 'Enter remark',
        },
        {
          name: 'orderId',
          type: 'int',
          label: '订单ID',
          labelEn: 'Order ID',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '自动回填',
          placeholderEn: 'Auto filled',
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
    await purgeModule(apiBaseUrl, masterModuleKey, accessToken, csrfToken, operationToken);
    await purgeModule(apiBaseUrl, detailModuleKey, accessToken, csrfToken, operationToken);
    return;
  }

  await purgeModule(apiBaseUrl, masterModuleKey, accessToken, csrfToken, operationToken);
  await purgeModule(apiBaseUrl, detailModuleKey, accessToken, csrfToken, operationToken);
  await generateModule(apiBaseUrl, buildDetailSchema(), accessToken, csrfToken, operationToken);
  await generateModule(apiBaseUrl, buildMasterSchema(), accessToken, csrfToken, operationToken);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
