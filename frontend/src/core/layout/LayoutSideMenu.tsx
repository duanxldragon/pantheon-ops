import React from 'react';
import { Layout, Menu, Spin } from '@arco-design/web-react';

const { Sider } = Layout;

interface LayoutSideMenuProps {
  appName: string;
  brandInitial: string;
  collapsed: boolean;
  isHorizontalLayout: boolean;
  loading: boolean;
  menuOpenKeys: string[];
  renderedMenuItems: React.ReactNode;
  selectedMenuPath: string;
  showExpandedBrand: boolean;
  siteLogo?: string;
  onCollapse: (collapsed: boolean) => void;
  onMenuItemClick: (key: string) => void;
}

const LayoutSideMenu: React.FC<LayoutSideMenuProps> = ({
  appName,
  brandInitial,
  collapsed,
  isHorizontalLayout,
  loading,
  menuOpenKeys,
  renderedMenuItems,
  selectedMenuPath,
  showExpandedBrand,
  siteLogo,
  onCollapse,
  onMenuItemClick,
}) => {
  if (isHorizontalLayout) {
    return (
      <div className="app-shell__top-nav">
        <Spin loading={loading} className="app-shell__menu-loading">
          <Menu
            mode="horizontal"
            className="app-shell__top-menu"
            selectedKeys={[selectedMenuPath]}
            triggerProps={{ className: 'app-shell__top-menu-popup' }}
            onClickMenuItem={onMenuItemClick}
          >
            {renderedMenuItems}
          </Menu>
        </Spin>
      </div>
    );
  }

  return (
    <Sider
      className="app-shell__sider"
      theme="light"
      trigger={null}
      width={248}
      collapsedWidth={76}
      collapsed={collapsed}
      collapsible
      breakpoint="xl"
      onCollapse={onCollapse}
    >
      <div
        className={collapsed ? 'app-shell__brand app-shell__brand--collapsed' : 'app-shell__brand'}
      >
        <div className="app-shell__brand-mark">
          {siteLogo ? <img src={siteLogo} alt={appName} /> : brandInitial}
        </div>
        {showExpandedBrand ? (
          <div className="app-shell__brand-text">
            <span className="app-shell__brand-title">{appName}</span>
          </div>
        ) : null}
      </div>
      <Spin loading={loading} className="app-shell__menu-loading">
        <Menu
          key={`${collapsed ? 'collapsed' : 'expanded'}-${menuOpenKeys.join(',')}`}
          className="app-shell__menu"
          theme="light"
          selectedKeys={[selectedMenuPath]}
          defaultOpenKeys={menuOpenKeys}
          onClickMenuItem={onMenuItemClick}
        >
          {renderedMenuItems}
        </Menu>
      </Spin>
    </Sider>
  );
};

export default LayoutSideMenu;
