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

  const disabledChild = React.createElement(children.type as React.ElementType, {
    ...children.props,
    disabled: true,
  });

  return (
    <Tooltip content={tooltip}>
      <span className="permission-action">{disabledChild}</span>
    </Tooltip>
  );
};

export default PermissionAction;
