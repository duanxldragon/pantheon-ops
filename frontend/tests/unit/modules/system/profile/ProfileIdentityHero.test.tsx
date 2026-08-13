import React from 'react';
import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import ProfileIdentityHero from '../../../../../src/modules/system/profile/ProfileIdentityHero';

describe('ProfileIdentityHero', () => {
  it('renders admin only once when nickname, username, and role are identical', () => {
    const { container } = render(
      <ProfileIdentityHero
        nickname="admin"
        username="admin"
        roles={['admin']}
        fallbackName="Profile"
      />,
    );

    expect(container.textContent?.match(/admin/gi)).toHaveLength(1);
  });

  it('keeps a distinct username and role visible', () => {
    const { container } = render(
      <ProfileIdentityHero
        nickname="Alice"
        username="alice.user"
        roles={['auditor']}
        fallbackName="Profile"
      />,
    );

    expect(container.textContent).toContain('Alice');
    expect(container.textContent).toContain('alice.user');
    expect(container.textContent).toContain('auditor');
  });
});
