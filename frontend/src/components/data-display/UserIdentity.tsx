import { useState } from 'react';
import { getUserInitial } from './userIdentityHelpers';

export interface UserAvatarContentProps {
  avatar?: string;
  userDisplayName: string;
}

interface UserAvatarImageProps extends UserAvatarContentProps {
  avatar: string;
}

function UserAvatarImage({ avatar, userDisplayName }: Readonly<UserAvatarImageProps>) {
  const [failed, setFailed] = useState(false);

  if (failed) {
    return getUserInitial(userDisplayName);
  }

  return <img src={avatar} alt="" onError={() => setFailed(true)} />;
}

export default function UserAvatarContent({
  avatar,
  userDisplayName,
}: Readonly<UserAvatarContentProps>) {
  if (!avatar) {
    return getUserInitial(userDisplayName);
  }

  return <UserAvatarImage key={avatar} avatar={avatar} userDisplayName={userDisplayName} />;
}
