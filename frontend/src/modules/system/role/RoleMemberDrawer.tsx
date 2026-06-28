import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Input, Select, Space, Tag, Typography } from '@arco-design/web-react';
import { IconDelete, IconPlus, IconRefresh, IconSearch } from '@arco-design/web-react/icon';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import {
  AppDrawer,
  AppTable,
  buildStandardPagination,
  PageEmpty,
  PageError,
  PageLoading,
  PermissionAction,
  showAppModalConfirm,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
} from '../../../components';
import { message } from '../../../components/feedback/message';
import { formatDateTime } from '../../../core/format/dateTime';
import {
  addRoleMembers,
  getRoleMemberCandidates,
  getRoleMembers,
  removeRoleMembers,
  type RoleMemberPageResp,
  type RoleMemberQuery,
  type RoleMemberRow,
  type RoleRow,
} from './api';

interface RoleMemberDrawerProps {
  role: RoleRow | null;
  visible: boolean;
  canEdit: boolean;
  onClose: () => void;
  onMembershipChanged?: () => void;
}

const defaultMemberQuery: RoleMemberQuery = {
  keyword: '',
  status: undefined,
  page: 1,
  pageSize: 10,
};

const RoleMemberDrawer: React.FC<RoleMemberDrawerProps> = ({
  role,
  visible,
  canEdit,
  onClose,
  onMembershipChanged,
}) => {
  const { t } = useTranslation();
  const [memberRows, setMemberRows] = useState<RoleMemberRow[]>([]);
  const [memberTotal, setMemberTotal] = useState(0);
  const [memberLoading, setMemberLoading] = useState(false);
  const [memberError, setMemberError] = useState(false);
  const [memberQuery, setMemberQuery] = useState<RoleMemberQuery>(defaultMemberQuery);
  const [candidateOptions, setCandidateOptions] = useState<Array<{ label: string; value: number }>>(
    [],
  );
  const [candidateLoading, setCandidateLoading] = useState(false);
  const [candidateKeyword, setCandidateKeyword] = useState('');
  const [selectedCandidateIds, setSelectedCandidateIds] = useState<number[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const resetDrawerState = () => {
    setMemberRows([]);
    setMemberTotal(0);
    setMemberError(false);
    setMemberQuery(defaultMemberQuery);
    setCandidateOptions([]);
    setCandidateKeyword('');
    setSelectedCandidateIds([]);
  };

  const handleClose = () => {
    resetDrawerState();
    onClose();
  };

  const toggleCandidate = (userId: number) => {
    setSelectedCandidateIds((current) =>
      current.includes(userId) ? current.filter((item) => item !== userId) : [...current, userId],
    );
  };

  const loadMembers = useCallback(
    async (nextQuery: RoleMemberQuery = defaultMemberQuery) => {
      if (!role) {
        return;
      }
      setMemberLoading(true);
      setMemberError(false);
      try {
        const result = await getRoleMembers(role.id, nextQuery);
        setMemberRows(result.items);
        setMemberTotal(result.total);
        setMemberQuery({
          keyword: nextQuery.keyword || '',
          status: nextQuery.status,
          page: result.page || nextQuery.page || defaultMemberQuery.page,
          pageSize: result.pageSize || nextQuery.pageSize || defaultMemberQuery.pageSize,
        });
      } catch {
        setMemberError(true);
        setMemberRows([]);
        setMemberTotal(0);
      } finally {
        setMemberLoading(false);
      }
    },
    [role],
  );

  const loadCandidates = useCallback(
    async (keyword = '') => {
      if (!role) {
        return;
      }
      setCandidateLoading(true);
      try {
        const result: RoleMemberPageResp = await getRoleMemberCandidates(role.id, {
          keyword,
          page: 1,
          pageSize: 20,
        });
        setCandidateOptions(
          result.items.map((item) => ({
            label: item.nickname ? `${item.username} / ${item.nickname}` : item.username,
            value: item.id,
          })),
        );
      } catch {
        setCandidateOptions([]);
        message.error(t('common.loadFailed'));
      } finally {
        setCandidateLoading(false);
      }
    },
    [role, t],
  );

  useEffect(() => {
    if (!visible || !role) {
      return;
    }
    const timer = globalThis.setTimeout(() => {
      void loadMembers(defaultMemberQuery);
      void loadCandidates('');
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadCandidates, loadMembers, role, visible]);

  const handleMemberTableChange: TableProps<RoleMemberRow>['onChange'] = (pagination) => {
    const nextQuery = {
      ...memberQuery,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || memberQuery.pageSize || defaultMemberQuery.pageSize,
    };
    void loadMembers(nextQuery);
  };

  const handleAdd = async () => {
    if (!role || selectedCandidateIds.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      const result = await addRoleMembers(role.id, { userIds: selectedCandidateIds });
      message.success(t('system.role.members.addSuccess', { count: result.addedCount }));
      setSelectedCandidateIds([]);
      onMembershipChanged?.();
      await loadMembers({ ...memberQuery, page: 1 });
      await loadCandidates('');
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemove = useCallback(
    async (userId: number) => {
      if (!role) {
        return;
      }
      setSubmitting(true);
      try {
        const result = await removeRoleMembers(role.id, { userIds: [userId] });
        message.success(t('system.role.members.removeSuccess', { count: result.removedCount }));
        onMembershipChanged?.();
        await loadMembers({
          ...memberQuery,
          page:
            memberRows.length === 1 && (memberQuery.page || 1) > 1
              ? (memberQuery.page || 1) - 1
              : memberQuery.page || 1,
        });
        await loadCandidates(candidateKeyword);
      } catch {
        message.error(t('common.actionFailed'));
      } finally {
        setSubmitting(false);
      }
    },
    [
      candidateKeyword,
      loadCandidates,
      loadMembers,
      memberQuery,
      memberRows.length,
      onMembershipChanged,
      role,
      t,
    ],
  );

  const confirmRemove = useCallback(
    (userId: number) => {
      showAppModalConfirm({
        size: 'sm',
        title: t('system.role.members.removeConfirm'),
        onOk: () => handleRemove(userId),
      });
    },
    [handleRemove, t],
  );

  const memberColumns: ColumnProps<RoleMemberRow>[] = useMemo(
    () => [
      {
        title: t('system.user.username'),
        dataIndex: 'username',
        width: TABLE_COLUMN_WIDTH.name,
      },
      {
        title: t('system.user.nickname'),
        dataIndex: 'nickname',
        width: TABLE_COLUMN_WIDTH.identity,
        render: (value: string) => value || '-',
      },
      {
        title: t('system.user.status'),
        dataIndex: 'status',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: number) => (
          <Tag color={value === 1 ? 'green' : 'red'}>
            {value === 1 ? t('system.user.status.enabled') : t('system.user.status.disabled')}
          </Tag>
        ),
      },
      {
        title: t('system.user.dept'),
        dataIndex: 'deptName',
        width: TABLE_COLUMN_WIDTH.name,
        render: (value: string) => value || '-',
      },
      {
        title: t('system.user.post'),
        dataIndex: 'postName',
        width: TABLE_COLUMN_WIDTH.name,
        render: (value: string) => value || '-',
      },
      {
        title: t('system.user.createdAt'),
        dataIndex: 'createdAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        render: (value: string) => formatDateTime(value),
      },
      {
        title: t('common.action'),
        width: TABLE_ACTION_COLUMN_WIDTH.compact,
        fixed: 'right',
        render: (_: unknown, row: RoleMemberRow) => (
          <PermissionAction allowed={canEdit} tooltip={t('common.noPermissionAction')}>
            <Button
              type="text"
              status="danger"
              icon={<IconDelete />}
              disabled={!canEdit}
              onClick={() => {
                confirmRemove(row.id);
              }}
            >
              {t('common.delete')}
            </Button>
          </PermissionAction>
        ),
      },
    ],
    [canEdit, confirmRemove, t],
  );

  const drawerTitle = role ? (
    <div className="role-member-drawer__title">
      <Typography.Text className="role-member-drawer__title-main">{role.roleName}</Typography.Text>
      <Typography.Text type="secondary" className="role-member-drawer__title-sub">
        {role.roleKey} · {t('system.role.members')}
      </Typography.Text>
    </div>
  ) : (
    t('system.role.members')
  );

  return (
    <AppDrawer
      className="role-member-drawer"
      title={drawerTitle}
      visible={visible}
      size="detail"
      footer={null}
      onCancel={handleClose}
    >
      <Space direction="vertical" size={16} className="role-member-drawer__stack">
        <Card className="dialog-grid-card role-member-drawer__summary-card" size="small">
          <div className="role-member-drawer__summary">
            <div className="role-member-drawer__summary-copy">
              <Typography.Text className="role-member-drawer__summary-title">
                {t('system.role.members.subtitle')}
              </Typography.Text>
              <Typography.Text type="secondary">{t('system.role.members.hint')}</Typography.Text>
            </div>
            <div className="role-member-drawer__summary-meta">
              <Tag color={role?.status === 1 ? 'green' : 'red'}>
                {role?.status === 1
                  ? t('system.user.status.enabled')
                  : t('system.user.status.disabled')}
              </Tag>
              <Typography.Text type="secondary">
                {t('common.total')} {memberTotal} / {t('system.role.members')}
              </Typography.Text>
            </div>
          </div>
          <div className="role-member-drawer__toolbar">
            <Input
              allowClear
              prefix={<IconSearch />}
              value={candidateKeyword}
              placeholder={t('system.role.members.candidatePlaceholder')}
              onChange={(value) => {
                setCandidateKeyword(value);
                void loadCandidates(value);
              }}
              onPressEnter={() => {
                void loadCandidates(candidateKeyword);
              }}
            />
            <PermissionAction allowed={canEdit} tooltip={t('common.noPermissionAction')}>
              <Button
                type="primary"
                icon={<IconPlus />}
                loading={submitting}
                disabled={!canEdit || selectedCandidateIds.length === 0}
                onClick={() => {
                  void handleAdd();
                }}
              >
                {t('common.add')}
              </Button>
            </PermissionAction>
            <Button
              icon={<IconRefresh />}
              loading={memberLoading}
              onClick={() => {
                void loadMembers(memberQuery);
                void loadCandidates(candidateKeyword);
              }}
            >
              {t('common.refresh')}
            </Button>
          </div>
          <div
            className="role-member-drawer__candidate-list"
            role="listbox"
            aria-label={t('system.role.members.candidatePlaceholder')}
          >
            {candidateOptions.length > 0 ? (
              candidateOptions.map((option) => {
                const selected = selectedCandidateIds.includes(option.value);
                return (
                  <Button
                    key={option.value}
                    className={`role-member-drawer__candidate-pill${selected ? ' role-member-drawer__candidate-pill--selected' : ''}`}
                    type={selected ? 'primary' : 'outline'}
                    size="small"
                    onClick={() => toggleCandidate(option.value)}
                  >
                    {option.label}
                  </Button>
                );
              })
            ) : (
              <Typography.Text type="secondary" className="role-member-drawer__candidate-empty">
                {candidateLoading ? t('common.loading') : t('system.role.members.candidateEmpty')}
              </Typography.Text>
            )}
          </div>
        </Card>

        <Card className="dialog-grid-card" size="small">
          <div className="role-member-drawer__filter-row">
            <Input
              allowClear
              prefix={<IconSearch />}
              value={memberQuery.keyword}
              placeholder={t('system.role.members.searchPlaceholder')}
              onChange={(value) => {
                setMemberQuery((current) => ({ ...current, keyword: value }));
              }}
              onPressEnter={() => {
                void loadMembers({ ...memberQuery, page: 1 });
              }}
            />
            <Select
              allowClear
              value={memberQuery.status}
              placeholder={t('system.role.members.filterStatus')}
              options={[
                { label: t('system.user.status.enabled'), value: 1 },
                { label: t('system.user.status.disabled'), value: 2 },
              ]}
              onChange={(value) => {
                setMemberQuery((current) => ({
                  ...current,
                  status: typeof value === 'number' ? value : undefined,
                }));
              }}
            />
            <Button
              type="primary"
              onClick={() => {
                void loadMembers({ ...memberQuery, page: 1 });
              }}
            >
              {t('common.search')}
            </Button>
            <Button
              onClick={() => {
                setMemberQuery(defaultMemberQuery);
                void loadMembers(defaultMemberQuery);
              }}
            >
              {t('common.reset')}
            </Button>
          </div>
          {memberLoading && memberRows.length === 0 ? <PageLoading /> : null}
          {!memberLoading && memberError ? (
            <PageError
              onRetry={() => {
                void loadMembers(memberQuery);
              }}
            />
          ) : null}
          {!memberLoading && !memberError && memberRows.length === 0 ? (
            <PageEmpty description={t('system.role.members.empty')} />
          ) : null}
          {!memberError && memberRows.length > 0 ? (
            <AppTable<RoleMemberRow>
              className="system-list__table"
              rowKey="id"
              data={memberRows}
              columns={memberColumns}
              loading={memberLoading}
              scroll={{ x: 'max-content' }}
              pagination={buildStandardPagination(t, {
                current: memberQuery.page || defaultMemberQuery.page,
                pageSize: memberQuery.pageSize || defaultMemberQuery.pageSize,
                total: memberTotal,
              })}
              onChange={handleMemberTableChange}
              emptyText={t('common.noData')}
            />
          ) : null}
        </Card>
      </Space>
    </AppDrawer>
  );
};

export default RoleMemberDrawer;
