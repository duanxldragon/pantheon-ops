import React from 'react';
import { Button, Dropdown, Menu, Popconfirm, Space } from '@arco-design/web-react';
import { IconMore } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { showAppModalConfirm } from './modals/AppModalActions';

type SystemRowActionConfirm = {
  title?: React.ReactNode;
  content?: React.ReactNode;
  okText?: React.ReactNode;
  cancelText?: React.ReactNode;
  onOk?: () => void | Promise<void>;
  onCancel?: () => void;
  disabled?: boolean;
};

export interface SystemRowAction {
  key: React.Key;
  text?: React.ReactNode;
  icon?: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  hidden?: boolean;
  loading?: boolean;
  status?: React.ComponentProps<typeof Button>['status'];
  confirm?: SystemRowActionConfirm;
  node?: React.ReactNode;
  iconOnly?: boolean;
  ariaLabel?: string;
}

interface SystemRowActionsProps {
  actions: SystemRowAction[];
  className?: string;
  maxInlineActions?: number;
  overflowLabel?: string;
}

const SystemRowActions: React.FC<SystemRowActionsProps> = ({
  actions,
  className,
  maxInlineActions = 3,
  overflowLabel,
}) => {
  const { t } = useTranslation();
  const visibleActions = actions.filter((action) => !action.hidden);
  if (visibleActions.length === 0) {
    return null;
  }

  const buttonActions = visibleActions.filter((action) => !action.node);
  const inlineButtonLimit = Math.max(0, maxInlineActions);
  const overflowButtonCount = Math.max(0, buttonActions.length - inlineButtonLimit);
  const overflowButtonStart = buttonActions.length - overflowButtonCount;
  const inlineActions: SystemRowAction[] = [];
  const overflowActions: SystemRowAction[] = [];
  let buttonIndex = 0;

  for (const action of visibleActions) {
    if (action.node) {
      inlineActions.push(action);
      continue;
    }

    if (buttonIndex >= overflowButtonStart) {
      overflowActions.push(action);
    } else {
      inlineActions.push(action);
    }

    buttonIndex += 1;
  }

  const resolvedOverflowLabel = overflowLabel ?? t('common.actions');

  const runAction = (action: SystemRowAction) => {
    if (action.disabled || action.loading) {
      return;
    }

    if (action.confirm) {
      const { title, content, okText, cancelText, onOk, onCancel } = action.confirm;
      showAppModalConfirm({
        title,
        content,
        okText,
        cancelText,
        onOk,
        onCancel,
        size: 'sm',
      });
      return;
    }

    action.onClick?.();
  };

  const renderButtonAction = (action: SystemRowAction) => {
    if (action.node) {
      return <React.Fragment key={action.key}>{action.node}</React.Fragment>;
    }

    const isIconOnly = action.iconOnly || !action.text;
    const label =
      action.ariaLabel ?? (typeof action.text === 'string' ? action.text : String(action.key));
    const button = (
      <Button
        key={action.key}
        type="text"
        size="small"
        icon={action.icon}
        status={action.status}
        loading={action.loading}
        disabled={action.disabled}
        onClick={action.confirm ? undefined : action.onClick}
        aria-label={isIconOnly ? label : undefined}
        title={isIconOnly ? label : undefined}
      >
        {isIconOnly ? null : action.text}
      </Button>
    );

    if (!action.confirm) {
      return button;
    }

    return (
      <Popconfirm
        key={action.key}
        {...action.confirm}
        disabled={action.confirm.disabled ?? action.disabled ?? action.loading}
      >
        {button}
      </Popconfirm>
    );
  };

  const overflowMenu =
    overflowActions.length > 0 ? (
      <Menu
        className="system-row-actions__overflow-menu"
        onClickMenuItem={(key) => {
          const action = overflowActions.find((item) => String(item.key) === String(key));
          if (!action) {
            return;
          }

          runAction(action);
        }}
      >
        {overflowActions.map((action) => {
          const isDanger = action.status === 'danger' || Boolean(action.confirm);
          const label = action.text ?? action.ariaLabel ?? String(action.key);

          return (
            <Menu.Item
              key={String(action.key)}
              disabled={action.disabled || action.loading}
              className={[
                'system-row-actions__overflow-item',
                isDanger ? 'system-row-actions__overflow-item--danger' : '',
              ]
                .filter(Boolean)
                .join(' ')}
            >
              <span className="system-row-actions__overflow-item-copy">
                {action.icon ? (
                  <span className="system-row-actions__overflow-item-icon">{action.icon}</span>
                ) : null}
                <span className="system-row-actions__overflow-item-label">{label}</span>
              </span>
            </Menu.Item>
          );
        })}
      </Menu>
    ) : null;

  return (
    <Space size={4} className={['system-list__actions', className].filter(Boolean).join(' ')}>
      {inlineActions.map((action) => renderButtonAction(action))}
      {overflowMenu ? (
        <Dropdown trigger="click" position="br" droplist={overflowMenu}>
          <Button
            type="text"
            size="small"
            icon={<IconMore />}
            aria-label={resolvedOverflowLabel}
            title={resolvedOverflowLabel}
            className="system-row-actions__overflow-trigger"
          />
        </Dropdown>
      ) : null}
    </Space>
  );
};

export default SystemRowActions;
