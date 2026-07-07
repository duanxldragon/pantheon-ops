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

export function showAppModalConfirm(config: AppModalStaticConfig) {
  return Modal.confirm({
    ...config,
    className: mergeDialogClassName('app-dialog', config.className),
    style: resolveDialogStyle(config),
  });
}

export function showAppModalSuccess(config: AppModalStaticConfig) {
  return Modal.success({
    ...config,
    className: mergeDialogClassName('app-dialog', config.className),
    style: resolveDialogStyle(config),
  });
}

export function showAppModalError(config: AppModalStaticConfig) {
  return Modal.error({
    ...config,
    className: mergeDialogClassName('app-dialog', config.className),
    style: resolveDialogStyle(config),
  });
}
