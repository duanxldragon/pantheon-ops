import {
  IconApps,
  IconBook,
  IconBranch,
  IconClockCircle,
  IconCloud,
  IconCode,
  IconDashboard,
  IconDesktop,
  IconFile,
  IconIdcard,
  IconLanguage,
  IconList,
  IconLock,
  IconMenu,
  IconSafe,
  IconSettings,
  IconStorage,
  IconTags,
  IconTool,
  IconUser,
  IconUserGroup,
} from '@arco-design/web-react/icon';

export type MenuIconKey =
  | 'dashboard'
  | 'user'
  | 'user-group'
  | 'idcard'
  | 'safe'
  | 'lock'
  | 'clock'
  | 'desktop'
  | 'cloud'
  | 'file'
  | 'menu'
  | 'settings'
  | 'tool'
  | 'list'
  | 'book'
  | 'language'
  | 'code'
  | 'apps'
  | 'storage'
  | 'branch'
  | 'tags';

export const MENU_ICON_OPTIONS: Array<{ value: MenuIconKey; labelKey: string }> = [
  { value: 'dashboard', labelKey: 'system.menu.icon.dashboard' },
  { value: 'user', labelKey: 'system.menu.icon.user' },
  { value: 'user-group', labelKey: 'system.menu.icon.userGroup' },
  { value: 'idcard', labelKey: 'system.menu.icon.idcard' },
  { value: 'safe', labelKey: 'system.menu.icon.safe' },
  { value: 'lock', labelKey: 'system.menu.icon.lock' },
  { value: 'clock', labelKey: 'system.menu.icon.clock' },
  { value: 'desktop', labelKey: 'system.menu.icon.desktop' },
  { value: 'cloud', labelKey: 'system.menu.icon.cloud' },
  { value: 'file', labelKey: 'system.menu.icon.file' },
  { value: 'menu', labelKey: 'system.menu.icon.menu' },
  { value: 'settings', labelKey: 'system.menu.icon.settings' },
  { value: 'tool', labelKey: 'system.menu.icon.tool' },
  { value: 'list', labelKey: 'system.menu.icon.list' },
  { value: 'book', labelKey: 'system.menu.icon.book' },
  { value: 'language', labelKey: 'system.menu.icon.language' },
  { value: 'code', labelKey: 'system.menu.icon.code' },
  { value: 'apps', labelKey: 'system.menu.icon.apps' },
  { value: 'storage', labelKey: 'system.menu.icon.storage' },
  { value: 'branch', labelKey: 'system.menu.icon.branch' },
  { value: 'tags', labelKey: 'system.menu.icon.tags' },
];

export function renderMenuIcon(icon?: string) {
  switch ((icon || '').trim().toLowerCase()) {
    case 'dashboard':
      return <IconDashboard />;
    case 'user':
      return <IconUser />;
    case 'user-group':
      return <IconUserGroup />;
    case 'idcard':
      return <IconIdcard />;
    case 'safe':
      return <IconSafe />;
    case 'lock':
    case 'permission':
      return <IconLock />;
    case 'clock':
    case 'log':
      return <IconClockCircle />;
    case 'desktop':
    case 'session':
      return <IconDesktop />;
    case 'cloud':
    case 'host':
      return <IconCloud />;
    case 'file':
    case 'audit':
      return <IconFile />;
    case 'settings':
      return <IconSettings />;
    case 'tool':
    case 'config':
      return <IconTool />;
    case 'list':
      return <IconList />;
    case 'book':
    case 'dict':
      return <IconBook />;
    case 'language':
      return <IconLanguage />;
    case 'code':
      return <IconCode />;
    case 'apps':
      return <IconApps />;
    case 'storage':
      return <IconStorage />;
    case 'branch':
    case 'dept':
      return <IconBranch />;
    case 'tags':
    case 'post':
      return <IconTags />;
    case 'menu':
    default:
      return <IconMenu />;
  }
}
