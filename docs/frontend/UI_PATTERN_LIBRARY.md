# UI 模式库

## 1. 列表页模式

### 1.1 标准列表页结构

```tsx
import { SearchToolbar } from '@/components/patterns/SearchToolbar';

function ListPage() {
  return (
    <div className="page-container">
      {/* 页面头部 */}
      <div className="page-header">
        <div className="page-header__meta">
          <h1 className="page-header__title">用户管理</h1>
          <p className="page-header__description">管理系统用户和权限</p>
        </div>
        <div className="page-header__actions">
          <Button type="primary" icon={<IconPlus />}>
            新增用户
          </Button>
        </div>
      </div>

      {/* 筛选工具栏 */}
      <SearchToolbar
        keyword={keyword}
        onKeywordChange={setKeyword}
        filters={filters}
        onFiltersChange={setFilters}
        filterConfig={filterConfig}
        onReset={handleReset}
      />

      {/* 数据表格 */}
      <Card bordered={false}>
        <Table
          loading={loading}
          data={data}
          columns={columns}
          pagination={pagination}
          onChange={handleTableChange}
        />
      </Card>
    </div>
  );
}
```

### 1.2 筛选工具栏配置

```tsx
const filterConfig = [
  {
    key: 'status',
    label: '状态',
    type: 'select',
    options: [
      { label: '启用', value: 'active' },
      { label: '禁用', value: 'inactive' },
    ],
    inline: true, // 显示在行内，变更即查询
  },
  {
    key: 'role',
    label: '角色',
    type: 'select',
    options: roleOptions,
    inline: false, // 收进"筛选"弹层
  },
  {
    key: 'dateRange',
    label: '创建时间',
    type: 'dateRange',
    inline: false,
  },
];
```

### 1.3 表格操作列

```tsx
const columns = [
  // ... 数据列
  {
    title: '操作',
    dataIndex: 'actions',
    width: 200,
    fixed: 'right' as const,
    render: (_: any, record: User) => (
      <Space>
        <Button
          type="text"
          size="small"
          onClick={() => handleEdit(record)}
        >
          编辑
        </Button>
        <Button
          type="text"
          size="small"
          status="danger"
          onClick={() => handleDelete(record)}
        >
          删除
        </Button>
      </Space>
    ),
  },
];
```

## 2. 表单页模式

### 2.1 标准表单布局

```tsx
function FormPage() {
  return (
    <div className="page-container">
      <div className="page-header">
        <h1 className="page-header__title">用户信息</h1>
      </div>

      <Card bordered={false}>
        <Form
          form={form}
          layout="vertical"
          onSubmit={handleSubmit}
          style={{ maxWidth: 640 }}
        >
          <Form.Item
            label="用户名"
            field="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>

          <Form.Item
            label="邮箱"
            field="email"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          {/* 表单底部操作栏 */}
          <div className="form-actions">
            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>
                保存
              </Button>
              <Button onClick={handleCancel}>
                取消
              </Button>
            </Space>
          </div>
        </Form>
      </Card>
    </div>
  );
}
```

```css
.form-actions {
  margin-top: var(--space-2xl);
  padding-top: var(--space-lg);
  border-top: 1px solid var(--panel-border);
}
```

### 2.2 表单验证模式

```tsx
const validateRules = {
  username: [
    { required: true, message: '请输入用户名' },
    { minLength: 3, message: '用户名至少 3 个字符' },
    { maxLength: 20, message: '用户名最多 20 个字符' },
    {
      validator: (value, cb) => {
        if (!/^[a-zA-Z0-9_]+$/.test(value)) {
          cb('用户名只能包含字母、数字和下划线');
        }
        cb();
      },
    },
  ],
  password: [
    { required: true, message: '请输入密码' },
    { minLength: 8, message: '密码至少 8 个字符' },
    {
      validator: (value, cb) => {
        if (!/(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/.test(value)) {
          cb('密码必须包含大小写字母和数字');
        }
        cb();
      },
    },
  ],
};
```

## 3. 对话框模式

### 3.1 表单对话框

```tsx
function FormDialog({ visible, onClose, onSubmit, initialData }: FormDialogProps) {
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (visible) {
      form.resetFields();
      if (initialData) {
        form.setFieldsValue(initialData);
      }
    }
  }, [visible, initialData]);

  const handleSubmit = async () => {
    try {
      const values = await form.validate();
      setSubmitting(true);
      await onSubmit(values);
      onClose();
    } catch (error) {
      // 表单验证失败
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      visible={visible}
      onCancel={onClose}
      onOk={handleSubmit}
      confirmLoading={submitting}
      title={initialData ? '编辑用户' : '新增用户'}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          label="用户名"
          field="username"
          rules={[{ required: true }]}
        >
          <Input placeholder="请输入用户名" />
        </Form.Item>
        {/* 其他表单项 */}
      </Form>
    </Modal>
  );
}
```

### 3.2 确认对话框

```tsx
function handleDelete(record: User) {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除用户 "${record.username}" 吗？此操作不可恢复。`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: {
      status: 'danger',
    },
    onOk: async () => {
      try {
        await deleteUser(record.id);
        Message.success('删除成功');
        reload();
      } catch (error) {
        Message.error('删除失败');
      }
    },
  });
}
```

## 4. 数据展示模式

### 4.1 统计卡片

```tsx
function StatCard({ label, value, trend, icon }: StatCardProps) {
  return (
    <div className="stat-card">
      <div className="stat-card__header">
        <span className="stat-card__label">{label}</span>
        {icon && <span className="stat-card__icon">{icon}</span>}
      </div>
      <div className="stat-card__value">{value}</div>
      {trend && (
        <div className={`stat-card__trend stat-card__trend--${trend.direction}`}>
          <IconArrowUp />
          <span>{trend.value}</span>
        </div>
      )}
    </div>
  );
}
```

```css
.stat-card {
  background-color: var(--container-display-elevated);
  border: 1px solid var(--container-display-border);
  border-radius: var(--radius-md);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.stat-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-card__label {
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-card__value {
  color: var(--text-primary);
  font-size: 28px;
  font-weight: 600;
  font-feature-settings: 'tnum';
}

.stat-card__trend {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2xs);
  font-size: 13px;
  font-weight: 500;
}

.stat-card__trend--up {
  color: rgb(var(--green-6));
}

.stat-card__trend--down {
  color: rgb(var(--red-6));
}
```

### 4.2 描述列表

```tsx
function ProfileInfo({ user }: { user: User }) {
  return (
    <Card title="基本信息" bordered={false}>
      <Descriptions
        column={2}
        data={[
          { label: '用户名', value: user.username },
          { label: '姓名', value: user.name },
          { label: '邮箱', value: user.email },
          { label: '手机号', value: user.phone },
          { label: '角色', value: user.role },
          { label: '状态', value: <Badge status={user.status} text={user.statusText} /> },
          { label: '创建时间', value: formatDate(user.createdAt) },
          { label: '最后登录', value: formatDate(user.lastLoginAt) },
        ]}
      />
    </Card>
  );
}
```

### 4.3 空状态

```tsx
function EmptyState({ description, action }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state__icon">
        <IconEmpty />
      </div>
      <div className="empty-state__description">
        {description || '暂无数据'}
      </div>
      {action && (
        <div className="empty-state__action">
          {action}
        </div>
      )}
    </div>
  );
}
```

```css
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-3xl);
  color: var(--text-tertiary);
}

.empty-state__icon {
  font-size: 48px;
  margin-bottom: var(--space-lg);
  opacity: 0.4;
}

.empty-state__description {
  font-size: 14px;
  margin-bottom: var(--space-lg);
}
```

## 5. 加载状态模式

### 5.1 骨架屏

```tsx
function ListSkeleton() {
  return (
    <Card bordered={false}>
      <Skeleton
        loading={true}
        animation
        text={{ rows: 5, width: ['100%', '100%', '100%', '100%', '100%'] }}
      />
    </Card>
  );
}
```

### 5.2 局部加载

```tsx
function PartialLoading({ loading, children }: { loading: boolean; children: React.ReactNode }) {
  return (
    <Spin loading={loading} style={{ display: 'block' }}>
      {children}
    </Spin>
  );
}
```

## 6. 错误状态模式

### 6.1 表单错误

```tsx
<Form.Item
  label="邮箱"
  field="email"
  validateStatus={emailError ? 'error' : undefined}
  help={emailError}
>
  <Input placeholder="请输入邮箱" />
</Form.Item>
```

### 6.2 页面错误

```tsx
function ErrorState({ error, onRetry }: ErrorStateProps) {
  return (
    <div className="error-state">
      <div className="error-state__icon">
        <IconExclamationCircle />
      </div>
      <div className="error-state__title">加载失败</div>
      <div className="error-state__description">
        {error?.message || '网络异常，请稍后重试'}
      </div>
      {onRetry && (
        <Button onClick={onRetry} icon={<IconRefresh />}>
          重新加载
        </Button>
      )}
    </div>
  );
}
```

```css
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-3xl);
}

.error-state__icon {
  font-size: 48px;
  color: rgb(var(--red-6));
  margin-bottom: var(--space-lg);
}

.error-state__title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: var(--space-sm);
}

.error-state__description {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: var(--space-lg);
}
```

## 7. 响应式模式

### 7.1 移动端适配

```css
.toolbar {
  display: flex;
  gap: var(--space-md);
  
  @media (max-width: 768px) {
    flex-direction: column;
    gap: var(--space-sm);
  }
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-lg);
  
  @media (max-width: 1024px) {
    grid-template-columns: repeat(2, 1fr);
  }
  
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
}
```

### 7.2 表格响应式

```tsx
<Table
  columns={columns}
  data={data}
  scroll={{ x: true }} // 横向滚动
/>
```

```css
@media (max-width: 768px) {
  .arco-table {
    font-size: 12px;
  }
  
  .arco-table-th,
  .arco-table-td {
    padding: var(--space-sm);
  }
}
```

## 8. 交互反馈模式

### 8.1 成功提示

```tsx
Message.success('操作成功');

// 或带描述
Notification.success({
  title: '创建成功',
  content: '用户已创建，系统已发送激活邮件',
});
```

### 8.2 错误提示

```tsx
Message.error('操作失败');

// 或带详细错误
Notification.error({
  title: '创建失败',
  content: error.message,
  duration: 5000,
});
```

### 8.3 加载提示

```tsx
const handleSubmit = async () => {
  const loadingKey = Message.loading('提交中...', 0);
  try {
    await api.submit(data);
    Message.success('提交成功');
  } catch (error) {
    Message.error('提交失败');
  } finally {
    Message.remove(loadingKey);
  }
};
```

## 9. 导航模式

### 9.1 面包屑

```tsx
function PageWithBreadcrumb() {
  return (
    <div className="page-container">
      <Breadcrumb>
        <Breadcrumb.Item>系统管理</Breadcrumb.Item>
        <Breadcrumb.Item>用户管理</Breadcrumb.Item>
        <Breadcrumb.Item>用户详情</Breadcrumb.Item>
      </Breadcrumb>
      
      <div className="page-header">
        <h1 className="page-header__title">用户详情</h1>
      </div>
      
      {/* 页面内容 */}
    </div>
  );
}
```

### 9.2 标签页

```tsx
function TabsPage() {
  const [activeTab, setActiveTab] = useState('basic');
  
  return (
    <div className="page-container">
      <Tabs activeTab={activeTab} onChange={setActiveTab}>
        <Tabs.TabPane key="basic" title="基本信息">
          <BasicInfo />
        </Tabs.TabPane>
        <Tabs.TabPane key="security" title="安全设置">
          <SecuritySettings />
        </Tabs.TabPane>
        <Tabs.TabPane key="logs" title="操作日志">
          <OperationLogs />
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
}
```

## 10. 批量操作模式

### 10.1 批量选择

```tsx
function BatchOperationList() {
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: string[]) => setSelectedRowKeys(keys),
  };

  const handleBatchDelete = () => {
    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${selectedRowKeys.length} 项吗？`,
      onOk: async () => {
        await api.batchDelete(selectedRowKeys);
        Message.success('删除成功');
        setSelectedRowKeys([]);
        reload();
      },
    });
  };

  return (
    <div className="page-container">
      {/* 批量操作栏 */}
      {selectedRowKeys.length > 0 && (
        <div className="batch-actions">
          <span>已选 {selectedRowKeys.length} 项</span>
          <Space>
            <Button onClick={() => setSelectedRowKeys([])}>
              取消选择
            </Button>
            <Button status="danger" onClick={handleBatchDelete}>
              批量删除
            </Button>
          </Space>
        </div>
      )}

      <Table
        rowSelection={rowSelection}
        columns={columns}
        data={data}
      />
    </div>
  );
}
```

```css
.batch-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-md);
  background: var(--container-action-bg);
  border: 1px solid var(--container-action-border);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-md);
}
```

## 11. 权限控制模式

### 11.1 按钮权限

```tsx
import { usePermission } from '@/hooks/usePermission';

function UserList() {
  const { hasPermission } = usePermission();

  return (
    <div className="page-header__actions">
      {hasPermission('system:user:create') && (
        <Button type="primary" icon={<IconPlus />}>
          新增用户
        </Button>
      )}
    </div>
  );
}
```

### 11.2 操作列权限

```tsx
const columns = [
  {
    title: '操作',
    render: (_: any, record: User) => (
      <Space>
        {hasPermission('system:user:update') && (
          <Button type="text" size="small" onClick={() => handleEdit(record)}>
            编辑
          </Button>
        )}
        {hasPermission('system:user:delete') && (
          <Button
            type="text"
            size="small"
            status="danger"
            onClick={() => handleDelete(record)}
          >
            删除
          </Button>
        )}
      </Space>
    ),
  },
];
```

## 12. 最佳实践总结

### 12.1 页面结构

1. 使用 `.page-container` 作为页面根容器
2. 使用 `.page-header` 包含标题和操作按钮
3. 筛选工具栏使用 `SearchToolbar` 组件
4. 数据表格包裹在 `Card` 中

### 12.2 样式约定

1. 所有颜色使用 Pantheon token
2. 所有间距使用 `--space-*` token
3. 所有圆角使用 `--radius-*` token
4. 组件样式使用 BEM 命名

### 12.3 交互反馈

1. 操作成功使用 `Message.success`
2. 操作失败使用 `Message.error`
3. 删除操作必须二次确认
4. 加载状态使用 `loading` prop 或 `Spin` 组件

### 12.4 响应式

1. 移动端断点：`max-width: 768px`
2. 表格在移动端支持横向滚动
3. 工具栏在移动端垂直排列
4. 表单在移动端使用 `layout="vertical"`

### 12.5 无障碍

1. 按钮添加 `aria-label`
2. 表单控件关联 `label`
3. 交互元素支持键盘操作
4. 颜色不作为唯一信息传达方式
