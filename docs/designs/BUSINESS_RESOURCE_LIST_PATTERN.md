---
title: 业务资源列表页模板 (Business Resource List Pattern)
doc_type: Design
layer: business/* (从 pantheon-ops 的 CMDB + Deploy 抽象)
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# 业务资源列表页模板 (Business Resource List Pattern)

English version: [BUSINESS_RESOURCE_LIST_PATTERN.en.md](./BUSINESS_RESOURCE_LIST_PATTERN.en.md)

本文沉淀**业务资源列表页**的标准实现模板。所有 `business/<module>/<resource>` 类资源列表（CMDB 主机、Deploy 任务、Deploy 软件包、CMDB 分组、CMDB 标签规范、未来 CRM 客户、WMS 物料等）都应遵循此模板。

不替代 `FRONTEND_PAGE_TEMPLATES.md §3 ListPage`——后者定义骨架；本文定义业务领域特化。

---

## 1. 何时使用此模板

当业务资源满足以下任一条件：

- ✅ 有 list / create / update / delete 四件套 API
- ✅ 有筛选和分页需求
- ✅ 有状态字段（如 enabled/disabled、online/offline、pending/running/success）
- ✅ 需要按权限渲染操作按钮

不适用：

- ❌ 树形资源（用 `FRONTEND_PAGE_TEMPLATES.md §4 TreePage` 或 `BUSINESS_LIFECYCLE_DETAIL_PATTERN.md` 的左树右表变体）
- ❌ 看板类（用 Dashboard 模板）
- ❌ 极简「单类一行」的资源（用 ConfigPage 模板）

---

## 2. 文件结构

```
frontend/src/modules/business/<module>/<resource>/
├── api.ts                  // request 函数 + TypeScript 类型
├── <Resource>List.tsx       // 列表页（本模板）
├── <Resource>Form.tsx       // 表单组件（Modal 内）
├── <Resource>Detail.tsx     // 详情页（如有）
└── locales/
    ├── zh-CN.json
    └── en-US.json
```

业务模块根目录：

```
frontend/src/modules/business/<module>/
├── <module>.css           // 模块级样式（如有）
└── <resource>/...
```

---

## 3. 必须 import 的平台组件

```tsx
import { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Card, Button, Tag, Space, Popconfirm, Form, Input, Select, Message, Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import PageContainer    from '../../../../components/patterns/PageContainer';
import PageHeader       from '../../../../components/patterns/PageHeader';
import FilterPanel      from '../../../../components/patterns/FilterPanel';
import AppTable         from '../../../../components/data-display/AppTable';
import ListHeaderActions from '../../../../components/patterns/ListHeaderActions';
import { usePermission } from '../../../../hooks/usePermission';
```

**禁止**：
- 直接 import `react-router-dom` 之外的路由库
- 自写表格组件（必须用 `AppTable`）
- 自写 Modal（必须用 `AppModal`）
- 重写状态色映射（用 base 的 status 主题 token，见 §6）

---

## 4. 标准 state 结构

```tsx
const [data, setData] = useState<<Resource>Row[]>([]);
const [total, setTotal] = useState(0);
const [loading, setLoading] = useState(false);
const [error, setError] = useState<unknown>(null);
const [query, setQuery] = useState<<Resource>ListQuery>({ page: 1, pageSize: 10 });

// 筛选字段（按业务需要展开）
const [keyword, setKeyword] = useState('');
const [filterStatus, setFilterStatus] = useState<string>('');
// ... 其他业务专属筛选

// 表单 Modal 相关
const [visible, setVisible] = useState(false);
const [editing, setEditing] = useState<<Resource>Row | null>(null);
const [submitting, setSubmitting] = useState(false);
```

---

## 5. 权限标志

```tsx
const { hasPerm } = usePermission();
const canCreate = hasPerm('business:<module>:<resource>:create');
const canUpdate = hasPerm('business:<module>:<resource>:update');
const canDelete = hasPerm('business:<module>:<resource>:delete');
// 业务特殊权限：
const canCollect = hasPerm('business:cmdb:host:collect'); // 示例
```

**约束**：权限点命名必须遵循 `business:<module>:<resource>:<action>` 四段式（详见 `PERMISSION_MODEL.md`），禁止新增 `biz:*`。

---

## 6. 状态色映射

业务级状态色 Tag 使用 base 的 `THEME_TOKENS_REFERENCE.md` §4 状态色 token，**绝不**自定义颜色。映射示例：

```tsx
// 业务无关的语义状态
const statusColorMap: Record<string, string> = {
  pending:    'gray',      // info-bg 中性
  running:    'arcoblue',  // info
  success:    'green',     // success
  failed:     'red',       // error
  online:     'green',     // success
  offline:    'red',       // error
  maintenance: 'orange',   // warning
  canceled:   'gray',      // 中性
  skipped:    'gray',
  enabled:    'green',
  disabled:   'gray',
};
```

文案走 i18n key：`Tag color={statusColorMap[row.status]}>{t('business.<module>.<resource>.status.' + row.status)}</Tag>`。

---

## 7. loadData 闭包

```tsx
const loadData = useCallback(
  async (nextQuery = query) => {
    setLoading(true);
    setError(null);
    try {
      const result = await get<Resource>List(nextQuery);
      setData(result.items);
      setTotal(result.total);
    } catch (err) {
      setError(err);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  },
  [query, t]
);

useEffect(() => { loadData(); }, [loadData]);
```

**约束**：
- 错误必须 `setError` 而非吞掉（让 `PageError` 渲染）
- 必须 `setLoading(true)` 起，`finally` 中 `setLoading(false)`
- Message.error 文案走 `common.loadFailed` 或更具体的 i18n key

---

## 8. 渲染骨架

```tsx
return (
  <PageContainer>
    <PageHeader title={t('business.<module>.<resource>.title')} />

    <FilterPanel onSearch={() => loadData({ ...query, page: 1 })}>
      <Input placeholder={t('common.search')} value={keyword} onChange={setKeyword} />
      <Select placeholder={...} value={filterStatus} onChange={setFilterStatus}>...</Select>
      {/* 其他业务筛选 */}
    </FilterPanel>

    <ListHeaderActions>
      {canCreate && (
        <Button type="primary" icon={<IconPlus />} onClick={openCreateModal}>
          {t('common.create')}
        </Button>
      )}
      {/* 其他全局动作 */}
    </ListHeaderActions>

    {/* 状态分支 */}
    {loading && <PageLoading />}
    {!loading && error && <PageError onRetry={loadData} />}
    {!loading && !error && data.length === 0 && (
      keyword || filterStatus
        ? <PageEmpty variant="filtered" onClear={clearFilters} />
        : <PageEmpty variant="initial" onCreate={canCreate ? openCreateModal : undefined} />
    )}
    {!loading && !error && data.length > 0 && (
      <AppTable
        columns={columns}
        dataSource={data}
        pagination={{
          current: query.page,
          pageSize: query.pageSize,
          total,
          onChange: (page, pageSize) => {
            setQuery({ ...query, page, pageSize });
            loadData({ ...query, page, pageSize });
          },
        }}
      />
    )}

    <AppModal
      visible={visible}
      title={editing ? t('common.edit') : t('common.create')}
      onCancel={closeModal}
      footer={null}
    >
      <<Resource>Form
        initial={editing}
        submitting={submitting}
        onSubmit={handleSubmit}
        onCancel={closeModal}
      />
    </AppModal>
  </PageContainer>
);
```

---

## 9. 表格列定义约束

```tsx
const columns: ColumnProps<<Resource>Row>[] = [
  { dataIndex: 'name',   title: t('business.<module>.<resource>.name'), priority: 1 },
  { dataIndex: 'status', title: t('business.<module>.<resource>.status'),
    priority: 2,
    render: (status) => <Tag color={statusColorMap[status]}>{t('...')}</Tag> },
  // 业务专属列
  { dataIndex: 'updatedAt', title: t('common.updatedAt'), priority: 4 },
  {
    title: t('common.actions'),
    priority: 1,                   // 操作列始终保留
    fixed: 'right',
    render: (_, row) => (
      <Space>
        {canUpdate && <Button icon={<IconEdit />} onClick={() => openEditModal(row)} />}
        {canDelete && (
          <Popconfirm content={t('common.deleteConfirm')} onOk={() => handleDelete(row.id)}>
            <Button icon={<IconDelete />} status="danger" />
          </Popconfirm>
        )}
      </Space>
    ),
  },
];
```

**约束**：
- 每列必须有 `priority` 字段（1-5，1=必显），用于 `MOBILE_RESPONSIVE_BREAKPOINTS.md` §4 的断点退化
- 操作列必须 `fixed: 'right'`
- 删除必须 `Popconfirm` 二次确认

---

## 10. 创建 / 编辑 / 删除 handler

```tsx
const openCreateModal = () => { setEditing(null); setVisible(true); };
const openEditModal = (row: <Resource>Row) => { setEditing(row); setVisible(true); };
const closeModal = () => { setVisible(false); setEditing(null); };

const handleSubmit = async (payload: Create<Resource>Payload) => {
  setSubmitting(true);
  try {
    if (editing) {
      await update<Resource>(editing.id, payload);
      Message.success(t('common.updateSuccess'));
    } else {
      await create<Resource>(payload);
      Message.success(t('common.createSuccess'));
    }
    closeModal();
    loadData();
  } catch (err) {
    // 错误已被 request interceptor 处理为 Message.error
  } finally {
    setSubmitting(false);
  }
};

const handleDelete = async (id: number) => {
  try {
    await delete<Resource>(id);
    Message.success(t('common.deleteSuccess'));
    loadData();
  } catch (err) {
    // interceptor handle
  }
};
```

---

## 11. 页面状态变体清单

按 `EMPTY_LOADING_ERROR_STATES.md` §3 ListPage 状态变体表，本模板必须覆盖：

| 状态 | 触发 | 组件 |
|---|---|---|
| `loading` | 首次或刷新时 | `<PageLoading />` |
| `empty-initial` | 无筛选，结果为空 | `<PageEmpty variant="initial" onCreate={...} />` |
| `empty-filtered` | 有筛选，结果为空 | `<PageEmpty variant="filtered" onClear={...} />` |
| `error-network` / `error-server` | catch | `<PageError onRetry={loadData} />` |
| `forbidden` | 路由级拦截，不进入本组件 | 由 `RoutePermissionGuard` 处理 |

---

## 12. 可达性

- 表格行内按钮必须有 `aria-label`（icon-only 按钮）
- Popconfirm 必须包裹有 `aria-label` 的 trigger 按钮
- 排序、筛选变化时，由 `AppTable` 自动公告（`role="status"`）
- 详见 `ACCESSIBILITY.md` §3

---

## 13. 实例参考

- `pantheon-ops/frontend/src/modules/business/cmdb/host/CmdbHostList.tsx`
- `pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskList.tsx`
- `pantheon-ops/frontend/src/modules/business/cmdb/group/CmdbGroupList.tsx`（左树右表变体）
- `pantheon-ops/frontend/src/modules/business/cmdb/label/CmdbLabelSchemaList.tsx`

---

## 14. 验收

新业务资源列表页合入前，验证：

- [ ] 6 个核心 import 全部存在（`PageContainer`、`PageHeader`、`FilterPanel`、`AppTable`、`ListHeaderActions`、`AppModal`）
- [ ] 权限 hook `usePermission` 正确调用，权限点命名符合 canonical 四段式
- [ ] `statusColorMap` 与平台主题 token 对齐
- [ ] `loadData` 用 useCallback + 标准 try/catch/finally 三段式
- [ ] 5 种状态变体（loading/empty-initial/empty-filtered/error/正常）全部覆盖
- [ ] 表格每列有 priority 字段
- [ ] 删除有 Popconfirm
- [ ] 文案 100% 走 i18n key
- [ ] 移动端断点测试（sm/md）通过
- [ ] 暗色模式通过 `THEME_TOKENS_REFERENCE.md` 主题切换无样式破缺

---

## 15. 关联

- `FRONTEND_PAGE_TEMPLATES.md` §3 ListPage 通用骨架
- `FRONTEND_COMPONENT_PLAN.md` §4.2 列表页类组件
- `EMPTY_LOADING_ERROR_STATES.md` §3 ListPage 状态变体
- `THEME_TOKENS_REFERENCE.md` §4 状态色 token
- `ACCESSIBILITY.md`
- `MOBILE_RESPONSIVE_BREAKPOINTS.md` §4 ListPage 行为
- `BUSINESS_LIFECYCLE_DETAIL_PATTERN.md` 详情页配套模板
- `PERMISSION_MODEL.md` 权限点命名约定
