import React, { useRef, useState } from 'react';
import { Dropdown, Menu } from '@arco-design/web-react';
import { IconClose, IconPushpin } from '@arco-design/web-react/icon';
import { preloadRouteComponent } from '../router/prefetch';
import type { ShellLayoutMode } from '../shellState';
import type { OpenedPageTab, TabActionKey } from './layoutTabs';

type TranslateLabel = (key: string, options?: Record<string, unknown>) => string;

interface LayoutOpenedTabsProps {
  enabled: boolean;
  layoutMode: ShellLayoutMode;
  tabs: OpenedPageTab[];
  currentPath: string;
  onNavigate: (path: string) => void;
  onMoveTab: (dragPath: string, targetPath: string) => void;
  onTabAction: (targetPath: string, action: TabActionKey) => void;
  t: TranslateLabel;
}

// Keyboard model (manual-activation tabs): every tab is a tab stop; Enter and
// Space activate the focused tab; Arrow keys / Home / End move focus between
// tabs (Left/Right in horizontal mode, Up/Down in vertical mode). The close
// button inside each tab is its own keyboard-accessible control.
function getTabFocusOffset(
  layoutMode: ShellLayoutMode,
  key: string,
): number | 'home' | 'end' | null {
  if (layoutMode === 'horizontal') {
    if (key === 'ArrowRight') return 1;
    if (key === 'ArrowLeft') return -1;
  } else {
    if (key === 'ArrowDown') return 1;
    if (key === 'ArrowUp') return -1;
  }
  if (key === 'Home') return 'home';
  if (key === 'End') return 'end';
  return null;
}

const LayoutOpenedTabs: React.FC<LayoutOpenedTabsProps> = ({
  enabled,
  layoutMode,
  tabs,
  currentPath,
  onNavigate,
  onMoveTab,
  onTabAction,
  t,
}) => {
  const [draggingTabPath, setDraggingTabPath] = useState<string | null>(null);
  const [dragOverTabPath, setDragOverTabPath] = useState<string | null>(null);
  const tabRefs = useRef(new Map<string, HTMLDivElement>());

  const moveTabFocus = (offset: number | 'home' | 'end', currentIndex: number) => {
    if (tabs.length === 0) {
      return;
    }
    let nextIndex: number;
    if (offset === 'home') {
      nextIndex = 0;
    } else if (offset === 'end') {
      nextIndex = tabs.length - 1;
    } else {
      nextIndex = (currentIndex + offset + tabs.length) % tabs.length;
    }
    tabRefs.current.get(tabs[nextIndex].path)?.focus();
  };

  if (!enabled) {
    return null;
  }

  return (
    <div
      className={[
        'app-shell__tabs',
        layoutMode === 'horizontal' ? 'app-shell__tabs--horizontal' : 'app-shell__tabs--vertical',
      ].join(' ')}
      role="tablist"
      aria-label={t('app.openedTabs')}
    >
      {tabs.map((item) => {
        const active = item.path === currentPath;
        const itemIndex = tabs.findIndex((tab) => tab.path === item.path);
        const canCloseCurrent = item.closable && !item.pinned;
        const canCloseOthers = tabs.some(
          (tab) => tab.path !== item.path && tab.closable && !tab.pinned,
        );
        const canCloseRight = tabs.slice(itemIndex + 1).some((tab) => tab.closable && !tab.pinned);
        const canCloseAll = tabs.some((tab) => tab.closable && !tab.pinned);

        return (
          <Dropdown
            key={item.path}
            trigger="contextMenu"
            position="bl"
            droplist={
              <Menu
                onClickMenuItem={(key) => onTabAction(item.path, key as TabActionKey)}
                className="app-shell__tab-menu"
              >
                <Menu.Item key="togglePin" disabled={item.path === '/dashboard'}>
                  {item.pinned ? t('app.tab.unpin') : t('app.tab.pin')}
                </Menu.Item>
                <Menu.Item key="close" disabled={!canCloseCurrent}>
                  {t('common.close')}
                </Menu.Item>
                <Menu.Item key="closeOthers" disabled={!canCloseOthers}>
                  {t('app.tab.closeOthers')}
                </Menu.Item>
                <Menu.Item key="closeRight" disabled={!canCloseRight}>
                  {t('app.tab.closeRight')}
                </Menu.Item>
                <Menu.Item key="closeAll" disabled={!canCloseAll}>
                  {t('app.tab.closeAll')}
                </Menu.Item>
              </Menu>
            }
          >
            <div
              role="tab"
              tabIndex={0}
              aria-selected={active}
              ref={(element) => {
                if (element) {
                  tabRefs.current.set(item.path, element);
                } else {
                  tabRefs.current.delete(item.path);
                }
              }}
              className={[
                'app-shell__tab',
                active ? 'app-shell__tab--active' : '',
                item.pinned ? 'app-shell__tab--pinned' : '',
                draggingTabPath === item.path ? 'app-shell__tab--dragging' : '',
                dragOverTabPath === item.path ? 'app-shell__tab--drag-over' : '',
              ]
                .filter(Boolean)
                .join(' ')}
              draggable={item.path !== '/dashboard'}
              onClick={() => onNavigate(item.path)}
              onMouseEnter={() => {
                preloadRouteComponent(item.path).catch(() => undefined);
              }}
              onFocus={() => {
                preloadRouteComponent(item.path).catch(() => undefined);
              }}
              onDoubleClick={() => onTabAction(item.path, 'close')}
              onMouseDown={(event) => {
                if (event.button === 1) {
                  event.preventDefault();
                }
              }}
              onAuxClick={(event) => {
                if (event.button === 1) {
                  event.preventDefault();
                  onTabAction(item.path, 'close');
                }
              }}
              onDragStart={(event) => {
                if (item.path === '/dashboard') {
                  event.preventDefault();
                  return;
                }
                setDraggingTabPath(item.path);
                event.dataTransfer.effectAllowed = 'move';
                event.dataTransfer.setData('text/plain', item.path);
              }}
              onDragOver={(event) => {
                if (!draggingTabPath || draggingTabPath === item.path) {
                  return;
                }
                const dragTab = tabs.find((tab) => tab.path === draggingTabPath);
                if (
                  !dragTab ||
                  dragTab.path === '/dashboard' ||
                  Boolean(dragTab.pinned) !== Boolean(item.pinned)
                ) {
                  return;
                }
                event.preventDefault();
                event.dataTransfer.dropEffect = 'move';
                if (dragOverTabPath !== item.path) {
                  setDragOverTabPath(item.path);
                }
              }}
              onDragLeave={() => {
                if (dragOverTabPath === item.path) {
                  setDragOverTabPath(null);
                }
              }}
              onDrop={(event) => {
                event.preventDefault();
                if (draggingTabPath) {
                  onMoveTab(draggingTabPath, item.path);
                }
                setDraggingTabPath(null);
                setDragOverTabPath(null);
              }}
              onDragEnd={() => {
                setDraggingTabPath(null);
                setDragOverTabPath(null);
              }}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault();
                  onNavigate(item.path);
                  return;
                }
                const offset = getTabFocusOffset(layoutMode, event.key);
                if (offset !== null) {
                  event.preventDefault();
                  moveTabFocus(offset, itemIndex);
                }
              }}
            >
              {item.pinned ? <IconPushpin className="app-shell__tab-pin" /> : null}
              <span className="app-shell__tab-label">
                {item.titleKey ? t(item.titleKey) : item.fallbackTitle}
              </span>
              {canCloseCurrent ? (
                <button
                  type="button"
                  aria-label={t('common.close')}
                  className="app-shell__tab-close"
                  onClick={(event) => {
                    event.stopPropagation();
                    onTabAction(item.path, 'close');
                  }}
                >
                  <IconClose />
                </button>
              ) : null}
            </div>
          </Dropdown>
        );
      })}
    </div>
  );
};

export default LayoutOpenedTabs;
