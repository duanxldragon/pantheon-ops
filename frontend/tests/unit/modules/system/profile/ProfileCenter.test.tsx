import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

const { getProfileMock, stableT } = vi.hoisted(() => ({
  getProfileMock: vi.fn(),
  // A stable t identity keeps ProfileCenter's loadProfile callback (and its
  // load effect) from re-firing on every render.
  stableT: (key: string) => key,
}));

// Arco's Grid (used by ProfileCenter) registers responsive observers in jsdom.
beforeAll(() => {
  window.matchMedia =
    window.matchMedia ||
    ((query: string) =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }) as MediaQueryList);
});

vi.mock('../../../../../src/modules/system/iam/user/api', () => ({
  getProfile: (...args: unknown[]) => getProfileMock(...args),
  updateProfile: vi.fn(),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: stableT }),
}));

// Arco's imperative Message portal does not work under jsdom/React 18.
vi.mock('../../../../../src/components/feedback/message', () => ({
  message: {
    error: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
}));

import ProfileCenter from '../../../../../src/modules/system/profile/ProfileCenter';

function renderProfileCenter() {
  return render(
    <MemoryRouter initialEntries={['/profile']}>
      <ProfileCenter />
    </MemoryRouter>,
  );
}

const successProfile = {
  id: 1,
  username: 'admin',
  nickname: 'Admin',
  avatar: '',
  email: 'admin@example.com',
  phone: '',
  roles: ['admin'],
  perms: [],
  preferences: {},
  createdAt: '2026-01-01T00:00:00Z',
};

describe('ProfileCenter load failure recovery', () => {
  beforeEach(() => {
    getProfileMock.mockReset();
  });

  it('shows a persistent error state with retry and no save action when the profile fails to load', async () => {
    getProfileMock.mockRejectedValueOnce(new Error('network down'));

    const { container } = renderProfileCenter();

    await waitFor(() => {
      expect(container.querySelector('.page-result')).not.toBeNull();
    });

    // Save stays unavailable until valid data has loaded.
    expect(screen.queryByText('system.profile.saveProfile')).toBeNull();

    // The error state persists (not a transient toast): still rendered after settling.
    expect(container.querySelector('.page-result')).not.toBeNull();
  });

  it('recovers into the editable form after a successful retry', async () => {
    getProfileMock
      .mockRejectedValueOnce(new Error('network down'))
      .mockResolvedValueOnce(successProfile);

    const { container } = renderProfileCenter();

    await waitFor(() => {
      expect(container.querySelector('.page-result')).not.toBeNull();
    });

    const retryButton = container.querySelector<HTMLButtonElement>('.page-result button');
    expect(retryButton).not.toBeNull();
    fireEvent.click(retryButton as HTMLButtonElement);

    await waitFor(() => {
      expect(container.querySelector('.page-result')).toBeNull();
      expect(screen.getByText('system.profile.saveProfile')).not.toBeNull();
    });

    const usernameInput = screen.getByDisplayValue('admin');
    expect(usernameInput).toHaveProperty('disabled', true);
    expect(getProfileMock).toHaveBeenCalledTimes(2);
  });
});
