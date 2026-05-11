import React from 'react';
import { Modal } from '@arco-design/web-react';
import { appModalSizeWidthMap, mergeDialogClassName, type AppModalProps } from './AppModal.shared';

const AppModal: React.FC<AppModalProps> = ({
  className,
  size = 'lg',
  style,
  maskClosable = false,
  unmountOnExit = true,
  ...rest
}) => {
  const width = style?.width ?? appModalSizeWidthMap[size];

  return (
    <Modal
      className={mergeDialogClassName('app-dialog', className)}
      style={{ ...style, width }}
      maskClosable={maskClosable}
      unmountOnExit={unmountOnExit}
      {...rest}
    />
  );
};

export default AppModal;
