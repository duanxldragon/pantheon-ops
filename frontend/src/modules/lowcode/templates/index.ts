/**
 * 前端代码生成器模板导出
 *
 * 所有前端模板字符串集中在此处导出。
 * 模板内容存储在对应的 .txt 文件中，便于单独阅读和编辑。
 *
 * 加载策略: 在 Node.js 环境或 bundler 配置下,
 * 可以通过 fs/readFile 或 raw-loader 加载 .txt 文件。
 *
 * 当前策略: 模板以 TypeScript 字符串常量形式内联导出,
 * 配合 vscode 语法高亮 (安装 "lit-html" 或 " Tagged Template Literal" 插件)
 * 即可获得良好的编辑体验。
 */

// ─── 模板 1: index.ts (模块注册) ───────────────────────────────────────────
export const INDEX_TEMPLATE = `import { defineModule } from '{{toSrcRoot}}/core/router/types';

export const {{modelName}}Module = defineModule({
  name: '{{name}}',
  scope: '{{scope}}',
  routes: {{generateNavigation}}
    ? [
  {
    path: '{{routePath}}',
    routeName: '{{routeName}}',
    titleKey: '{{titleKey}}',
    icon: 'apps',
    pagePermission: '{{permissionPrefix}}:list',
    componentKey: '{{listComponentKey}}',
  },
  {{generateDetailRoute}}
    ? {
  path: '{{routePath}}/:id',
  routeName: '{{routeName}}-detail',
  titleKey: '{{titleKey}}',
  pagePermission: '{{permissionPrefix}}:view',
  activeMenu: '{{buildPageRoutePath}}',
  componentKey: '{{detailComponentKey}}',
},
    : ''
}
  ]
    : '[]'
},
  menus: {{generateNavigation}}
    ? [
  { path: '{{buildPageRoutePath}}', titleKey: '{{titleKey}}', icon: 'apps', routeName: '{{routeName}}', module: '{{moduleNamespace}}' },
]
    : '[]'
},{{dashboardWidgets}}
  permissions: {{permissionItems}}
    ? [
  {{permissionItems}}
]
    : '[]'
},
  i18nNamespaces: ['{{moduleNamespace}}'],
});
`;

// ─── 模板 2: api.ts (API 层) ───────────────────────────────────────────────
export const API_TEMPLATE = `import { apiRequest } from '{{toSrcRoot}}/api/request';
{{extraImports}}

export interface {{modelName}}ListRow {
  id: number;
{{generateTSFields:list}}
  createdAt: string;
}

export interface {{modelName}}Detail {
  id: number;
{{generateTSFields:detail}}
  createdAt: string;
  updatedAt: string;
}

export interface {{modelName}}ListQuery {
{{generateTSQueryFields}}
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface {{modelName}}ListPageResp {
  items: {{modelName}}ListRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface {{modelName}}CreatePayload {
{{generateTSFields:create}}
}

export interface {{modelName}}UpdatePayload {
{{generateTSFields:update}}
}

{{relationLookupApi}}
{{relationBindingApi}}

export function get{{modelName}}List(params?: {{modelName}}ListQuery) {
  return apiRequest<{{modelName}}ListPageResp>({
    url: '{{routePath}}/list',
    method: 'get',
    params,
  });
}

export function get{{modelName}}Detail(id: number) {
  return apiRequest<{{modelName}}Detail>({
    url: \`{{routePath}}/\${id}\`,
    method: 'get',
  });
}

{{mutatingApi}}
{{extraApi}}
`;

// ─── 模板 3: List.tsx (列表页) ─────────────────────────────────────────────
// 完整列表页模板 (约 186 行)
export const LIST_TEMPLATE = `import React, { useCallback, useEffect, useState } from 'react';
import { {{listImports}} } from '@arco-design/web-react';
import { IconDelete, IconDownload, IconEye, IconPlus, IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { message } from '{{toSrcRoot}}/components/feedback/message';
import {
  {{generateListActionImports}}
  get{{modelName}}List,
  type {{modelName}}ListQuery,
  type {{modelName}}ListRow,
} from './api';
import {
  AppTable,
  FilterPanel,
  GovernanceSummaryBar,
  ImportCsvButton,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  SystemRowActions,
  TableBatchActionBar,
  TABLE_ACTION_COLUMN_WIDTH,
  buildStandardPagination,
} from '{{toSrcRoot}}/components';
import '{{relativeToSystemListPageCss}}';

{{governanceConstants}}

const FormItem = Form.Item;
const { Row, Col } = Grid;

const emptyQuery: {{modelName}}ListQuery = {
{{generateEmptyQueryFields}}
  page: 1,
  pageSize: 20,
};

const {{modelName}}List: React.FC = () => {
  const navigate = useNavigate();
  const [data, setData] = useState<{{modelName}}ListRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [query, setQuery] = useState<{{modelName}}ListQuery>(emptyQuery);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [submitting, setSubmitting] = useState(false);
  const [queryForm] = Form.useForm<{{modelName}}ListQuery>();
  const { t } = useTranslation();

  const loadData = useCallback(async (nextQuery: {{modelName}}ListQuery = query) => {
    setLoading(true);
    setError(null);
    try {
      const result = await get{{modelName}}List(nextQuery);
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
    {{#hasPageAction:export}}await export{{modelName}}s(query);{{/hasPageAction:export}}
    {{#hasPageAction:export}}{{^}}message.info(t('common.comingSoon'));{{/hasPageAction:export}}
  };

  const handleImport = async (file: File) => {
    {{#hasPageAction:import}}await import{{modelName}}s(file);
    message.success(t('common.importSuccess'));{{/hasPageAction:import}}
    {{#hasPageAction:import}}{{^}}void file;{{/hasPageAction:import}}
    await loadData(query);
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      await Promise.all(selectedRowKeys.map((rowKey) => delete{{modelName}}(Number(rowKey))));
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
    return <PageRequestError error={error} onRetry={() => { void loadData(); }} />;
  }

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
{{generateWorkbenchGovernanceBlock}}
{{generateWorkbenchSearchBlock}}
{{generateWorkbenchHeaderActions}}
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {error && data.length === 0 ? (
            <PageRequestError
              error={error}
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
              {{rowSelection}}
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
{{generateWorkbenchTableColumns}}
              ]}
            />
          ) : null}
        </Card>
      </Space>
    </PageContainer>
  );
};

export default {{modelName}}List;
`;

// ─── 模板 4: Form.tsx (表单组件) ───────────────────────────────────────────
export const FORM_TEMPLATE = `import React, { {{formHookImports}} } from 'react';
import { {{formImports}} } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import {
  {{relationLookupImports}}type {{modelName}}CreatePayload,
  type {{modelName}}UpdatePayload,
} from './api';
import { SubmitBar } from '{{toSrcRoot}}/components';

{{relationLookupContracts}}

const FormItem = Form.Item;

export interface {{modelName}}FormValues extends Partial<{{modelName}}CreatePayload>, Partial<{{modelName}}UpdatePayload>> {}

export interface {{modelName}}FormProps {
  mode: 'create' | 'update';
  initialValues?: {{modelName}}FormValues;
  submitting?: boolean;
  onSubmit: (values: {{modelName}}FormValues) => void | Promise<void>;
  onCancel?: () => void;
}

const {{modelName}}Form: React.FC<{{modelName}}FormProps> = ({
  mode,
  initialValues,
  submitting = false,
  onSubmit,
  onCancel,
}) => {
  const [form] = Form.useForm<{{modelName}}FormValues>();
{{relationStateBlock}}
  const { t } = useTranslation();

  useEffect(() => {
    form.setFieldsValue(initialValues || {});
  }, [form, initialValues]);

{{relationLookupLoaderEffect}}

  return (
    <Form form={form} layout="vertical" onSubmit={onSubmit}>
{{generateFormItems}}
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

export default {{modelName}}Form;
`;

// ─── 模板 5: Detail.tsx (详情页) ───────────────────────────────────────────
// ~1179 行详情页模板，通过 Vite ?raw 导入
import DETAIL_TEMPLATE from './detailTemplate_clean.txt?raw';

export { DETAIL_TEMPLATE };

