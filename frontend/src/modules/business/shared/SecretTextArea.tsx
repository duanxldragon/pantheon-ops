import { useState } from 'react';
import { Input } from '@arco-design/web-react';
import { IconEye, IconEyeInvisible } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import './secret-text-area.css';

// Multi-line counterpart of PasswordInput: kubeconfigs and PEM keys need
// newlines, so masking uses -webkit-text-security instead of a single-line
// password input.
type SecretTextAreaProps = React.ComponentProps<typeof Input.TextArea>;

export default function SecretTextArea({
  className,
  disabled,
  ...props
}: Readonly<SecretTextAreaProps>) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);

  return (
    <div className="secret-text-area">
      <Input.TextArea
        {...props}
        disabled={disabled}
        className={[visible ? '' : 'secret-text-area__masked', className].filter(Boolean).join(' ')}
      />
      <button
        type="button"
        className="password-input__toggle secret-text-area__toggle"
        disabled={disabled}
        aria-label={t(visible ? 'auth.password.hide' : 'auth.password.show')}
        onClick={() => setVisible((next) => !next)}
      >
        {visible ? <IconEye /> : <IconEyeInvisible />}
      </button>
    </div>
  );
}
