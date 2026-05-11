import React from 'react';
import { Button, Dropdown, Menu } from '@arco-design/web-react';
import { IconPalette } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { usePantheonTheme, type PantheonThemeKey } from './theme';

interface ThemeSwitcherProps {
  className?: string;
  showLabel?: boolean;
}

const ThemeSwitcher: React.FC<ThemeSwitcherProps> = ({ className, showLabel = true }) => {
  const { t } = useTranslation();
  const { theme, setTheme, options } = usePantheonTheme();
  const activeTheme = options.find((item) => item.key === theme) ?? options[0];

  return (
    <Dropdown
      position="br"
      droplist={
        <Menu
          selectedKeys={[theme]}
          onClickMenuItem={(key) => setTheme(key as PantheonThemeKey)}
          className="pantheon-theme-menu"
        >
          {options.map((item) => (
            <Menu.Item key={item.key}>
              <span className="pantheon-theme-menu__item">
                <span className="pantheon-theme-menu__swatch" style={{ background: item.accent }} />
                <span className="pantheon-theme-menu__copy">
                  <span className="pantheon-theme-menu__title">{t(item.labelKey)}</span>
                  <span className="pantheon-theme-menu__desc">{t(item.descriptionKey)}</span>
                </span>
              </span>
            </Menu.Item>
          ))}
        </Menu>
      }
    >
      <Button type="text" className={className} icon={<IconPalette />}>
        {showLabel ? t(activeTheme.labelKey) : null}
      </Button>
    </Dropdown>
  );
};

export default ThemeSwitcher;
