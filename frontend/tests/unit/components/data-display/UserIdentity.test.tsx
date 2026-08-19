import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import UserAvatarContent from '../../../../src/components/data-display/UserIdentity';
import {
  filterIdentityLabels,
  shouldShowIdentityLabel,
} from '../../../../src/components/data-display/userIdentityHelpers';

describe('UserAvatarContent', () => {
  it('falls back to the display-name initial after the image fails', () => {
    const { container } = render(
      <UserAvatarContent avatar="bad-image" userDisplayName="alice" />,
    );

    fireEvent.error(container.querySelector('img') as HTMLImageElement);

    expect(container.querySelector('img')).toBeNull();
    expect(screen.getByText('A')).not.toBeNull();
  });

  it('retries the same avatar after changing away and back', () => {
    const { container, rerender } = render(
      <UserAvatarContent avatar="avatar-a" userDisplayName="Alice" />,
    );
    fireEvent.error(container.querySelector('img') as HTMLImageElement);
    expect(container.querySelector('img')).toBeNull();

    rerender(<UserAvatarContent avatar="avatar-b" userDisplayName="Bob" />);
    expect(container.querySelector('img')?.getAttribute('src')).toBe('avatar-b');

    rerender(<UserAvatarContent avatar="avatar-a" userDisplayName="Alice" />);
    expect(container.querySelector('img')?.getAttribute('src')).toBe('avatar-a');
  });
});

describe('identity labels', () => {
  it('hides duplicate admin username and roles regardless of case', () => {
    expect(shouldShowIdentityLabel('admin', 'ADMIN')).toBe(false);
    expect(filterIdentityLabels('admin', ['admin'])).toEqual([]);
  });

  it('keeps different username and roles', () => {
    expect(shouldShowIdentityLabel('Alice', 'alice.user')).toBe(true);
    expect(filterIdentityLabels('Alice', ['admin', 'auditor'])).toEqual(['admin', 'auditor']);
  });
});
