import React from 'react';
import { Button, Popconfirm, Space } from '@arco-design/web-react';

export interface SystemRowAction {
  key: React.Key;
  text?: React.ReactNode;
  icon?: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  hidden?: boolean;
  status?: React.ComponentProps<typeof Button>['status'];
  confirm?: Omit<React.ComponentProps<typeof Popconfirm>, 'children'>;
  node?: React.ReactNode;
}

interface SystemRowActionsProps {
  actions: SystemRowAction[];
  className?: string;
}

const SystemRowActions: React.FC<SystemRowActionsProps> = ({ actions, className }) => {
  const visibleActions = actions.filter((action) => !action.hidden);
  if (visibleActions.length === 0) {
    return null;
  }

  return (
    <Space size={4} className={['system-list__actions', className].filter(Boolean).join(' ')}>
      {visibleActions.map((action) => {
        if (action.node) {
          return <React.Fragment key={action.key}>{action.node}</React.Fragment>;
        }

        if (!action.text) {
          return null;
        }

        const button = (
          <Button
            key={action.key}
            type="text"
            size="small"
            icon={action.icon}
            status={action.status}
            disabled={action.disabled}
            onClick={action.onClick}
          >
            {action.text}
          </Button>
        );

        if (!action.confirm) {
          return button;
        }

        return (
          <Popconfirm
            key={action.key}
            {...action.confirm}
            disabled={action.confirm.disabled ?? action.disabled}
          >
            {button}
          </Popconfirm>
        );
      })}
    </Space>
  );
};

export default SystemRowActions;
