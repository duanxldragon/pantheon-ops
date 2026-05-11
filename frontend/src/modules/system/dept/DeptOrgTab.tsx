import React from 'react';
import { Button, Card, Space, Tag } from '@arco-design/web-react';
import { IconEye, IconPlus } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import {
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
} from '../../../components';
import type { DeptNode } from './api';
import type { PostRow } from '../post/api';
import type { UserListRow } from '../user/api';

interface OrgDeptNodeProps {
  dept: DeptNode;
  postsByDept: Map<number, PostRow[]>;
  usersByDept: Map<number, UserListRow[]>;
  usersByPost: Map<number, UserListRow[]>;
  selectedDeptId: number;
  onSelect: (deptID: number) => void;
}

const OrgDeptNode: React.FC<OrgDeptNodeProps> = ({
  dept,
  postsByDept,
  usersByDept,
  usersByPost,
  selectedDeptId,
  onSelect,
}) => {
  const { t } = useTranslation();
  const posts = postsByDept.get(dept.id) || [];
  const deptUsers = usersByDept.get(dept.id) || [];
  const postIDs = new Set(posts.map((post) => post.id));
  const unassignedUsers = deptUsers.filter((user) => !user.postId || !postIDs.has(user.postId));
  const enabledPosts = posts.filter((post) => post.status === 1).length;

  const selectDept = () => onSelect(dept.id);
  const handleKeyDown: React.KeyboardEventHandler<HTMLDivElement> = (event) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      selectDept();
    }
  };

  return (
    <div className="org-chart__branch">
      <div
        className={`org-chart__dept-card${selectedDeptId === dept.id ? ' org-chart__dept-card--active' : ''}`}
        role="button"
        tabIndex={0}
        onClick={selectDept}
        onKeyDown={handleKeyDown}
      >
        <div className="org-chart__dept-header">
          <div className="org-chart__dept-title">
            <span>{dept.deptName}</span>
            {dept.isRoot ? <Tag color="arcoblue">{t('system.dept.root')}</Tag> : null}
          </div>
          <Tag color={dept.status === 1 ? 'green' : 'red'}>
            {dept.status === 1 ? t('system.user.status.enabled') : t('system.user.status.disabled')}
          </Tag>
        </div>
        <div className="org-chart__dept-meta">
          <span>
            {t('system.dept.leader')}: {dept.leader || '-'}
          </span>
          <span>
            {t('system.dept.orgChildren')}: {dept.children?.length || 0}
          </span>
        </div>
        <div className="org-chart__metric-row">
          <span>{t('system.dept.orgPostCount', { count: posts.length })}</span>
          <span>{t('system.dept.orgEnabledPostCount', { count: enabledPosts })}</span>
          <span>{t('system.dept.orgMemberCount', { count: deptUsers.length })}</span>
        </div>
        <div className="org-chart__posts">
          {posts.length > 0 ? (
            posts.map((post) => {
              const members = usersByPost.get(post.id) || [];
              const visibleMembers = members.slice(0, 4);
              return (
                <div className="org-chart__post" key={post.id}>
                  <div className="org-chart__post-head">
                    <span>{post.postName}</span>
                    <Tag size="small" color={post.status === 1 ? 'green' : 'red'}>
                      {post.status === 1
                        ? t('system.user.status.enabled')
                        : t('system.user.status.disabled')}
                    </Tag>
                  </div>
                  <div className="org-chart__post-code">{post.postCode}</div>
                  <div className="org-chart__member-row">
                    {visibleMembers.length > 0 ? (
                      visibleMembers.map((member) => (
                        <Tag key={member.id} size="small" color="arcoblue">
                          {member.nickname || member.username}
                        </Tag>
                      ))
                    ) : (
                      <span>{t('system.dept.orgNoMembers')}</span>
                    )}
                    {members.length > visibleMembers.length ? (
                      <Tag size="small">+{members.length - visibleMembers.length}</Tag>
                    ) : null}
                  </div>
                </div>
              );
            })
          ) : (
            <div className="org-chart__empty-line">{t('system.dept.orgNoPosts')}</div>
          )}
          {unassignedUsers.length > 0 ? (
            <div className="org-chart__post org-chart__post--unassigned">
              <div className="org-chart__post-head">
                <span>{t('system.dept.orgUnassignedPost')}</span>
              </div>
              <div className="org-chart__member-row">
                {unassignedUsers.slice(0, 6).map((member) => (
                  <Tag key={member.id} size="small">
                    {member.nickname || member.username}
                  </Tag>
                ))}
                {unassignedUsers.length > 6 ? (
                  <Tag size="small">+{unassignedUsers.length - 6}</Tag>
                ) : null}
              </div>
            </div>
          ) : null}
        </div>
      </div>
      {dept.children?.length ? (
        <div className="org-chart__children">
          {dept.children.map((child) => (
            <OrgDeptNode
              key={child.id}
              dept={child}
              postsByDept={postsByDept}
              usersByDept={usersByDept}
              usersByPost={usersByPost}
              selectedDeptId={selectedDeptId}
              onSelect={onSelect}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
};

interface DeptOrgTabProps {
  orgDepts: DeptNode[];
  orgPosts: PostRow[];
  orgUsers: UserListRow[];
  orgLoading: boolean;
  orgError: unknown;
  selectedOrgDeptId: number;
  onSelectDept: (deptId: number) => void;
  flatOrgDepts: DeptNode[];
  postsByDept: Map<number, PostRow[]>;
  usersByDept: Map<number, UserListRow[]>;
  usersByPost: Map<number, UserListRow[]>;
  selectedOrgDept: DeptNode | undefined;
  selectedOrgStats: { deptCount: number; postCount: number; userCount: number };
  canViewPosts: boolean;
  canViewUsers: boolean;
  canCreatePost: boolean;
  canViewUserDetail: boolean;
  onRefresh: () => void;
  onCreatePost: () => void;
  onViewUserDetail: (userId: number) => void;
}

const DeptOrgTab: React.FC<DeptOrgTabProps> = ({
  orgDepts,
  orgPosts,
  orgUsers,
  orgLoading,
  orgError,
  selectedOrgDeptId,
  onSelectDept,
  flatOrgDepts,
  postsByDept,
  usersByDept,
  usersByPost,
  selectedOrgDept,
  selectedOrgStats,
  canViewPosts,
  canViewUsers,
  canCreatePost,
  canViewUserDetail,
  onRefresh,
  onCreatePost,
  onViewUserDetail,
}) => {
  const { t } = useTranslation();

  const renderOrgErrorState = () => {
    if (isNetworkRequestError(orgError)) {
      return <PageNetworkError timeout={isTimeoutRequestError(orgError)} onRetry={onRefresh} />;
    }
    if (isServerRequestError(orgError)) {
      return <PageServerError onRetry={onRefresh} />;
    }
    return <PageError onRetry={onRefresh} />;
  };

  if (orgLoading && orgDepts.length === 0) {
    return <PageLoading />;
  }
  if (orgError && orgDepts.length === 0) {
    return renderOrgErrorState();
  }
  if (!orgLoading && !orgError && orgDepts.length === 0) {
    return <PageEmpty description={t('system.dept.orgNoDept')} />;
  }

  const selectedPosts = selectedOrgDept ? postsByDept.get(selectedOrgDept.id) || [] : [];
  const selectedUsers = selectedOrgDept ? usersByDept.get(selectedOrgDept.id) || [] : [];

  return (
    <Space direction="vertical" size={16} className="org-structure">
      <div className="org-structure__summary">
        <Card className="org-structure__summary-card">
          <span>{t('system.dept.orgDeptTotal')}</span>
          <strong>{flatOrgDepts.length}</strong>
        </Card>
        <Card className="org-structure__summary-card">
          <span>{t('system.dept.orgPostTotal')}</span>
          <strong>{orgPosts.length}</strong>
        </Card>
        <Card className="org-structure__summary-card">
          <span>{t('system.dept.orgMemberTotal')}</span>
          <strong>{orgUsers.length}</strong>
        </Card>
        <Card className="org-structure__summary-card org-structure__summary-card--active">
          <span>{t('system.dept.orgSelectedScope')}</span>
          <strong>
            {selectedOrgStats.deptCount}/{selectedOrgStats.postCount}/{selectedOrgStats.userCount}
          </strong>
        </Card>
      </div>
      {!canViewPosts || !canViewUsers ? (
        <Card className="org-structure__notice">
          {!canViewPosts ? <span>{t('system.dept.orgPostPermissionHint')}</span> : null}
          {!canViewUsers ? <span>{t('system.dept.orgUserPermissionHint')}</span> : null}
        </Card>
      ) : null}
      <div className="org-structure__body">
        <Card className="page-panel org-structure__chart-card">
          <div className="org-structure__section-head">
            <div>
              <div className="org-structure__section-title">{t('system.dept.orgChartTitle')}</div>
              <div className="org-structure__section-desc">{t('system.dept.orgChartHint')}</div>
            </div>
            <Button size="small" onClick={onRefresh} loading={orgLoading}>
              {t('common.refresh')}
            </Button>
          </div>
          <div className="org-chart">
            {orgDepts.map((dept) => (
              <OrgDeptNode
                key={dept.id}
                dept={dept}
                postsByDept={postsByDept}
                usersByDept={usersByDept}
                usersByPost={usersByPost}
                selectedDeptId={selectedOrgDeptId}
                onSelect={onSelectDept}
              />
            ))}
          </div>
        </Card>
        <Card className="page-panel org-structure__detail-card">
          <div className="org-structure__section-head">
            <div>
              <div className="org-structure__section-title">
                {selectedOrgDept?.deptName || t('system.dept.orgNoSelection')}
              </div>
              <div className="org-structure__section-desc">{t('system.dept.orgDetailHint')}</div>
            </div>
            <Button
              size="small"
              type="primary"
              icon={<IconPlus />}
              disabled={!canCreatePost || !selectedOrgDept || selectedOrgDept.isRoot}
              onClick={onCreatePost}
            >
              {t('system.dept.orgAddPost')}
            </Button>
          </div>
          {selectedOrgDept ? (
            <Space direction="vertical" size={14} className="org-structure__detail">
              <div className="org-structure__detail-grid">
                <div>
                  <span>{t('system.dept.leader')}</span>
                  <strong>{selectedOrgDept.leader || '-'}</strong>
                </div>
                <div>
                  <span>{t('system.dept.phone')}</span>
                  <strong>{selectedOrgDept.phone || '-'}</strong>
                </div>
                <div>
                  <span>{t('system.dept.orgChildren')}</span>
                  <strong>{selectedOrgDept.children?.length || 0}</strong>
                </div>
                <div>
                  <span>{t('system.dept.status')}</span>
                  <strong>
                    {selectedOrgDept.status === 1
                      ? t('system.user.status.enabled')
                      : t('system.user.status.disabled')}
                  </strong>
                </div>
              </div>
              <div>
                <div className="org-structure__sub-title-row">
                  <div className="org-structure__sub-title">{t('system.dept.orgDirectPosts')}</div>
                  {selectedOrgDept.isRoot ? <span>{t('system.dept.orgRootPostHint')}</span> : null}
                </div>
                <div className="org-structure__tag-list">
                  {selectedPosts.length > 0 ? (
                    selectedPosts.map((post) => (
                      <Tag key={post.id} color={post.status === 1 ? 'arcoblue' : 'gray'}>
                        {post.postName}
                      </Tag>
                    ))
                  ) : (
                    <span>{t('system.dept.orgNoPosts')}</span>
                  )}
                </div>
              </div>
              <div>
                <div className="org-structure__sub-title">{t('system.dept.orgDirectMembers')}</div>
                <div className="org-structure__member-list">
                  {selectedUsers.length > 0 ? (
                    selectedUsers.map((user) => (
                      <div className="org-structure__member-item" key={user.id}>
                        <div>
                          <strong>{user.nickname || user.username}</strong>
                          <span>
                            {user.username} · {user.postName || t('system.post.none')}
                          </span>
                        </div>
                        <Button
                          size="mini"
                          type="text"
                          icon={<IconEye />}
                          disabled={!canViewUserDetail}
                          onClick={() => onViewUserDetail(user.id)}
                        >
                          {t('common.detail')}
                        </Button>
                      </div>
                    ))
                  ) : (
                    <span>{t('system.dept.orgNoMembers')}</span>
                  )}
                </div>
              </div>
              <div className="org-structure__rule">{t('system.dept.orgRelationRule')}</div>
            </Space>
          ) : (
            <PageEmpty description={t('system.dept.orgNoSelection')} />
          )}
        </Card>
      </div>
    </Space>
  );
};

export default DeptOrgTab;
export { OrgDeptNode };
export type { OrgDeptNodeProps };
