---
title: 业务生命周期详情页模板 (Business Lifecycle Detail Pattern)
doc_type: Design
layer: business/* (从 pantheon-ops 的 CMDB + Deploy 抽象)
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# 业务生命周期详情页模板 (Business Lifecycle Detail Pattern)

English version: [BUSINESS_LIFECYCLE_DETAIL_PATTERN.en.md](./BUSINESS_LIFECYCLE_DETAIL_PATTERN.en.md)

本文沉淀**业务资源详情页**的标准实现模板，特别针对**有状态生命周期**的资源（如 Deploy 任务的 pending→running→success/failed，CMDB 主机的 pending→online→offline）。

配合 `BUSINESS_RESOURCE_LIST_PATTERN.md` 使用——前者管列表，本文管详情。

不替代 `FRONTEND_PAGE_TEMPLATES.md §5 DetailPage`——后者定义骨架；本文定义业务领域特化（hero stats、生命周期动作、子资源表）。

---

## 1. 何时使用此模板

满足以下任一条件的资源详情：

- ✅ 资源有 status 字段，且 status 有多个值
- ✅ 资源有可执行的状态转换动作（启动、取消、标记完成）
- ✅ 资源有关联的子资源列表（如 Deploy 任务的主机执行明细）
- ✅ 资源有需要展示的元信息块（创建时间、负责人、关联实体）

不适用：

- ❌ 极简资源（只有 1-2 个字段）→ 直接在列表 Modal 编辑
- ❌ 纯只读资源 → 简化为 `Descriptions` 单页
- ❌ 树/图状资源 → 用专用可视化组件

---

## 2. 文件结构

```
frontend/src/modules/business/<module>/<resource>/
├── api.ts
├── <Resource>List.tsx
├── <Resource>Form.tsx
├── <Resource>Detail.tsx       // 本模板
└── locales/
```

---

## 3. 必须 import 的平台组件

```tsx
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Card, Descriptions, Tag, Space, Button, Form, Input, Select, Message, Typography, Popconfirm,
} from '@arco-design/web-react';
import { IconLeft } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import PageContainer  from '../../../../components/patterns/PageContainer';
import PageHeader     from '../../../../components/patterns/PageHeader';
import FormSection    from '../../../../components/patterns/FormSection';
import SubmitBar      from '../../../../components/patterns/SubmitBar';
import { usePermission } from '../../../../hooks/usePermission';
```

如果详情页包含子资源表（如任务的主机明细）：

```tsx
import AppTable from '../../../../components/data-display/AppTable';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
```

---

## 4. 标准 state 结构

```tsx
const { id } = useParams<{ id: string }>();
const navigate = useNavigate();
const { t } = useTranslation();
const { hasPerm } = usePermission();

const [resource, setResource] = useState<<Resource>Row | null>(null);
const [loading, setLoading] = useState(true);
const [error, setError] = useState<unknown>(null);

// 状态转换动作的 in-flight 标志
const [acting, setActing] = useState<string | null>(null);  // null | 'start' | 'cancel' | 'markResult' | ...

// 子资源 modal（如标记结果对话框）
const [selectedChild, setSelectedChild] = useState<ChildRow | null>(null);
const [modalVisible, setModalVisible] = useState(false);
```

---

## 5. loadData

```tsx
const loadData = useCallback(async () => {
  if (!id) return;
  setLoading(true);
  setError(null);
  try {
    const result = await get<Resource>Detail(id);
    setResource(result);
  } catch (err) {
    setError(err);
  } finally {
    setLoading(false);
  }
}, [id]);

useEffect(() => { loadData(); }, [loadData]);
```

约束同列表页：try / catch（保留 error 给 `PageError`） / finally。

---

## 6. 渲染骨架（5 层结构）

```tsx
return (
  <PageContainer>
    <PageHeader
      title={t('business.<module>.<resource>.detail.title')}
      onBack={() => navigate(-1)}
      breadcrumb={[
        { label: t('business.<module>.title'), to: '/business/<module>' },
        { label: t('business.<module>.<resource>.list'), to: '/business/<module>/<resource>' },
        { label: resource?.name ?? id },   // 动态末节点
      ]}
    />

    {loading && <PageLoading />}
    {!loading && error && <PageError onRetry={loadData} />}
    {!loading && !error && !resource && <PageEmpty variant="notFound" onBack={() => navigate(-1)} />}

    {resource && (
      <>
        {/* §6.1 Hero stats 区 */}
        <Card>
          <HeroStats stats={heroStats} actions={renderActions()} />
        </Card>

        {/* §6.2 元信息块（Descriptions） */}
        <Card title={t('business.<module>.<resource>.section.summary')}>
          <Descriptions data={summaryData} column={2} />
        </Card>

        {/* §6.3 状态流可视化（仅有状态机的资源） */}
        {hasLifecycle && (
          <Card title={t('business.<module>.<resource>.section.lifecycle')}>
            <LifecycleVisualizer current={resource.status} history={resource.statusHistory} />
          </Card>
        )}

        {/* §6.4 子资源表（如有） */}
        {hasChildren && (
          <Card title={t('business.<module>.<resource>.section.children')}>
            <ChildrenTable rows={resource.children} onAction={handleChildAction} />
          </Card>
        )}

        {/* §6.5 关联实体 / 审计（可选） */}
      </>
    )}

    <AppModal visible={modalVisible} title={...} onCancel={closeModal}>
      {/* 子动作表单 */}
    </AppModal>
  </PageContainer>
);
```

---

## 7. Hero Stats 设计

`heroStats` 是详情页头部的关键指标条，用 `useMemo` 推导：

```tsx
const heroStats = useMemo(
  () => resource ? [
    {
      key: 'status',
      label: t('business.<module>.<resource>.status'),
      value: t(`business.<module>.<resource>.status.${resource.status}`),
      hint: t('business.<module>.<resource>.hero.statusHint'),
      tone: statusToneMap[resource.status],  // 用 base 的状态色 token
    },
    {
      key: 'progress',
      label: t('business.<module>.<resource>.progress'),
      value: `${resource.successCount}/${resource.totalCount}`,
    },
    // 4-6 个关键指标
  ] : [],
  [resource, t]
);
```

约束：
- Hero stats **最多 6 个**，按重要性排序
- 每个 stat 必须有 `tone` 或自然色，**禁止**自定义 hex
- 状态字段用 Tag 或类 Tag 视觉（颜色 + 文案），不裸文字

---

## 8. 状态转换动作

```tsx
const renderActions = () => {
  if (!resource) return null;

  const canStart  = hasPerm('business:<module>:<resource>:start')  && resource.status === 'draft';
  const canCancel = hasPerm('business:<module>:<resource>:cancel') && ['pending', 'running'].includes(resource.status);

  return (
    <Space>
      {canStart && (
        <Popconfirm content={t('business.<module>.<resource>.startConfirm')} onOk={() => handleAction('start')}>
          <Button type="primary" loading={acting === 'start'}>{t('common.start')}</Button>
        </Popconfirm>
      )}
      {canCancel && (
        <Popconfirm content={t('business.<module>.<resource>.cancelConfirm')} onOk={() => handleAction('cancel')}>
          <Button status="danger" loading={acting === 'cancel'}>{t('common.cancel')}</Button>
        </Popconfirm>
      )}
    </Space>
  );
};

const handleAction = async (kind: 'start' | 'cancel' | string) => {
  setActing(kind);
  try {
    await action<Resource>(resource!.id, kind);
    Message.success(t('common.actionSuccess'));
    loadData();   // 重新拉详情以同步状态
  } catch (err) {
    // interceptor handle
  } finally {
    setActing(null);
  }
};
```

约束：
- 每个动作必须**同时满足**权限和当前状态条件
- 高危动作必须 `Popconfirm`
- 动作期间按钮 `loading` + 其他动作不应该并发触发
- 完成后必须 `loadData()` 同步最新状态

---

## 9. 子资源表与子动作

如详情页包含子资源（如 Deploy 任务的主机明细），表格列：

```tsx
const childrenColumns: ColumnProps<ChildRow>[] = [
  { dataIndex: 'name',   title: t('child.name') },
  { dataIndex: 'status', title: t('child.status'),
    render: (status) => <Tag color={statusColorMap[status]}>{t('...')}</Tag> },
  // 业务列
  {
    title: t('common.actions'),
    fixed: 'right',
    render: (_, row) => (
      <Space>
        <Button onClick={() => openChildOutput(row)}>{t('common.viewOutput')}</Button>
        {canMark && row.status === 'pending' && (
          <Button onClick={() => openMarkModal(row, 'success')}>{t('child.markSuccess')}</Button>
        )}
        {canMark && row.status === 'pending' && (
          <Button onClick={() => openMarkModal(row, 'failed')}>{t('child.markFailed')}</Button>
        )}
      </Space>
    ),
  },
];
```

子动作的弹窗（如「标记失败」要求填错误描述）走 `AppModal` + 表单。

子动作完成后必须 `loadData()` 重拉父详情（父状态可能因子动作变化）。

---

## 10. 元信息块（Descriptions）

```tsx
const summaryData = useMemo(() => resource ? [
  { label: t('common.id'),         value: resource.id },
  { label: t('common.name'),       value: resource.name },
  { label: t('common.createdBy'),  value: resource.createdBy },
  { label: t('common.createdAt'),  value: formatDate(resource.createdAt) },
  { label: t('common.updatedAt'),  value: formatDate(resource.updatedAt) },
  // 业务字段
] : [], [resource, t]);
```

约束：
- 时间统一用 `formatDate` 工具，**不**手写格式
- 关联实体（如 `package_id` → 软件组件名）显示为可点击链接

---

## 11. 移动端断点行为

参考 `MOBILE_RESPONSIVE_BREAKPOINTS.md` §5：

- `xl`/`2xl`：5 个 section 多列网格
- `md`/`lg`：双列网格
- `sm`/`xs`：单列堆叠；子资源表切换为卡片视图

Hero stats 在 `sm` 下变成竖向堆叠（每个 stat 一行）。

---

## 12. 页面状态变体

| 状态 | 处理 |
|---|---|
| `loading` | `<PageLoading />`，整页 |
| `not-found` | `<PageEmpty variant="notFound" onBack={() => navigate(-1)} />` |
| `error-server` | `<PageError onRetry={loadData} />` |
| `forbidden` | 由 `RoutePermissionGuard` 拦截，不进本组件 |
| **部分 section 加载失败** | 该 section 内嵌 `<PageError>`，**不**让整页失败 |

详细见 `EMPTY_LOADING_ERROR_STATES.md` §4 DetailPage。

---

## 13. 实例参考

- `pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx`（含子资源表 + 标记结果）
- `pantheon-ops/frontend/src/modules/business/cmdb/host/CmdbHostDetail.tsx`（hero stats + 元信息 + SSH 采集动作）

---

## 14. 验收

- [ ] 5 层骨架（PageHeader + hero + summary + lifecycle + children）按需出现
- [ ] hero stats 不超过 6 个，状态色用 token
- [ ] 所有状态转换动作有 Popconfirm
- [ ] 动作完成后 `loadData()` 同步
- [ ] 子动作完成后**也** `loadData()` 同步父状态
- [ ] 单个 section 失败不让整页 fallback
- [ ] 面包屑动态末节点用资源名
- [ ] 移动端 `sm/md` 测试通过
- [ ] 暗色模式样式正确

---

## 15. 关联

- `BUSINESS_RESOURCE_LIST_PATTERN.md` 配套列表模板
- `FRONTEND_PAGE_TEMPLATES.md` §5 DetailPage
- `EMPTY_LOADING_ERROR_STATES.md` §4 DetailPage 状态变体
- `THEME_TOKENS_REFERENCE.md` §4 状态色 token
- `ACCESSIBILITY.md`
- `MOBILE_RESPONSIVE_BREAKPOINTS.md` §5 DetailPage 行为
- `NAVIGATION_IA_STRATEGY.md` §8 面包屑动态末节点
