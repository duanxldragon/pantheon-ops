import { Message } from '@arco-design/web-react';

type MessageContent = string | React.ReactNode;
type MessageConfig = { duration?: number; position?: 'top' | 'bottom' };

const defaults: MessageConfig = {
  duration: 3000,
  position: 'top',
};

function resolve(content: MessageContent, opts?: MessageConfig) {
  return { ...defaults, content, ...opts };
}

export const message = {
  success(content: MessageContent, opts?: MessageConfig) {
    return Message.success(resolve(content, opts));
  },
  error(content: MessageContent, opts?: MessageConfig) {
    return Message.error(resolve(content, opts));
  },
  warning(content: MessageContent, opts?: MessageConfig) {
    return Message.warning(resolve(content, opts));
  },
  info(content: MessageContent, opts?: MessageConfig) {
    return Message.info(resolve(content, opts));
  },
  loading(content: MessageContent, opts?: MessageConfig) {
    return Message.loading(resolve(content, opts));
  },
  clear() {
    return Message.clear();
  },
};
