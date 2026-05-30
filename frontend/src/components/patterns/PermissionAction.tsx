import React from 'react';
import { Tooltip } from '@arco-design/web-react';

interface PermissionActionProps {
  allowed: boolean;
  tooltip: string;
  children: React.ReactElement<{ disabled?: boolean }>;
}

const PermissionAction: React.FC<PermissionActionProps> = ({ allowed, tooltip, children }) => {
  if (allowed) {
    return children;
  }

  return (
    <Tooltip content={tooltip}>
      <span className="permission-action">{React.cloneElement(children, { disabled: true })}</span>
    </Tooltip>
  );
};

export default PermissionAction;
