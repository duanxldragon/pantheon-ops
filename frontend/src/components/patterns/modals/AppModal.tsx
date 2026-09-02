// AppModal - Shell visual contract compliance
import { Modal, type ModalProps } from '@arco-design/web-react';
import React from 'react';

export interface AppModalProps extends ModalProps {
  children?: React.ReactNode;
}

export const AppModal: React.FC<AppModalProps> = ({ wrapClassName, maskClosable = false, ...props }) => {
  return (
    <Modal
      wrapClassName={`app-dialog ${wrapClassName || ''}`}
      maskClosable={maskClosable}
      {...props}
    />
  );
};

export const showAppModalConfirm = (config: any) => {
  return Modal.confirm({
    ...config,
    wrapClassName: `app-dialog ${config?.wrapClassName || ''}`,
  });
};

export const showAppModalSuccess = (config: any) => {
  return Modal.success({
    ...config,
    wrapClassName: `app-dialog ${config?.wrapClassName || ''}`,
  });
};

export const showAppModalError = (config: any) => {
  return Modal.error({
    ...config,
    wrapClassName: `app-dialog ${config?.wrapClassName || ''}`,
  });
};

export {};

