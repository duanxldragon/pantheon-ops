import type { ModalProps } from '@arco-design/web-react';
import type React from 'react';

export type AppModalSize = 'sm' | 'md' | 'lg' | 'xl' | 'detail';

export interface AppModalProps extends ModalProps {
  children?: React.ReactNode;
  size?: AppModalSize;
}

export const appModalSizeWidthMap: Record<AppModalSize, number> = {
  sm: 560,
  md: 640,
  lg: 760,
  xl: 920,
  detail: 880,
};

export function mergeDialogClassName(base: string, className?: string | string[]) {
  if (Array.isArray(className)) {
    return [base, ...className];
  }
  return className ? `${base} ${className}` : base;
}
