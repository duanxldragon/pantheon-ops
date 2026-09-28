import { defineModule } from '../../../core/router/types';

export const ProfileModule = defineModule({
  name: 'profile',
  scope: 'system',
  routes: [
    {
      path: 'system/profile',
      routeName: 'system-profile',
      titleKey: 'system.profile.title',
      icon: 'user',
      componentKey: 'system/profile/ProfileCenter',
    },
  ],
  menus: [],
  permissions: [],
  i18nNamespaces: ['system.profile'],
});
