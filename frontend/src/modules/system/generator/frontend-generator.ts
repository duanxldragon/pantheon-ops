/**
 * 模块生成器 - 前端代码生成器
 *
 * 基于项目真实代码结构生成前端 TypeScript/React 代码:
 * - 完整的5类状态处理(loading/empty/error/forbidden/submitting)
 * - 权限检查集成
 * - 100% 国际化支持
 * - 强类型API接口
 *
 * 参考:
 * - system/user/index.ts
 * - system/user/api.ts
 * - system/user/UserList.tsx
 */

import type { ModuleSchema } from './schema';
import {
  buildComponentKey,
  buildDashboardQuickActionDescriptionKey,
  buildEnumOptionKey,
  buildFieldLabelKey,
  buildFieldPlaceholderKey,
  buildModuleNamespace,
  buildPermissionPrefix,
  buildRouteName,
  buildRoutePath,
  buildTitleKey,
  getPageActions,
  inferModelName,
  shouldGenerateNavigation,
  splitModuleSegments,
} from './schema';
import { TYPE_MAPPING } from './type-mapping';

export class FrontendGenerator {
  private schema: ModuleSchema;
  private modelName: string;

  constructor(schema: ModuleSchema) {
    this.schema = schema;
    this.modelName = inferModelName(schema);
  }

  /**
   * 生成 index.ts (模块注册)
   *
   * 参考: system/user/index.ts
   */
  generateModuleIndex(): string {
    const { scope, name } = this.schema;
    const modelName = this.modelName;
    const routePath = buildRoutePath(scope, name).replace(/^\//, '');
    const titleKey = buildTitleKey(scope, name);
    const permissionPrefix = buildPermissionPrefix(scope, name);
    const moduleNamespace = buildModuleNamespace(scope, name);
    const routeName = buildRouteName(scope, name);
    const listComponentKey = buildComponentKey(scope, name, modelName);
    const detailComponentKey = this.buildDetailComponentKey(scope, name);
    const toSrcRoot = this.relativeToSrcRoot();
    const pageActions = getPageActions(this.schema);
    const generateNavigation = shouldGenerateNavigation(this.schema);
    const generateDetailRoute = generateNavigation && (pageActions.includes('view') || pageActions.includes('detail'));
    const businessContext = splitModuleSegments(name)[0] || 'default';
    const quickActionDescriptionKey = buildDashboardQuickActionDescriptionKey(scope, name);

    const permissionItems = generateNavigation
      ? [
          `'${permissionPrefix}:list'`,
          ...pageActions
            .filter((action) => action !== 'detail')
            .map((action) => `'${permissionPrefix}:${action}'`),
        ]
      : [];
    const dashboardWidgets =
      scope === 'business' && generateNavigation && this.schema.includeDashboardWidget !== false
        ? `
  dashboardWidgets: [
    {
      key: '${permissionPrefix}',
      slot: 'quick-action',
      sourceDomain: 'business/${businessContext}',
      titleKey: '${titleKey}',
      descriptionKey: '${quickActionDescriptionKey}',
      path: '${buildRoutePath(scope, name)}',
      permission: '${permissionPrefix}:list',
      icon: 'apps',
      cleanupPolicy: 'remove_with_source_module',
      registrationOwner: '${moduleNamespace}',
    },
  ],`
        : '';

    return `import { defineModule } from '${toSrcRoot}/core/router/types';

export const ${modelName}Module = defineModule({
  name: '${name}',
  scope: '${scope}',
  routes: ${
    generateNavigation
      ? `[
    {
      path: '${routePath}',
      routeName: '${routeName}',
      titleKey: '${titleKey}',
      icon: 'apps',
      pagePermission: '${permissionPrefix}:list',
      componentKey: '${listComponentKey}',
    },
    ${
      generateDetailRoute
        ? `{
      path: '${routePath}/:id',
      routeName: '${routeName}-detail',
      titleKey: '${titleKey}',
      pagePermission: '${permissionPrefix}:view',
      activeMenu: '${buildRoutePath(scope, name)}',
      componentKey: '${detailComponentKey}',
    },`
        : ''
    }
  ]`
      : '[]'
  },
  menus: ${
    generateNavigation
      ? `[
    { path: '${buildRoutePath(scope, name)}', titleKey: '${titleKey}', icon: 'apps', routeName: '${routeName}', module: '${moduleNamespace}' },
  ]`
      : '[]'
  },${dashboardWidgets}
  permissions: ${
    permissionItems.length > 0
      ? `[
    ${permissionItems.join(',\n    ')}
  ]`
      : '[]'
  },
  i18nNamespaces: ['${moduleNamespace}'],
});
`;
  }

  /**
   * 生成 api.ts
   *
   * 参考: system/user/api.ts
   */
  generateAPI(): string {
    const modelName = this.modelName;
    const { scope, name } = this.schema;
    const routePath = buildRoutePath(scope, name);
    const toSrcRoot = this.relativeToSrcRoot();
    const pageActions = getPageActions(this.schema);
    const relationLookupApi = this.generateRelationLookupApi();
    const relationBindingApi = this.generateManyToManyRelationApi();

    const extraImports = [
      pageActions.includes('export') ? `import { downloadFile } from '${toSrcRoot}/api/file';` : '',
      pageActions.includes('import')
        ? `import { uploadImportFile } from '${toSrcRoot}/api/importExport';`
        : '',
    ]
      .filter(Boolean)
      .join('\n');

    const extraApi = [
      pageActions.includes('export')
        ? `
export function export${modelName}s(data?: ${modelName}ListQuery) {
  return downloadFile({
    url: '${routePath}/export',
    method: 'post',
    data,
    filename: '${buildRouteName(scope, name)}-export.csv',
  });
}`
        : '',
      pageActions.includes('import')
        ? `
export function import${modelName}s(file: File) {
  return uploadImportFile('${routePath}/import', file);
}`
        : '',
    ]
      .filter(Boolean)
      .join('\n\n');

    const mutatingApi = [
      pageActions.includes('create')
        ? `
export function create${modelName}(data: ${modelName}CreatePayload) {
  return apiRequest<${modelName}ListRow>({
    url: '${routePath}',
    method: 'post',
    data,
  });
}`
        : '',
      pageActions.includes('update')
        ? `
export function update${modelName}(id: number, data: ${modelName}UpdatePayload) {
  return apiRequest<${modelName}ListRow>({
    url: \`${routePath}/\${id}\`,
    method: 'put',
    data,
  });
}`
        : '',
      pageActions.includes('delete')
        ? `
export function delete${modelName}(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: \`${routePath}/\${id}\`,
    method: 'delete',
  });
}`
        : '',
    ]
      .filter(Boolean)
      .join('\n\n');

    return `import { apiRequest } from '${toSrcRoot}/api/request';
${extraImports ? `${extraImports}\n` : ''}

export interface ${modelName}ListRow {
  id: number;
${this.generateTSFields('list')}
  createdAt: string;
}

export interface ${modelName}Detail {
  id: number;
${this.generateTSFields('detail')}
  createdAt: string;
  updatedAt: string;
}

export interface ${modelName}ListQuery {
${this.generateTSQueryFields()}
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface ${modelName}ListPageResp {
  items: ${modelName}ListRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ${modelName}CreatePayload {
${this.generateTSFields('create')}
}

export interface ${modelName}UpdatePayload {
${this.generateTSFields('update')}
}

${relationLookupApi}
${relationBindingApi ? `\n${relationBindingApi}\n` : ''}

export function get${modelName}List(params?: ${modelName}ListQuery) {
  return apiRequest<${modelName}ListPageResp>({
    url: '${routePath}/list',
    method: 'get',
    params,
  });
}

export function get${modelName}Detail(id: number) {
  return apiRequest<${modelName}Detail>({
    url: \`${routePath}/\${id}\`,
    method: 'get',
  });
}

${mutatingApi ? `\n${mutatingApi}\n` : ''}
${extraApi ? `\n${extraApi}\n` : ''}
`;
  }

  /**
   * 生成 TypeScript 接口字段
   */
  private generateTSFields(mode: 'list' | 'detail' | 'create' | 'update'): string {
    return this.schema.model.fields
      .filter((f) => {
        if (mode === 'list') return f.visibleInList !== false;
        if (mode === 'create' || mode === 'update') return f.visibleInForm !== false;
        return true;
      })
      .map((field) => {
        const tsType = TYPE_MAPPING[field.type].ts;
        const isOptional = mode === 'update' || !field.required;
        return `  ${field.name}${isOptional ? '?' : ''}: ${tsType};`;
      })
      .join('\n');
  }

  /**
   * 生成 TypeScript 查询字段
   */
  private generateTSQueryFields(): string {
    const searchableFields = this.schema.model.fields.filter((f) => f.searchable);

    if (searchableFields.length === 0) {
      return '  // 无搜索字段';
    }

    return searchableFields
      .map((field) => {
        const tsType = TYPE_MAPPING[field.type].ts;
        return `  ${field.name}?: ${tsType};`;
      })
      .join('\n');
  }

  /**
   * 生成 List.tsx (列表页)
   *
   * 参考: system/user/UserList.tsx (726行完整实现)
   */
  generateListPage(): string {
    const { scope, name } = this.schema;
    const modelName = this.modelName;
    const toSrcRoot = this.relativeToSrcRoot();
    const titleKey = buildTitleKey(scope, name);
    const governanceEnabled = this.isListGovernanceEnabled();
    const searchEnabled = this.isListSearchEnabled();
    const headerActionsEnabled = this.isListHeaderActionsEnabled();
    const batchActionsEnabled = this.isListBatchActionsEnabled();
    const rowActionsEnabled = this.isListRowActionsEnabled();
    const governanceConstants = governanceEnabled
      ? `${this.generateListGovernanceConstants()}\n`
      : '';
    const listImports = ['Card', 'Typography', 'Space'];
    if (searchEnabled) {
      listImports.unshift('Button', 'Form', 'Grid', 'Input', 'Select');
    }
    if (headerActionsEnabled || rowActionsEnabled) {
      listImports.unshift('Popconfirm');
    }
    if (governanceEnabled) {
      listImports.unshift('Alert', 'Tag');
    }

    return `import React, { useCallback, useEffect, useState } from 'react';
import { ${listImports.join(', ')} } from '@arco-design/web-react';
import { IconDelete, IconDownload, IconEye, IconPlus, IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { message } from '${toSrcRoot}/components/feedback/message';
import {
  ${this.generateListActionImports()}
  get${modelName}List,
  type ${modelName}ListQuery,
  type ${modelName}ListRow,
} from './api';
import {
  AppTable,
  FilterPanel,
  ImportCsvButton,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SystemRowActions,
  TableBatchActionBar,
  TABLE_ACTION_COLUMN_WIDTH,
  buildStandardPagination,
} from '${toSrcRoot}/components';
import '${this.relativeToSystemListPageCss()}';

${governanceConstants}

const FormItem = Form.Item;
const { Row, Col } = Grid;

const emptyQuery: ${modelName}ListQuery = {
${this.generateEmptyQueryFields()}
  page: 1,
  pageSize: 20,
};

const ${modelName}List: React.FC = () => {
  const navigate = useNavigate();
  const [data, setData] = useState<${modelName}ListRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [query, setQuery] = useState<${modelName}ListQuery>(emptyQuery);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [submitting, setSubmitting] = useState(false);
  const [queryForm] = Form.useForm<${modelName}ListQuery>();
  const { t } = useTranslation();

  const loadData = useCallback(async (nextQuery: ${modelName}ListQuery = query) => {
    setLoading(true);
    setError(null);
    try {
      const result = await get${modelName}List(nextQuery);
      setData(result.items);
      setTotal(result.total);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const search = () => {
    const values = queryForm.getFieldsValue();
    setSelectedRowKeys([]);
    const nextQuery = {
      ...query,
      ...values,
      page: 1,
    };
    setQuery(nextQuery);
    void loadData(nextQuery);
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
    void loadData(emptyQuery);
  };

  const handleExport = async () => {
    ${this.hasPageAction('export') ? `await export${modelName}s(query);` : 'message.info(t(\'common.comingSoon\'));'}
  };

  const handleImport = async (file: File) => {
    ${this.hasPageAction('import') ? `await import${modelName}s(file);
    message.success(t('common.importSuccess'));` : 'void file;'}
    await loadData(query);
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      await Promise.all(selectedRowKeys.map((rowKey) => delete${modelName}(Number(rowKey))));
      message.success(t('common.deleteSuccess'));
      setSelectedRowKeys([]);
      await loadData(query);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading && data.length === 0) {
    return <PageLoading />;
  }

  if (error) {
    return <PageError onRetry={() => { void loadData(); }} />;
  }

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
${this.generateWorkbenchGovernanceBlock()}
${this.generateWorkbenchSearchBlock()}
        <Card className="page-panel system-list__table-card">
          ${this.generateWorkbenchSummaryHead(titleKey)}
          ${this.generateWorkbenchHeaderActions(headerActionsEnabled, batchActionsEnabled)}
          {loading && data.length === 0 ? <PageLoading /> : null}
          {error && data.length === 0 ? (
            <PageError
              onRetry={() => {
                void loadData(query);
              }}
            />
          ) : null}
          {!loading && !error && data.length === 0 ? (
            <PageEmpty description={t('common.noData')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              className="system-list__table"
              data={data}
              rowKey="id"
              scroll={{ x: 'max-content' }}
              ${batchActionsEnabled ? `rowSelection={{
                type: 'checkbox',
                selectedRowKeys,
                checkCrossPage: true,
                preserveSelectedRowKeys: true,
                fixed: true,
                onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
              }}` : ''}
              pagination={buildStandardPagination(t, {
                current: query.page,
                pageSize: query.pageSize,
                total,
                onChange: (page, pageSize) => {
                  const nextQuery = { ...query, page, pageSize };
                  setQuery(nextQuery);
                  void loadData(nextQuery);
                },
              })}
              columns={[
${this.generateWorkbenchTableColumns(rowActionsEnabled)}
              ]}
            />
          ) : null}
        </Card>
      </Space>
    </PageContainer>
  );
};

export default ${modelName}List;
      `;
  }

  generateFormComponent(): string {
    const modelName = this.modelName;
    const toSrcRoot = this.relativeToSrcRoot();
    const relationLookupContracts = this.generateRelationLookupContracts();
    const relationLookupImports = this.generateRelationLookupImports();
    const relationLookupLoaderEffect = this.generateRelationLookupLoaderEffect();
    const relationStateBlock = this.generateRelationFormStateBlock();
    const formImports = this.buildFormComponentImports();
    const formHookImports = this.schema.model.fields.some((field) => field.type === 'relation')
      ? 'useEffect, useState'
      : 'useEffect';

    return `import React, { ${formHookImports} } from 'react';
import { ${formImports.join(', ')} } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import {
  ${relationLookupImports}type ${modelName}CreatePayload,
  type ${modelName}UpdatePayload,
} from './api';
import { SubmitBar } from '${toSrcRoot}/components';

${relationLookupContracts}

const FormItem = Form.Item;

export interface ${modelName}FormValues extends Partial<${modelName}CreatePayload>, Partial<${modelName}UpdatePayload> {}

export interface ${modelName}FormProps {
  mode: 'create' | 'update';
  initialValues?: ${modelName}FormValues;
  submitting?: boolean;
  onSubmit: (values: ${modelName}FormValues) => void | Promise<void>;
  onCancel?: () => void;
}

const ${modelName}Form: React.FC<${modelName}FormProps> = ({
  mode,
  initialValues,
  submitting = false,
  onSubmit,
  onCancel,
}) => {
  const [form] = Form.useForm<${modelName}FormValues>();
${relationStateBlock}
  const { t } = useTranslation();

  useEffect(() => {
    form.setFieldsValue(initialValues || {});
  }, [form, initialValues]);

${relationLookupLoaderEffect}

  return (
    <Form form={form} layout="vertical" onSubmit={onSubmit}>
${this.generateFormItems()}
      <SubmitBar
        onCancel={onCancel}
        onSubmit={() => {
          form.submit();
        }}
        loading={submitting}
        submitText={mode === 'create' ? t('common.create') : t('common.save')}
      />
    </Form>
  );
};

export default ${modelName}Form;
`;
  }

  generateDetailPage(): string {
    const modelName = this.modelName;
    const titleKey = buildTitleKey(this.schema.scope, this.schema.name);
    const toSrcRoot = this.relativeToSrcRoot();
    const relationRuntimeHelpers = this.generateRelationRuntimeHelpers();
    const tableName = this.schema.model.tableName;

    return `import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { apiRequest } from '${toSrcRoot}/api/request';
import { message } from '${toSrcRoot}/components/feedback/message';
import { isArcoFormValidationError } from '${toSrcRoot}/core/arco/formValidation';
import { getGeneratedModuleSchema } from '${toSrcRoot}/modules/system/dynamicmodule/api';
import { get${modelName}Detail, type ${modelName}Detail } from './api';
import { AppModal, PageContainer, PageEmpty, PageError, PageLoading, SubmitBar } from '${toSrcRoot}/components';

${this.generateGovernanceConstants()}
${relationRuntimeHelpers}

const FormItem = Form.Item;
const TextArea = Input.TextArea;

const ${modelName}DetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const scope = '${this.schema.scope}';
  const moduleName = '${this.schema.name}';
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [detail, setDetail] = useState<${modelName}Detail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [relatedData, setRelatedData] = useState<Record<string, RelationRuntimeState>>({});
  const [childSchemas, setChildSchemas] = useState<Record<string, ChildModuleSchema | null>>({});
  const [childSchemaErrors, setChildSchemaErrors] = useState<Record<string, boolean>>({});
  const [editingChildRelation, setEditingChildRelation] = useState<RelationEditorState | null>(null);
  const [bindingRelation, setBindingRelation] = useState<ManyToManyBindingState | null>(null);
  const [childSubmitting, setChildSubmitting] = useState(false);
  const [manyToManySubmitting, setManyToManySubmitting] = useState(false);
  const [childRelationOptions, setChildRelationOptions] = useState<Record<string, RelationEditorOption[]>>({});
  const [childRelationLoading, setChildRelationLoading] = useState<Record<string, boolean>>({});
  const [childForm] = Form.useForm();
  const [bindForm] = Form.useForm();

  const loadDetail = useCallback(async () => {
    if (!id) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await get${modelName}Detail(Number(id));
      setDetail(result);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, [id]);

  const loadChildRelationOptions = useCallback(async (schema: ChildModuleSchema) => {
    const relationContracts = buildChildRelationContractMap(schema);
    const loadingKeys: string[] = [];

    setChildRelationLoading((prev) => {
      const next = { ...prev };
      Object.values(relationContracts).forEach((contract) => {
        if (!contract.lookupApi) {
          return;
        }
        const optionKey = buildChildRelationOptionKey(schema.name, contract.field);
        next[optionKey] = true;
        loadingKeys.push(optionKey);
      });
      return next;
    });

    try {
      const nextOptions: Record<string, RelationEditorOption[]> = {};
      await Promise.all(
        Object.values(relationContracts).map(async (contract) => {
          if (!contract.lookupApi) {
            return;
          }
          const rows = await loadLookupRelationRows(contract.lookupApi);
          nextOptions[buildChildRelationOptionKey(schema.name, contract.field)] =
            normalizeChildRelationOptions(rows, contract);
        }),
      );
      if (Object.keys(nextOptions).length > 0) {
        setChildRelationOptions((prev) => ({ ...prev, ...nextOptions }));
      }
    } finally {
      if (loadingKeys.length > 0) {
        setChildRelationLoading((prev) => {
          const next = { ...prev };
          loadingKeys.forEach((key) => {
            next[key] = false;
          });
          return next;
        });
      }
    }
  }, []);

  const loadRelationData = useCallback(
    async (nextDetail: ${modelName}Detail) => {
      const nextState: Record<string, RelationRuntimeState> = {};
      await Promise.all(
        governanceRelations.map(async (relation) => {
          nextState[relation.name] = { loading: true, items: [] };
          try {
            if (relation.type === 'lookup' && relation.lookupApi) {
              const optionRows = await loadLookupRelationRows(relation.lookupApi);
              const localValue = nextDetail[relation.localField as keyof ${modelName}Detail];
              const matched = optionRows.filter((row) => {
                const rawValue =
                  row[relation.lookupValueField || relation.targetField] ?? row.value ?? row.id;
                return String(rawValue ?? '') === String(localValue ?? '');
              });
              nextState[relation.name] = {
                loading: false,
                items: matched,
              };
              return;
            }
            const localValue = nextDetail[relation.localField as keyof ${modelName}Detail];
            if (relation.type === 'manyToMany') {
              if (!relation.lookupApi) {
                nextState[relation.name] = {
                  loading: false,
                  items: [],
                  unsupported: true,
                };
                return;
              }
              const [bindingRows, optionRows] = await Promise.all([
                loadManyToManyRelationRows(scope, moduleName, localValue, relation.name),
                loadLookupRelationRows(relation.lookupApi),
              ]);
              nextState[relation.name] = {
                loading: false,
                items: mergeManyToManyRelationRows(bindingRows, optionRows, relation),
              };
              return;
            }
            if (localValue === undefined || localValue === null || localValue === '') {
              nextState[relation.name] = { loading: false, items: [] };
              return;
            }
            const listResult = await loadModuleRelationRows(
              scope,
              relation.targetModule,
              relation.targetField,
              localValue,
            );
            nextState[relation.name] = {
              loading: false,
              items: listResult,
            };
          } catch {
            nextState[relation.name] = {
              loading: false,
              items: [],
              error: true,
            };
          }
        }),
      );
      setRelatedData(nextState);
    },
    [moduleName, scope],
  );

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  useEffect(() => {
    const childRelations = governanceRelations.filter(
      (relation) => relation.type === 'oneToMany' && !relation.lookupApi,
    );
    if (childRelations.length === 0) {
      return;
    }
    void Promise.all(
      childRelations.map(async (relation) => {
        try {
          const schema = await loadManagedModuleSchema(
            getGeneratedModuleSchema,
            scope,
            relation.targetModule,
          );
          setChildSchemas((prev) => ({ ...prev, [relation.name]: schema }));
        } catch {
          setChildSchemaErrors((prev) => ({ ...prev, [relation.name]: true }));
        }
      }),
    );
  }, [scope]);

  useEffect(() => {
    if (!detail) {
      return;
    }
    void loadRelationData(detail);
  }, [detail, loadRelationData]);

  const openChildCreateModal = useCallback(
    async (relation: (typeof governanceRelations)[number]) => {
      if (!detail) {
        return;
      }
      const schema = childSchemas[relation.name];
      if (!schema) {
        message.error(t('generator.wizard.result.childTableSchemaLoadFailed'));
        return;
      }
      childForm.setFieldsValue(
        buildChildRelationInitialValues(relation, detail as unknown as Record<string, unknown>),
      );
      setEditingChildRelation({ relationName: relation.name, mode: 'create' });
      await loadChildRelationOptions(schema);
    },
    [childForm, childSchemas, detail, loadChildRelationOptions, t],
  );

  const openChildEditModal = useCallback(
    async (relation: (typeof governanceRelations)[number], record: RelationRuntimeRow) => {
      if (!detail) {
        return;
      }
      const schema = childSchemas[relation.name];
      if (!schema) {
        message.error(t('generator.wizard.result.childTableSchemaLoadFailed'));
        return;
      }
      childForm.setFieldsValue(
        buildChildRelationInitialValues(
          relation,
          detail as unknown as Record<string, unknown>,
          record,
        ),
      );
      setEditingChildRelation({ relationName: relation.name, mode: 'edit', record });
      await loadChildRelationOptions(schema);
    },
    [childForm, childSchemas, detail, loadChildRelationOptions, t],
  );

  const closeChildDialog = useCallback(() => {
    setEditingChildRelation(null);
    childForm.resetFields();
  }, [childForm]);

  const openRelationBindModal = useCallback(
    async (relation: (typeof governanceRelations)[number]) => {
      if (!detail || relation.type !== 'manyToMany' || !relation.lookupApi) {
        return;
      }
      const optionRows = await loadLookupRelationRows(relation.lookupApi);
      const options = normalizeManyToManyBindingOptions(optionRows, relation);
      const selectedTargetIds = (relatedData[relation.name]?.items || [])
        .map((row) => resolveRelationRowValue(row, relation))
        .filter((value): value is string | number => value !== null);
      bindForm.setFieldsValue({ targetIds: selectedTargetIds });
      setBindingRelation({
        relationName: relation.name,
        options,
      });
    },
    [bindForm, detail, relatedData],
  );

  const closeRelationBindModal = useCallback(() => {
    setBindingRelation(null);
    bindForm.resetFields();
  }, [bindForm]);

  const submitChildRelationForm = useCallback(async () => {
    if (!detail || !editingChildRelation) {
      return;
    }
    const relation =
      governanceRelations.find((item) => item.name === editingChildRelation.relationName) || null;
    if (!relation) {
      return;
    }

    let values: Record<string, unknown>;
    try {
      values = await childForm.validate();
    } catch (submitError) {
      if (isArcoFormValidationError(submitError)) {
        return;
      }
      throw submitError;
    }

    setChildSubmitting(true);
    try {
      const payload = {
        ...values,
        [relation.targetField]: detail[relation.localField as keyof ${modelName}Detail],
      };
      if (editingChildRelation.mode === 'edit') {
        const recordId = Number(editingChildRelation.record?.id ?? 0);
        if (!recordId) {
          message.error(t('common.actionFailed'));
          return;
        }
        await apiRequest({
          url: \`\${buildModuleCrudBasePath(scope, relation.targetModule)}/\${recordId}\`,
          method: 'put',
          data: payload,
        });
        message.success(t('common.updateSuccess'));
      } else {
        await apiRequest({
          url: buildModuleCrudBasePath(scope, relation.targetModule),
          method: 'post',
          data: payload,
        });
        message.success(t('common.createSuccess'));
      }
      closeChildDialog();
      await loadRelationData(detail);
    } finally {
      setChildSubmitting(false);
    }
  }, [childForm, closeChildDialog, detail, editingChildRelation, loadRelationData, scope, t]);

  const submitManyToManyBinding = useCallback(async () => {
    if (!detail || !bindingRelation) {
      return;
    }
    const relation = governanceRelations.find((item) => item.name === bindingRelation.relationName) || null;
    if (!relation) {
      return;
    }

    let values: Record<string, unknown>;
    try {
      values = await bindForm.validate();
    } catch (submitError) {
      if (isArcoFormValidationError(submitError)) {
        return;
      }
      throw submitError;
    }

    const targetIds = Array.isArray(values.targetIds)
      ? values.targetIds.filter((item) => item !== null && item !== undefined && item !== '')
      : [];

    setManyToManySubmitting(true);
    try {
      await apiRequest({
        url: buildManyToManyRelationBasePath(scope, moduleName, detail.id, relation.name),
        method: 'post',
        data: { targetIds: targetIds.map((item) => String(item)) },
      });
      message.success(t('common.updateSuccess'));
      closeRelationBindModal();
      await loadRelationData(detail);
    } finally {
      setManyToManySubmitting(false);
    }
  }, [bindForm, bindingRelation, closeRelationBindModal, detail, loadRelationData, moduleName, scope, t]);

  const unbindManyToManyRelation = useCallback(
    async (relation: (typeof governanceRelations)[number], record: RelationRuntimeRow) => {
      if (!detail) {
        return;
      }
      const targetId = resolveRelationRowValue(record, relation);
      if (targetId === null) {
        message.error(t('common.actionFailed'));
        return;
      }
      await apiRequest({
        url: buildManyToManyRelationItemPath(scope, moduleName, detail.id, relation.name, targetId),
        method: 'delete',
      });
      message.success(t('common.updateSuccess'));
      await loadRelationData(detail);
    },
    [detail, loadRelationData, moduleName, scope, t],
  );

  const activeChildRelation =
    editingChildRelation
      ? governanceRelations.find((item) => item.name === editingChildRelation.relationName) || null
      : null;
  const activeChildSchema = activeChildRelation ? childSchemas[activeChildRelation.name] || null : null;
  const editableChildFields =
    activeChildRelation && activeChildSchema
      ? buildEditableChildFields(activeChildRelation, activeChildSchema)
      : [];
  const activeChildContracts = activeChildSchema ? buildChildRelationContractMap(activeChildSchema) : {};

  if (loading) {
    return <PageLoading />;
  }

  if (error) {
    return <PageError onRetry={() => { void loadDetail(); }} />;
  }

  if (!detail) {
    return <PageEmpty />;
  }

  return (
    <PageContainer>
      <Card bordered={false}>
        <Typography.Title heading={5}>{t('${titleKey}')}</Typography.Title>
        <Descriptions column={1} data={[
${this.generateDetailDescriptionItems()}
        ]} />
      </Card>
      <Card bordered={false}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Typography.Title heading={6}>
            {t('generator.wizard.result.contractTitle')}
          </Typography.Title>
          <Space wrap>
            <Tag color="purple">{t(\`generator.wizard.tableRole.\${governanceTableRole}\`)}</Tag>
            {governancePrimaryTable ? (
              <Tag color="arcoblue">
                {t('generator.wizard.primaryTable')}: {governancePrimaryTable}
              </Tag>
            ) : null}
            {governanceRelationFromField ? <Tag color="orange">{governanceRelationFromField}</Tag> : null}
            {governanceRelationToField ? <Tag color="orange">{governanceRelationToField}</Tag> : null}
          </Space>
          {governanceDependencies.length > 0 ? (
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              <Typography.Text type="secondary">
                {t('generator.wizard.result.dependencies')}
              </Typography.Text>
              {governanceDependencies.map((dependency) => (
                <Typography.Text key={dependency.module} code>
                  {dependency.module}
                  {dependency.reason ? \` · \${dependency.reason}\` : ''}
                </Typography.Text>
              ))}
            </Space>
          ) : null}
          {governanceRelations.length > 0 ? (
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              <Typography.Text type="secondary">
                {t('generator.wizard.result.relations')}
              </Typography.Text>
              {governanceRelations.map((relation) => (
                <Typography.Text key={\`\${relation.name}-\${relation.targetModule}\`} code>
                  {relation.name} · {relation.type} · {relation.targetModule} · {relation.localField} → {relation.targetField}
                  {relation.junctionTable ? \` · \${relation.junctionTable}\` : ''}
                </Typography.Text>
              ))}
            </Space>
          ) : null}
        </Space>
      </Card>
      {governanceRelations.length > 0 ? (
        <Card bordered={false}>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Typography.Title heading={6}>
              {t('generator.wizard.result.relatedData')}
            </Typography.Title>
            {governanceRelations.map((relation) => {
              const relationState = relatedData[relation.name];
              const rows = relationState?.items || [];
              return (
                <Card key={relation.name} size="small">
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space wrap>
                      <Typography.Text code>{relation.name}</Typography.Text>
                      <Tag color="arcoblue">{relation.type}</Tag>
                      <Typography.Text type="secondary">{relation.targetModule}</Typography.Text>
                      {resolveEditableChildRelation(
                        relation,
                        childSchemas[relation.name] || null,
                        '${tableName}',
                      ) ? (
                        <Button
                          size="mini"
                          type="primary"
                          onClick={() => {
                            void openChildCreateModal(relation);
                          }}
                        >
                          {t('generator.wizard.result.childTableCreate')}
                        </Button>
                      ) : relation.type === 'manyToMany' && relation.lookupApi ? (
                        <Button
                          size="mini"
                          type="primary"
                          onClick={() => {
                            void openRelationBindModal(relation);
                          }}
                        >
                          {t('generator.wizard.result.relatedDataBind')}
                        </Button>
                      ) : null}
                      <Button
                        size="mini"
                        onClick={() => navigate(buildRelationRuntimePath(scope, relation.targetModule))}
                      >
                        {t('generator.wizard.result.openRelatedModule')}
                      </Button>
                    </Space>
                    {childSchemaErrors[relation.name] ? (
                      <Alert type="warning" content={t('generator.wizard.result.childTableSchemaLoadFailed')} />
                    ) : null}
                    {relationState?.loading ? (
                      <Typography.Text type="secondary">{t('common.loading')}</Typography.Text>
                    ) : relationState?.unsupported ? (
                      <Alert type="info" content={t('generator.wizard.result.relatedDataUnsupported')} />
                    ) : relationState?.error ? (
                      <Alert type="warning" content={t('generator.wizard.result.relatedDataLoadFailed')} />
                    ) : rows.length === 0 ? (
                      <Typography.Text type="secondary">{t('common.noData')}</Typography.Text>
                    ) : (
                      <Table
                        size="small"
                        pagination={false}
                        rowKey={(record) =>
                          String(
                            record.id ??
                              record[relation.targetField] ??
                              record[relation.lookupValueField || relation.targetField] ??
                              relation.name,
                          )
                        }
                        data={rows.slice(0, 5)}
                        columns={buildRelationRuntimeColumns(
                          relation,
                          rows,
                          resolveEditableChildRelation(
                            relation,
                            childSchemas[relation.name] || null,
                            '${tableName}',
                          )
                            ? (record) => {
                                void openChildEditModal(relation, record);
                              }
                            : undefined,
                          relation.type === 'manyToMany'
                            ? (record) => {
                                void unbindManyToManyRelation(relation, record);
                              }
                            : undefined,
                          t('generator.wizard.result.childTableActions'),
                          t('generator.wizard.result.childTableEdit'),
                          t('generator.wizard.result.relatedDataUnbind'),
                        )}
                      />
                    )}
                  </Space>
                </Card>
              );
            })}
          </Space>
        </Card>
      ) : null}
      <AppModal
        title={t('generator.wizard.result.relatedDataBind')}
        visible={Boolean(bindingRelation)}
        size="md"
        onCancel={closeRelationBindModal}
        footer={
          <SubmitBar
            onCancel={closeRelationBindModal}
            onSubmit={() => {
              void submitManyToManyBinding();
            }}
            loading={manyToManySubmitting}
            submitText={t('common.save')}
          />
        }
      >
        <Form form={bindForm} layout="vertical">
          <FormItem
            field="targetIds"
            label={bindingRelation?.relationName || 'targetIds'}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Select
              mode="multiple"
              allowClear
              options={bindingRelation?.options || []}
              placeholder={t('generator.wizard.result.relatedDataBind')}
            />
          </FormItem>
        </Form>
      </AppModal>
      <AppModal
        title={
          editingChildRelation?.mode === 'edit'
            ? t('generator.wizard.result.childTableDialogEdit')
            : t('generator.wizard.result.childTableDialogCreate')
        }
        visible={Boolean(editingChildRelation)}
        size="lg"
        onCancel={closeChildDialog}
        footer={
          <SubmitBar
            onCancel={closeChildDialog}
            onSubmit={() => {
              void submitChildRelationForm();
            }}
            loading={childSubmitting}
            submitText={
              editingChildRelation?.mode === 'edit' ? t('common.save') : t('common.create')
            }
          />
        }
        unmountOnExit
      >
        <Form
          form={childForm}
          layout="vertical"
          onSubmit={() => {
            void submitChildRelationForm();
          }}
        >
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            {activeChildRelation && activeChildSchema ? (
              editableChildFields.length > 0 ? (
                editableChildFields.map((field) => {
                  const optionKey = buildChildRelationOptionKey(activeChildSchema.name, field.name);
                  return (
                    <FormItem
                      key={field.name}
                      label={field.label || field.name}
                      field={field.name}
                      rules={field.required ? [{ required: true }] : undefined}
                      triggerPropName={field.type === 'bool' ? 'checked' : undefined}
                    >
                      {renderChildFieldInput(
                        field,
                        activeChildContracts[field.name],
                        childRelationOptions[optionKey] || [],
                        Boolean(childRelationLoading[optionKey]),
                      )}
                    </FormItem>
                  );
                })
              ) : (
                <Alert type="info" content={t('generator.wizard.result.childTableNoEditableFields')} />
              )
            ) : (
              <Alert type="warning" content={t('generator.wizard.result.childTableSchemaLoadFailed')} />
            )}
          </Space>
        </Form>
      </AppModal>
    </PageContainer>
  );
};

export default ${modelName}DetailPage;
`;
  }

  /**
   * 生成空查询字段
   */
  private generateEmptyQueryFields(): string {
    const searchableFields = this.schema.model.fields.filter((f) => f.searchable);

    return searchableFields
      .map((field) => {
        const defaultValue = this.getDefaultValue(field.type);
        return `  ${field.name}: ${defaultValue},`;
      })
      .join('\n');
  }

  private generateFormItems(): string {
    return this.schema.model.fields
      .filter((field) => field.visibleInForm !== false)
      .map((field) => this.renderFormItem(field))
      .join('\n');
  }

  private renderFormItem(field: ModuleSchema['model']['fields'][number]): string {
    const labelKey = buildFieldLabelKey(this.schema.scope, this.schema.name, field.name);
    const placeholderKey = buildFieldPlaceholderKey(this.schema.scope, this.schema.name, field.name);
    const rules = field.required ? ` rules={[{ required: true }]}` : '';

    if (field.type === 'enum') {
      const options = (field.enumOptions ?? [])
        .map(
          (option) =>
            `                <Select.Option value="${option.value}">{t('${buildEnumOptionKey(this.schema.scope, this.schema.name, field.name, option.value)}')}</Select.Option>`,
        )
        .join('\n');
      return `      <FormItem label={t('${labelKey}')} field="${field.name}"${rules}>
        <Select allowClear>
${options}
        </Select>
      </FormItem>`;
    }

    if (field.type === 'relation') {
      return `      <FormItem label={t('${labelKey}')} field="${field.name}"${rules}>
        <Select
          allowClear
          loading={Boolean(relationLoading['${field.name}'])}
          disabled={!relationLookupContractMap['${field.name}']?.lookupApi}
          placeholder={t('${placeholderKey}')}
          options={relationOptions['${field.name}'] || []}
        />
        <Space direction="vertical" size={4} style={{ width: '100%' }}>
          <Typography.Text type="secondary">
            {t('generator.fieldEditor.type.relation')} · ${field.name} · ${this.buildRelationFieldTargetHint(field.name)}
          </Typography.Text>
          <Typography.Text type="secondary">
            {relationLookupContractMap['${field.name}']?.lookupApi || relationLookupContractMap['${field.name}']?.targetModule || 'lookup pending'}
          </Typography.Text>
          <Typography.Text type="secondary">
            {relationLookupContractMap['${field.name}']?.targetLabelField || relationLookupContractMap['${field.name}']?.targetField || 'label pending'}
          </Typography.Text>
        </Space>
      </FormItem>`;
    }

    if (field.type === 'int' || field.type === 'float') {
      return `      <FormItem label={t('${labelKey}')} field="${field.name}"${rules}>
        <InputNumber style={{ width: '100%' }} />
      </FormItem>`;
    }

    if (field.type === 'bool') {
      return `      <FormItem label={t('${labelKey}')} field="${field.name}" triggerPropName="checked">
        <Switch />
      </FormItem>`;
    }

    return `      <FormItem label={t('${labelKey}')} field="${field.name}"${rules}>
        <Input />
      </FormItem>`;
  }

  private generateDetailDescriptionItems(): string {
    return this.schema.model.fields
      .map((field) => {
        const labelKey = buildFieldLabelKey(this.schema.scope, this.schema.name, field.name);
        return `          { label: t('${labelKey}'), value: String(detail.${field.name} ?? '-') },`;
      })
      .join('\n');
  }

  private buildDetailComponentKey(scope: string, name: string): string {
    return `${scope}/${splitModuleSegments(name).join('/')}/${this.modelName}Detail`;
  }

  private generateListGovernanceConstants(): string {
    const tableRole = this.schema.metadata?.tableRole || 'main';
    const primaryTable = this.schema.metadata?.primaryTable || '';
    const relationFromField = this.schema.metadata?.relationFromField || '';
    const relationToField = this.schema.metadata?.relationToField || '';

    return `const governanceTableRole = '${tableRole}';
const governancePrimaryTable = '${primaryTable}';
const governanceRelationFromField = '${relationFromField}';
const governanceRelationToField = '${relationToField}';
`;
  }

  private shouldRenderListGovernanceNotice(): boolean {
    const tableRole = this.schema.metadata?.tableRole || 'main';
    const primaryTable = this.schema.metadata?.primaryTable || '';
    const relationFromField = this.schema.metadata?.relationFromField || '';
    const relationToField = this.schema.metadata?.relationToField || '';

    return (
      tableRole === 'detail' ||
      tableRole === 'relation' ||
      Boolean(primaryTable) ||
      Boolean(relationFromField) ||
      Boolean(relationToField)
    );
  }

  private generateWorkbenchTableColumns(includeRowActions: boolean): string {
    const listFields = this.schema.model.fields.filter((field) => field.visibleInList !== false);
    const columns = listFields.map((field) => {
      const labelKey = buildFieldLabelKey(this.schema.scope, this.schema.name, field.name);
      return `                {
                  title: t('${labelKey}'),
                  dataIndex: '${field.name}',
                },`;
    });

    if (includeRowActions) {
      columns.push(`                {
                  title: t('common.action'),
                  width: TABLE_ACTION_COLUMN_WIDTH.medium,
                  fixed: 'right',
                  render: (_: unknown, row: ${this.modelName}ListRow) => (
                    <SystemRowActions
                      actions={[
${this.generateRowActionItems()}
                      ]}
                    />
                  ),
                },`);
    }

    return columns.join('\n');
  }

  private generateRowActionItems(): string {
    const actions = getPageActions(this.schema);
    const items: string[] = [];

    if (actions.includes('view') || actions.includes('detail')) {
      items.push(`                        {
                          key: 'detail',
                          text: t('common.detail'),
                          icon: <IconEye />,
                          onClick: () => navigate('${buildRoutePath(this.schema.scope, this.schema.name)}/' + row.id),
                        },`);
    }
    if (actions.includes('delete')) {
      items.push(`                        {
                          key: 'delete',
                          text: t('common.delete'),
                          icon: <IconDelete />,
                          status: 'danger',
                          confirm: {
                            title: t('common.deleteConfirm'),
                            onOk: async () => {
                              await delete${this.modelName}(row.id);
                              message.success(t('common.deleteSuccess'));
                              await loadData(query);
                            },
                          },
                        },`);
    }

    return items.join('\n');
  }

  private generateListActionImports(): string {
    const imports: string[] = [];
    if (this.hasPageAction('delete')) {
      imports.push(`delete${this.modelName}`);
    }
    if (this.hasPageAction('export')) {
      imports.push(`export${this.modelName}s`);
    }
    if (this.hasPageAction('import')) {
      imports.push(`import${this.modelName}s`);
    }
    return imports.length > 0 ? `${imports.join(', ')} ,` : '';
  }

  private hasPageAction(action: string): boolean {
    return getPageActions(this.schema).includes(action as never);
  }

  private isListGovernanceEnabled(): boolean {
    return this.schema.listLayout?.governance !== false && this.shouldRenderListGovernanceNotice();
  }

  private isListSearchEnabled(): boolean {
    const hasSearchableFields = this.schema.model.fields.some((field) => field.searchable);
    return hasSearchableFields && this.schema.listLayout?.search !== false;
  }

  private isListHeaderActionsEnabled(): boolean {
    const actions = getPageActions(this.schema);
    return (
      this.schema.listLayout?.headerActions !== false &&
      (actions.includes('create') || actions.includes('export') || actions.includes('import'))
    );
  }

  private isListBatchActionsEnabled(): boolean {
    const actions = getPageActions(this.schema);
    return this.schema.listLayout?.batchActions !== false && actions.includes('delete');
  }

  private isListRowActionsEnabled(): boolean {
    const actions = getPageActions(this.schema);
    return (
      this.schema.listLayout?.rowActions !== false &&
      (actions.includes('view') ||
        actions.includes('detail') ||
        actions.includes('delete') ||
        actions.includes('update'))
    );
  }

  private generateWorkbenchGovernanceBlock(): string {
    if (!this.isListGovernanceEnabled()) {
      return '';
    }
    return `        <GovernanceSummaryBar
          eyebrow={t('generator.wizard.listLayout.governance', 'Governance')}
          title={t('${buildTitleKey(this.schema.scope, this.schema.name)}')}
          description={t('generator.wizard.relationFields.help')}
          metrics={[
            {
              key: 'tableRole',
              label: t('generator.wizard.tableRole'),
              value: t(\`generator.wizard.tableRole.\${governanceTableRole}\`),
            },
            {
              key: 'primaryTable',
              label: t('generator.wizard.primaryTable'),
              value: governancePrimaryTable || '-',
            },
          ]}
        />`;
  }

  private generateWorkbenchSearchBlock(): string {
    if (!this.isListSearchEnabled()) {
      return '';
    }
    const searchableFields = this.schema.model.fields.filter((field) => field.searchable);
    return `        <FilterPanel>
          <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
            <Row gutter={16}>
${searchableFields
  .map(
    (field) => `              <Col xs={24} md={6}>
                <FormItem label={t('${buildFieldLabelKey(this.schema.scope, this.schema.name, field.name)}')} field="${field.name}">
                  ${this.renderSearchField(field)}
                </FormItem>
              </Col>`,
  )
  .join('\n')}
              <Col xs={24} md={6}>
                <FormItem className="filter-panel__action-item">
                  <Space>
                    <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                      {t('common.search')}
                    </Button>
                    <Button onClick={reset}>{t('common.reset')}</Button>
                  </Space>
                </FormItem>
              </Col>
            </Row>
          </Form>
        </FilterPanel>`;
  }

  private renderSearchField(field: ModuleSchema['model']['fields'][number]): string {
    if (field.type === 'enum') {
      const options = (field.enumOptions ?? [])
        .map(
          (option) =>
            `{ label: t('${buildEnumOptionKey(this.schema.scope, this.schema.name, field.name, option.value)}'), value: '${option.value}' }`,
        )
        .join(', ');
      return `<Select allowClear options={[${options}]} />`;
    }
    return `<Input onPressEnter={() => queryForm.submit()} />`;
  }

  private generateWorkbenchSummaryHead(titleKey: string): string {
    return `<div className="system-list__table-head">
            <div className="system-list__table-head-copy">
              <Typography.Text className="system-list__table-head-title">
                {t('${titleKey}')}
              </Typography.Text>
              <Typography.Text type="secondary" className="system-list__table-head-desc">
                {t('common.total', { count: total })}
              </Typography.Text>
            </div>
          </div>`;
  }

  private generateWorkbenchHeaderActions(
    headerActionsEnabled: boolean,
    batchActionsEnabled: boolean,
  ): string {
    if (!headerActionsEnabled && !batchActionsEnabled) {
      return '';
    }

    const utilityActions: string[] = [];
    const primaryActions: string[] = [];

    if (this.hasPageAction('export')) {
      utilityActions.push(`                  <Button icon={<IconDownload />} onClick={() => { void handleExport(); }}>
                    {t('common.export')}
                  </Button>`);
    }
    if (this.hasPageAction('import')) {
      utilityActions.push(`                  <ImportCsvButton onSelect={(file) => { void handleImport(file); }}>
                    {t('common.import')}
                  </ImportCsvButton>`);
    }
    if (this.hasPageAction('create')) {
      primaryActions.push(`                  <Button
                    type="primary"
                    icon={<IconPlus />}
                    onClick={() => {
                      message.info(t('common.comingSoon'));
                    }}
                  >
                    {t('common.add')}
                  </Button>`);
    }

    return `          <TableBatchActionBar
            selectedCount={selectedRowKeys.length}
            selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
            clearText={t('common.clearSelection')}
            clearSuccessText={t('common.clearSelectionSuccess')}
            onClear={() => setSelectedRowKeys([])}
            prefixActions={
              <ListHeaderActions
                utility={
                  <>
${utilityActions.join('\n')}
                  </>
                }
                primary={
                  <>
${primaryActions.join('\n')}
                  </>
                }
              />
            }
            ${batchActionsEnabled ? `actions={
              <Popconfirm
                title={t('common.deleteConfirm')}
                onOk={() => {
                  void handleBatchDelete();
                }}
                disabled={selectedRowKeys.length === 0 || submitting}
              >
                <Button
                  status="danger"
                  icon={<IconDelete />}
                  disabled={selectedRowKeys.length === 0 || submitting}
                  loading={submitting}
                >
                  {t('common.deleteSelected')}
                </Button>
              </Popconfirm>
            }` : ''}
          />`;
  }

  private generateGovernanceConstants(): string {
    const dependencies = JSON.stringify(this.schema.dependencies ?? [], null, 2);
    const relations = JSON.stringify(this.schema.relations ?? [], null, 2);
    const tableRole = this.schema.metadata?.tableRole || 'main';
    const primaryTable = this.schema.metadata?.primaryTable || '';
    const relationFromField = this.schema.metadata?.relationFromField || '';
    const relationToField = this.schema.metadata?.relationToField || '';

    return `const governanceDependencies = ${dependencies} as Array<{
  module: string;
  required?: boolean;
  reason?: string;
}>;

const governanceRelations = ${relations} as Array<{
  name: string;
  type: string;
  targetModule: string;
  localField: string;
  targetField: string;
  targetLabelField?: string;
  lookupApi?: string;
  lookupValueField?: string;
  junctionTable?: string;
}>;

const governanceTableRole = '${tableRole}';
const governancePrimaryTable = '${primaryTable}';
const governanceRelationFromField = '${relationFromField}';
const governanceRelationToField = '${relationToField}';
`;
  }

  private generateRelationLookupContracts(): string {
    const relationFields = this.schema.model.fields.filter((field) => field.type === 'relation');
    if (relationFields.length === 0) {
      return '';
    }

    const contracts = relationFields.map((field) => {
      const relation = (this.schema.relations ?? []).find((item) => item.localField === field.name);
      return {
        field: field.name,
        targetModule: relation?.targetModule || '',
        targetField: relation?.targetField || '',
        targetLabelField: relation?.targetLabelField || '',
        lookupApi: relation?.lookupApi || '',
        lookupValueField: relation?.lookupValueField || relation?.targetField || '',
        type: relation?.type || 'lookup',
      };
    });

    return `const relationLookupContractMap = ${JSON.stringify(contracts, null, 2)}.reduce<Record<string, {
  field: string;
  targetModule: string;
  targetField: string;
  targetLabelField: string;
  lookupApi: string;
  lookupValueField: string;
  type: string;
}>>((acc, item) => {
  acc[item.field] = item;
  return acc;
}, {});
`;
  }

  private generateRelationLookupImports(): string {
    const relationFields = this.schema.model.fields.filter((field) => field.type === 'relation');
    if (relationFields.length === 0) {
      return '';
    }
    return relationFields
      .map((field) => `get${this.toPascalCase(field.name)}RelationOptions, `)
      .join('');
  }

  private generateRelationLookupLoaderEffect(): string {
    const relationFields = this.schema.model.fields.filter((field) => field.type === 'relation');
    if (relationFields.length === 0) {
      return '';
    }

    const calls = relationFields
      .map((field) => {
        const functionName = this.toPascalCase(field.name);
        return `      if (relationLookupContractMap['${field.name}']?.lookupApi) {
        nextLoading['${field.name}'] = true;
        loaders.push(
          get${functionName}RelationOptions()
            .then((options) => {
              nextOptions['${field.name}'] = options;
            })
            .finally(() => {
              setRelationLoading((prev) => ({ ...prev, ['${field.name}']: false }));
            }),
        );
      }`;
      })
      .join('\n');

    return `  useEffect(() => {
    const nextOptions: Record<string, Array<{ label: string; value: string | number }>> = {};
    const nextLoading: Record<string, boolean> = {};
    const loaders: Array<Promise<void>> = [];
${calls}
    setRelationLoading(nextLoading);
    if (loaders.length === 0) {
      return;
    }
    void Promise.all(loaders).then(() => {
      setRelationOptions(nextOptions);
    });
  }, []);
`;
  }

  private generateRelationFormStateBlock(): string {
    const relationFields = this.schema.model.fields.filter((field) => field.type === 'relation');
    if (relationFields.length === 0) {
      return '';
    }
    return `  const [relationOptions, setRelationOptions] = useState<Record<string, Array<{ label: string; value: string | number }>>>({});
  const [relationLoading, setRelationLoading] = useState<Record<string, boolean>>({});`;
  }

  private buildFormComponentImports(): string[] {
    const imports = new Set<string>(['Form', 'Input']);

    for (const field of this.schema.model.fields.filter((item) => item.visibleInForm !== false)) {
      if (field.type === 'enum' || field.type === 'relation') {
        imports.add('Select');
      }
      if (field.type === 'int' || field.type === 'float') {
        imports.add('InputNumber');
      }
      if (field.type === 'bool') {
        imports.add('Switch');
      }
    }

    return Array.from(imports);
  }

  private generateRelationLookupApi(): string {
    const relationFields = this.schema.model.fields.filter((field) => field.type === 'relation');
    if (relationFields.length === 0) {
      return '';
    }

    const helper = `type RelationLookupRawItem = Record<string, unknown>;

function normalizeRelationOptionValue(value: unknown): string | number {
  if (typeof value === 'number' || typeof value === 'string') {
    return value;
  }
  return String(value ?? '');
}

function normalizeRelationOptionLabel(value: unknown, fallback: string | number): string {
  if (typeof value === 'string' && value.trim()) {
    return value;
  }
  if (typeof value === 'number') {
    return String(value);
  }
  return String(fallback);
}

function normalizeRelationOptionRows(payload: unknown): RelationLookupRawItem[] {
  if (Array.isArray(payload)) {
    return payload as RelationLookupRawItem[];
  }
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>;
    if (Array.isArray(record.items)) {
      return record.items as RelationLookupRawItem[];
    }
    if (Array.isArray(record.list)) {
      return record.list as RelationLookupRawItem[];
    }
    if (Array.isArray(record.rows)) {
      return record.rows as RelationLookupRawItem[];
    }
    if (Array.isArray(record.data)) {
      return record.data as RelationLookupRawItem[];
    }
  }
  return [];
}
`;

    const functions = relationFields
      .map((field) => {
        const relation = (this.schema.relations ?? []).find((item) => item.localField === field.name);
        const functionName = this.toPascalCase(field.name);
        const labelField = relation?.targetLabelField || relation?.targetField || 'label';
        const valueField = relation?.lookupValueField || relation?.targetField || 'value';
        return `
export interface ${functionName}RelationOption {
  label: string;
  value: string | number;
  raw?: Record<string, unknown>;
}

export const ${functionName}RelationLookupContract = {
  field: '${field.name}',
  targetModule: '${relation?.targetModule || ''}',
  targetField: '${relation?.targetField || ''}',
  targetLabelField: '${relation?.targetLabelField || ''}',
  lookupApi: '${relation?.lookupApi || ''}',
  lookupValueField: '${relation?.lookupValueField || relation?.targetField || ''}',
} as const;

export async function get${functionName}RelationOptions(): Promise<${functionName}RelationOption[]> {
  if (!${functionName}RelationLookupContract.lookupApi) {
    return [];
  }
  const payload = await apiRequest<unknown>({
    url: ${functionName}RelationLookupContract.lookupApi,
    method: 'get',
  });
  return normalizeRelationOptionRows(payload).map((item) => {
    const rawValue = item['${valueField}'] ?? item.value ?? item.id ?? '';
    const value = normalizeRelationOptionValue(rawValue);
    const rawLabel = item['${labelField}'] ?? item.label ?? item.name;
    return {
      label: normalizeRelationOptionLabel(rawLabel, value),
      value,
      raw: item,
    };
  });
}`;
      })
      .join('\n');

    return `${helper}\n${functions}`;
  }

  private generateManyToManyRelationApi(): string {
    const manyToManyRelations = (this.schema.relations ?? []).filter(
      (relation) => relation.type === 'manyToMany',
    );
    if (manyToManyRelations.length === 0) {
      return '';
    }

    const routePath = buildRoutePath(this.schema.scope, this.schema.name);
    return manyToManyRelations
      .map((relation) => {
        const functionName = this.toPascalCase(relation.name);
        return `export interface ${functionName}RelationBindPayload {
  targetIds: number[] | string[];
}

export function bind${functionName}Relation(id: number, data: ${functionName}RelationBindPayload) {
  return apiRequest<{ success: boolean }>({
    url: \`${routePath}/\${id}/relations/${relation.name}\`,
    method: 'post',
    data: {
      targetIds: data.targetIds.map((item) => String(item)),
    },
  });
}

export function unbind${functionName}Relation(id: number, targetId: number | string) {
  return apiRequest<{ success: boolean }>({
    url: \`${routePath}/\${id}/relations/${relation.name}/\${targetId}\`,
    method: 'delete',
  });
}`;
      })
      .join('\n\n');
  }

  private generateRelationRuntimeHelpers(): string {
    return `type RelationRuntimeRow = Record<string, unknown>;

type ChildModuleField = {
  name: string;
  type: string;
  label?: string;
  required?: boolean;
  visibleInForm?: boolean;
  placeholder?: string;
  enumOptions?: Array<{ value: string; label: string }>;
};

type ChildModuleRelation = {
  name: string;
  type: string;
  localField: string;
  targetField: string;
  targetLabelField?: string;
  lookupApi?: string;
  lookupValueField?: string;
};

interface ChildModuleSchema {
  name: string;
  displayName: string;
  scope: string;
  relations?: ChildModuleRelation[];
  metadata?: {
    tableRole?: string;
    primaryTable?: string;
  };
  model: {
    tableName: string;
    fields: ChildModuleField[];
  };
}

interface RelationRuntimeState {
  loading: boolean;
  items: RelationRuntimeRow[];
  error?: boolean;
  unsupported?: boolean;
}

interface RelationEditorState {
  relationName: string;
  mode: 'create' | 'edit';
  record?: RelationRuntimeRow;
}

interface ManyToManyBindingState {
  relationName: string;
  options: RelationEditorOption[];
}

interface RelationEditorOption {
  label: string;
  value: string | number;
}

interface ChildRelationContract {
  field: string;
  lookupApi?: string;
  targetField?: string;
  targetLabelField?: string;
  lookupValueField?: string;
}

function normalizeRelationOptionValue(value: unknown): string | number {
  if (typeof value === 'number' || typeof value === 'string') {
    return value;
  }
  return String(value ?? '');
}

function normalizeRelationOptionLabel(value: unknown, fallback: string | number): string {
  if (typeof value === 'string' && value.trim()) {
    return value;
  }
  if (typeof value === 'number') {
    return String(value);
  }
  return String(fallback);
}

function normalizeRelationOptionRows(payload: unknown): RelationRuntimeRow[] {
  if (Array.isArray(payload)) {
    return payload as RelationRuntimeRow[];
  }
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>;
    if (Array.isArray(record.items)) {
      return record.items as RelationRuntimeRow[];
    }
    if (Array.isArray(record.list)) {
      return record.list as RelationRuntimeRow[];
    }
    if (Array.isArray(record.rows)) {
      return record.rows as RelationRuntimeRow[];
    }
    if (Array.isArray(record.data)) {
      return record.data as RelationRuntimeRow[];
    }
  }
  return [];
}

async function loadLookupRelationRows(url: string): Promise<RelationRuntimeRow[]> {
  const payload = await apiRequest<unknown>({
    url,
    method: 'get',
    skipErrorMessage: true,
  });
  return normalizeRelationOptionRows(payload);
}

async function loadModuleRelationRows(
  scope: string,
  moduleName: string,
  targetField: string,
  targetValue: unknown,
): Promise<RelationRuntimeRow[]> {
  const payload = await apiRequest<{ items?: RelationRuntimeRow[] }>({
    url: \`/\${scope}/\${moduleName}/list\`,
    method: 'get',
    params: {
      [targetField]: targetValue,
      page: 1,
      pageSize: 5,
    },
    skipErrorMessage: true,
  });
  return Array.isArray(payload?.items) ? payload.items : [];
}

async function loadManyToManyRelationRows(
  scope: string,
  moduleName: string,
  ownerId: unknown,
  relationName: string,
): Promise<RelationRuntimeRow[]> {
  if (ownerId === undefined || ownerId === null || ownerId === '') {
    return [];
  }
  const payload = await apiRequest<{ items?: RelationRuntimeRow[] }>({
    url: buildManyToManyRelationBasePath(scope, moduleName, ownerId, relationName),
    method: 'get',
    skipErrorMessage: true,
  });
  return Array.isArray(payload?.items) ? payload.items : [];
}

function buildModuleCrudBasePath(scope: string, moduleName: string): string {
  const normalized = String(moduleName || '')
    .replace(/\\\\/g, '/')
    .replace(/^\\/+/, '');
  return \`/\${scope}/\${normalized}\`;
}

function buildManyToManyRelationBasePath(
  scope: string,
  moduleName: string,
  ownerId: unknown,
  relationName: string,
): string {
  return \`\${buildModuleCrudBasePath(scope, moduleName)}/\${ownerId}/relations/\${relationName}\`;
}

function buildManyToManyRelationItemPath(
  scope: string,
  moduleName: string,
  ownerId: unknown,
  relationName: string,
  targetId: string | number,
): string {
  return \`\${buildManyToManyRelationBasePath(scope, moduleName, ownerId, relationName)}/\${targetId}\`;
}

function stringifyRelationValue(value: unknown): string {
  if (value === null || value === undefined || value === '') {
    return '-';
  }
  return String(value);
}

function buildRelationRuntimePath(scope: string, moduleName: string): string {
  return buildModuleCrudBasePath(scope, moduleName);
}

function buildManagedModuleKey(scope: string, moduleName: string): string {
  const normalized = String(moduleName || '')
    .replace(/\\\\/g, '/')
    .replace(/^\\/+/, '')
    .replace(/\\//g, '.');
  return \`\${scope}.\${normalized}\`;
}

async function loadManagedModuleSchema(
  getGeneratedModuleSchema: (module: string) => Promise<ChildModuleSchema>,
  scope: string,
  moduleName: string,
): Promise<ChildModuleSchema> {
  return getGeneratedModuleSchema(buildManagedModuleKey(scope, moduleName));
}

function resolveEditableChildRelation(
  relation: (typeof governanceRelations)[number],
  childSchema: ChildModuleSchema | null,
  currentTableName: string,
): boolean {
  if (!childSchema || relation.type !== 'oneToMany') {
    return false;
  }
  return (
    childSchema.metadata?.tableRole === 'detail' &&
    childSchema.metadata?.primaryTable === currentTableName
  );
}

function buildEditableChildFields(
  relation: (typeof governanceRelations)[number],
  childSchema: ChildModuleSchema,
): ChildModuleField[] {
  return (childSchema.model.fields || []).filter(
    (field) => field.visibleInForm !== false && field.name !== relation.targetField,
  );
}

function buildChildRelationInitialValues(
  relation: (typeof governanceRelations)[number],
  detail: Record<string, unknown>,
  record?: RelationRuntimeRow,
): Record<string, unknown> {
  return {
    ...(record || {}),
    [relation.targetField]: detail[relation.localField],
  };
}

function buildChildRelationContractMap(
  childSchema: ChildModuleSchema,
): Record<string, ChildRelationContract> {
  return (childSchema.relations || []).reduce<Record<string, ChildRelationContract>>((acc, relation) => {
    acc[relation.localField] = {
      field: relation.localField,
      lookupApi: relation.lookupApi,
      targetField: relation.targetField,
      targetLabelField: relation.targetLabelField,
      lookupValueField: relation.lookupValueField,
    };
    return acc;
  }, {});
}

function buildChildRelationOptionKey(schemaName: string, fieldName: string): string {
  return \`\${schemaName}:\${fieldName}\`;
}

function resolveRelationRowValue(
  row: RelationRuntimeRow,
  relation: (typeof governanceRelations)[number],
): string | number | null {
  const rawValue =
    row[relation.lookupValueField || relation.targetField] ?? row.value ?? row.id ?? null;
  if (rawValue === null || rawValue === undefined || rawValue === '') {
    return null;
  }
  return normalizeRelationOptionValue(rawValue);
}

function normalizeChildRelationOptions(
  rows: RelationRuntimeRow[],
  contract: ChildRelationContract,
): RelationEditorOption[] {
  return rows.map((item) => {
    const rawValue =
      item[contract.lookupValueField || contract.targetField || 'value'] ?? item.value ?? item.id ?? '';
    const value = normalizeRelationOptionValue(rawValue);
    const rawLabel =
      item[contract.targetLabelField || contract.targetField || 'label'] ?? item.label ?? item.name;
    return {
      value,
      label: normalizeRelationOptionLabel(rawLabel, value),
    };
  });
}

function normalizeManyToManyBindingOptions(
  rows: RelationRuntimeRow[],
  relation: (typeof governanceRelations)[number],
): RelationEditorOption[] {
  return rows.map((item) => {
    const value = resolveRelationRowValue(item, relation);
    const fallbackValue = value === null ? '' : value;
    const rawLabel =
      item[relation.targetLabelField || relation.targetField || 'label'] ?? item.label ?? item.name;
    return {
      value: fallbackValue,
      label: normalizeRelationOptionLabel(rawLabel, fallbackValue),
    };
  });
}

function mergeManyToManyRelationRows(
  relationRows: RelationRuntimeRow[],
  optionRows: RelationRuntimeRow[],
  relation: (typeof governanceRelations)[number],
): RelationRuntimeRow[] {
  const optionMap = new Map<string, RelationRuntimeRow>();
  optionRows.forEach((row) => {
    const value = resolveRelationRowValue(row, relation);
    if (value === null) {
      return;
    }
    optionMap.set(String(value), row);
  });

  return relationRows.map((row) => {
    const value = resolveRelationRowValue(row, relation);
    if (value === null) {
      return row;
    }
    const optionRow = optionMap.get(String(value));
    return {
      ...row,
      ...(optionRow || {}),
      [relation.lookupValueField || relation.targetField || 'value']: value,
      value,
      id: value,
    };
  });
}

function renderChildFieldInput(
  field: ChildModuleField,
  relationContract: ChildRelationContract | undefined,
  options: RelationEditorOption[],
  loading: boolean,
) {
  switch (field.type) {
    case 'enum':
      return <Select allowClear options={(field.enumOptions || []).map((item) => ({ label: item.label, value: item.value }))} />;
    case 'relation':
      return (
        <Select
          allowClear
          loading={loading}
          disabled={!relationContract?.lookupApi}
          placeholder={field.placeholder || field.label || field.name}
          options={options}
        />
      );
    case 'int':
    case 'float':
      return <InputNumber style={{ width: '100%' }} />;
    case 'bool':
      return <Switch />;
    case 'text':
      return <TextArea autoSize />;
    default:
      return <Input placeholder={field.placeholder || field.label || field.name} />;
  }
}

function buildRelationRuntimeColumns(
  relation: (typeof governanceRelations)[number],
  rows: RelationRuntimeRow[],
  onEdit?: (row: RelationRuntimeRow) => void,
  onUnbind?: (row: RelationRuntimeRow) => void,
  actionTitle?: string,
  actionLabel?: string,
  unbindLabel?: string,
) {
  const labelField = relation.targetLabelField || 'label';
  const columns: Array<{
    title: string;
    dataIndex: string;
    render: (_: unknown, row: RelationRuntimeRow) => React.ReactNode;
  }> = [
    {
      title: labelField,
      dataIndex: labelField,
      render: (_: unknown, row: RelationRuntimeRow) =>
        stringifyRelationValue(row[labelField] ?? row.label ?? row.name ?? row[relation.targetField]),
    },
    {
      title: relation.targetField,
      dataIndex: relation.targetField,
      render: (_: unknown, row: RelationRuntimeRow) =>
        stringifyRelationValue(row[relation.targetField] ?? row.value ?? row.id),
    },
  ];
  const firstRow = rows[0];
  if (firstRow && relation.lookupValueField && relation.lookupValueField !== relation.targetField) {
    columns.push({
      title: relation.lookupValueField,
      dataIndex: relation.lookupValueField,
      render: (_: unknown, row: RelationRuntimeRow) =>
        stringifyRelationValue(row[relation.lookupValueField || relation.targetField]),
    });
  }
  if (onEdit || onUnbind) {
    columns.push({
      title: actionTitle || 'Actions',
      dataIndex: 'actions',
      render: (_: unknown, row: RelationRuntimeRow) => (
        <Space size={8}>
          {onEdit ? (
            <Button
              size="mini"
              onClick={() => {
                onEdit(row);
              }}
            >
              {actionLabel || 'Edit'}
            </Button>
          ) : null}
          {onUnbind ? (
            <Button
              size="mini"
              status="danger"
              onClick={() => {
                onUnbind(row);
              }}
            >
              {unbindLabel || 'Remove'}
            </Button>
          ) : null}
        </Space>
      ),
    });
  }
  return columns;
}
`;
  }

  private buildRelationFieldTargetHint(fieldName: string): string {
    const relation = (this.schema.relations ?? []).find((item) => item.localField === fieldName);
    if (!relation) {
      return this.schema.metadata?.primaryTable || 'lookup pending';
    }
    return `${relation.targetModule} · ${relation.targetField}`;
  }

  /**
   * 获取默认值
   */
  private getDefaultValue(fieldType: string): string {
    switch (fieldType) {
      case 'string':
      case 'text':
      case 'enum':
        return "''";
      case 'int':
      case 'float':
      case 'relation':
        return '0';
      case 'bool':
        return 'false';
      case 'date':
        return "''";
      default:
        return "''";
    }
  }

  private relativeToSrcRoot(): string {
    return '../'.repeat(splitModuleSegments(this.schema.name).length + 2).replace(/\/$/, '');
  }

  private relativeToSystemListPageCss(): string {
    return `${this.relativeToSrcRoot()}/modules/system/list-page.css`;
  }

  private toPascalCase(value: string): string {
    return value
      .split(/[_-]/)
      .filter(Boolean)
      .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
      .join('');
  }
}
