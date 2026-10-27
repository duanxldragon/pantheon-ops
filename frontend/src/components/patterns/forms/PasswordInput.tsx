import { useState } from 'react';
import { Input } from '@arco-design/web-react';
import { IconEye, IconEyeInvisible } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import './password-input.css';

// Arco's own toggle is a focusable <svg> with no accessible name, so every
// password field has to supply a real button instead.
type PasswordInputProps = Omit<
  React.ComponentProps<typeof Input.Password>,
  'suffix' | 'visibility' | 'visibilityToggle' | 'defaultVisibility' | 'onVisibilityChange'
>;

export default function PasswordInput(props: Readonly<PasswordInputProps>) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);

  return (
    <Input.Password
      {...props}
      visibilityToggle={false}
      visibility={visible}
      suffix={
        <button
          type="button"
          className="password-input__toggle"
          disabled={props.disabled}
          aria-label={t(visible ? 'auth.password.hide' : 'auth.password.show')}
          // Arco's input wrapper refocuses the <input> on any bubbled click,
          // which would strand keyboard users after a single activation.
          onClick={(event) => {
            event.stopPropagation();
            setVisible((next) => !next);
          }}
        >
          {visible ? <IconEye /> : <IconEyeInvisible />}
        </button>
      }
    />
  );
}
