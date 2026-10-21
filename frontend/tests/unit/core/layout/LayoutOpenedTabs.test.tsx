import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

// Importing the real prefetch module pulls the whole module registry (and its
// network warmers) into the test; the tabs component only calls the preload fn.
vi.mock('../../../../src/core/router/prefetch', () => ({
  preloadRouteComponent: vi.fn(() => Promise.resolve()),
}));

import LayoutOpenedTabs from '../../../../src/core/layout/LayoutOpenedTabs';
import type { OpenedPageTab } from '../../../../src/core/layout/layoutTabs';

const tabs: OpenedPageTab[] = [
  { path: '/dashboard', fallbackTitle: 'Dashboard', closable: false, pinned: true },
  { path: '/system/user', fallbackTitle: 'Users', closable: true },
  { path: '/system/role', fallbackTitle: 'Roles', closable: true },
];

const baseProps = {
  enabled: true,
  tabs,
  currentPath: '/system/user',
  onNavigate: vi.fn(),
  onMoveTab: vi.fn(),
  onTabAction: vi.fn(),
  t: (key: string) => key,
};

function renderTabs(layoutMode: 'horizontal' | 'vertical' = 'horizontal') {
  return render(<LayoutOpenedTabs {...baseProps} layoutMode={layoutMode} />);
}

describe('LayoutOpenedTabs keyboard model', () => {
  it('activates the focused tab with Enter and Space', () => {
    const onNavigate = vi.fn();
    render(
      <LayoutOpenedTabs {...baseProps} layoutMode="horizontal" onNavigate={onNavigate} />,
    );

    const tab = screen.getByRole('tab', { selected: true });
    expect(tab.textContent).toContain('Users');

    fireEvent.keyDown(tab, { key: 'Enter' });
    expect(onNavigate).toHaveBeenCalledWith('/system/user');

    fireEvent.keyDown(tab, { key: ' ' });
    expect(onNavigate).toHaveBeenCalledTimes(2);
  });

  it('moves focus with arrow keys in horizontal mode and wraps at the edges', () => {
    const { container } = renderTabs('horizontal');
    const tabElements = Array.from(
      container.querySelectorAll<HTMLDivElement>('[role="tab"]'),
    );

    fireEvent.keyDown(tabElements[1], { key: 'ArrowRight' });
    expect(document.activeElement).toBe(tabElements[2]);

    fireEvent.keyDown(tabElements[2], { key: 'ArrowRight' });
    expect(document.activeElement).toBe(tabElements[0]);

    fireEvent.keyDown(tabElements[0], { key: 'ArrowLeft' });
    expect(document.activeElement).toBe(tabElements[2]);

    fireEvent.keyDown(tabElements[2], { key: 'ArrowLeft' });
    expect(document.activeElement).toBe(tabElements[1]);
  });

  it('moves focus with Up/Down in vertical mode and supports Home/End', () => {
    const { container } = renderTabs('vertical');
    const tabElements = Array.from(
      container.querySelectorAll<HTMLDivElement>('[role="tab"]'),
    );

    fireEvent.keyDown(tabElements[1], { key: 'ArrowDown' });
    expect(document.activeElement).toBe(tabElements[2]);

    fireEvent.keyDown(tabElements[1], { key: 'Home' });
    expect(document.activeElement).toBe(tabElements[0]);

    fireEvent.keyDown(tabElements[0], { key: 'End' });
    expect(document.activeElement).toBe(tabElements[2]);
  });

  it('exposes the tablist semantics for the opened tabs', () => {
    renderTabs();
    const list = screen.getByRole('tablist');
    expect(list.getAttribute('aria-label')).toBe('app.openedTabs');
    expect(screen.getAllByRole('tab')).toHaveLength(3);
  });
});
