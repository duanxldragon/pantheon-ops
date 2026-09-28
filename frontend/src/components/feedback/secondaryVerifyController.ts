export const SECONDARY_VERIFY_CANCELLED_ERROR = 'Verification cancelled';

let verifyResolver: ((token: string) => void) | null = null;
let verifyRejecter: ((err: Error) => void) | null = null;

export const showSecondaryVerify = () =>
  new Promise<string>((resolve, reject) => {
    verifyResolver = resolve;
    verifyRejecter = reject;
    globalThis.dispatchEvent(new CustomEvent('pantheon:show-verify-modal'));
  });

export const handleVerifySuccess = (token: string) => {
  verifyResolver?.(token);
  verifyResolver = null;
  verifyRejecter = null;
};

export const handleVerifyCancel = () => {
  verifyRejecter?.(new Error(SECONDARY_VERIFY_CANCELLED_ERROR));
  verifyResolver = null;
  verifyRejecter = null;
};
