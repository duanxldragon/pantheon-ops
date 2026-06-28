export interface UserPlatformPreferences {
  theme?: 'indigo' | 'emerald' | 'violet' | 'slate';
  language?: 'zh-CN' | 'en-US' | 'ja-JP' | 'ko-KR' | 'fr-FR';
  layoutMode?: 'vertical' | 'horizontal';
  densityMode?: 'comfortable' | 'compact';
}

export interface UserInfo {
  id: number;
  username: string;
  nickname: string;
  avatar?: string;
  email?: string;
  phone?: string;
  roles?: string[];
  perms?: string[];
  preferences?: UserPlatformPreferences;
}
