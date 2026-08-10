import { Modal } from '@arco-design/web-react';
import { appModalSizeWidthMap, mergeDialogClassName, type AppModalSize } from './AppModal.shared';

type AppModalStaticConfig = Parameters<typeof Modal.confirm>[0] & {
  size?: AppModalSize;
};

function resolveDialogStyle(config: AppModalStaticConfig) {
  return {
    ...config.style,
    width: config.style?.width ?? appModalSizeWidthMap[config.size ?? 'lg'],
  };
}

function resolveDialogWrapStyle(config: AppModalStaticConfig) {
  return {
    ...config.wrapStyle,
    zIndex: config.wrapStyle?.zIndex ?? config.style?.zIndex ?? 2500,
  };
}

export function showAppModalConfirm(config: AppModalStaticConfig) {
  return Modal.confirm({
    ...config,
    className: mergeDialogClassName('app-dialog', config.className),
    style: resolveDialogStyle(config),
    wrapStyle: resolveDialogWrapStyle(config),
  });
}

export function showAppModalSuccess(config: AppModalStaticConfig) {
  return Modal.success({
    ...config,
    className: mergeDialogClassName('app-dialog', config.className),
    style: resolveDialogStyle(config),
    wrapStyle: resolveDialogWrapStyle(config),
  });
}

export function showAppModalError(config: AppModalStaticConfig) {
  return Modal.error({
    ...config,
    className: mergeDialogClassName('app-dialog', config.className),
    style: resolveDialogStyle(config),
    wrapStyle: resolveDialogWrapStyle(config),
  });
}
