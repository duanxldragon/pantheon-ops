import React from 'react';
import { Modal } from '@arco-design/web-react';
import { appModalSizeWidthMap, mergeDialogClassName, type AppModalProps } from './AppModal.shared';

const AppModal: React.FC<AppModalProps> = ({
  className,
  size = 'lg',
  style,
  wrapStyle,
  maskClosable = false,
  unmountOnExit = true,
  ...rest
}) => {
  const width = style?.width ?? appModalSizeWidthMap[size];
  const overlayZIndex = wrapStyle?.zIndex ?? style?.zIndex ?? 2500;

  return (
    <Modal
      className={mergeDialogClassName('app-dialog', className)}
      style={{ ...style, width }}
      wrapStyle={{ ...wrapStyle, zIndex: overlayZIndex }}
      maskClosable={maskClosable}
      unmountOnExit={unmountOnExit}
      {...rest}
    />
  );
};

export default AppModal;
