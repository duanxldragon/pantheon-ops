import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => `label:${key}` }),
}));

import PasswordInput from '../../../../../src/components/patterns/forms/PasswordInput';

function field() {
  return document.querySelector<HTMLInputElement>('#pw_input')!;
}

describe('PasswordInput', () => {
  it("exposes the visibility toggle as a named button instead of Arco's unnamed svg", () => {
    render(<PasswordInput id="pw_input" defaultValue="hunter2" />);

    const toggle = screen.getByRole('button', { name: 'label:auth.password.show' });
    expect(toggle).toHaveProperty('type', 'button');
    expect(field().type).toBe('password');
  });

  it('flips the input type and keeps the typed value in both states', () => {
    render(<PasswordInput id="pw_input" defaultValue="hunter2" />);

    fireEvent.click(screen.getByRole('button', { name: 'label:auth.password.show' }));
    expect(field().type).toBe('text');
    expect(field().value).toBe('hunter2');
    expect(screen.getByRole('button', { name: 'label:auth.password.hide' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'label:auth.password.hide' }));
    expect(field().type).toBe('password');
    expect(field().value).toBe('hunter2');
  });

  // Arco's input wrapper refocuses the <input> on any click that bubbles to it,
  // which stranded keyboard users after a single activation.
  it('keeps focus on the toggle after activating it', () => {
    render(<PasswordInput id="pw_input" />);

    const toggle = screen.getByRole('button', { name: 'label:auth.password.show' });
    toggle.focus();
    fireEvent.click(toggle);

    expect(document.activeElement).toBe(toggle);
    expect(screen.getByRole('button', { name: 'label:auth.password.hide' })).toBeTruthy();
  });

  it('does not submit the surrounding form when toggled', () => {
    const onSubmit = vi.fn();
    render(
      <form onSubmit={onSubmit}>
        <PasswordInput id="pw_input" />
      </form>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'label:auth.password.show' }));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('disables the toggle with the field so no dead control stays focusable', () => {
    render(<PasswordInput id="pw_input" disabled />);

    expect(screen.getByRole('button', { name: 'label:auth.password.show' })).toHaveProperty(
      'disabled',
      true,
    );
  });
});
