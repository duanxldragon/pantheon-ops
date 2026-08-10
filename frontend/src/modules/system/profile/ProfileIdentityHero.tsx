import React from 'react';
import { Avatar, Space, Tag, Typography } from '@arco-design/web-react';
import {
  filterIdentityLabels,
  shouldShowIdentityLabel,
  UserAvatarContent,
} from '../../../components';

export interface ProfileIdentityHeroProps {
  avatar?: string;
  nickname?: string;
  username?: string;
  roles?: string[];
  fallbackName: string;
}

const ProfileIdentityHero: React.FC<ProfileIdentityHeroProps> = ({
  avatar,
  nickname,
  username,
  roles,
  fallbackName,
}) => {
  const displayName = nickname || username || fallbackName;
  const showUsername = shouldShowIdentityLabel(displayName, username || '');
  const visibleRoles = filterIdentityLabels(displayName, roles);

  return (
    <Space align="start" size={16}>
      <Avatar size={56}>
        <UserAvatarContent avatar={avatar} userDisplayName={displayName} />
      </Avatar>
      <Space direction="vertical" size={4}>
        <Typography.Title heading={5} style={{ margin: 0 }}>
          {displayName}
        </Typography.Title>
        {showUsername ? <Typography.Text type="secondary">{username}</Typography.Text> : null}
        {visibleRoles.length > 0 ? (
          <Space wrap>
            {visibleRoles.map((role) => (
              <Tag key={role} color="arcoblue">
                {role}
              </Tag>
            ))}
          </Space>
        ) : null}
      </Space>
    </Space>
  );
};

export default ProfileIdentityHero;
