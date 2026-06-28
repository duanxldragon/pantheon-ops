/**
 * 前端列表页模板 (List.tsx)
 *
 * 基于真实 system/user/UserList.tsx 结构 (726行完整实现)
 *
 * 模板变量占位符:
 * - ${toSrcRoot}                      - 相对路径到 src 根目录
 * - ${listImports.join(', ')}          - ArcoDesign 导入列表字符串
 * - ${modelName}                      - PascalCase 模型名
 * - ${governanceConstants}             - 治理常量字符串
 * - ${generateListActionImports()}     - 列表动作导入字符串
 * - ${relativeToSystemListPageCss()}  - CSS 路径字符串
 * - ${generateEmptyQueryFields()}      - 空查询字段初始化字符串
 * - ${hasPageAction:export}            - 是否含导出动作
 * - ${hasPageAction:import}            - 是否含导入动作
 * - ${generateWorkbenchGovernanceBlock()} - 治理区块字符串
 * - ${generateWorkbenchSearchBlock()}  - 搜索区块字符串
 * - ${generateWorkbenchHeaderActions}  - 头部动作区块字符串
 * - ${generateWorkbenchTableColumns}  - 表格列配置字符串
 */

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
    {{hasPageAction:export}}await export{{modelName}}s(query);{{/hasPageAction:export}}
    {{#hasPageAction:export}}{{^}}message.info(t('common.comingSoon'));{{/hasPageAction:export}}
  };

  const handleImport = async (file: File) => {
    {{hasPageAction:import}}await import{{modelName}}s(file);
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
