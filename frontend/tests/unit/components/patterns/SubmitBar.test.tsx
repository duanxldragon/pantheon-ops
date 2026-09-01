import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import SubmitBar from '../../../../src/components/patterns/actions/SubmitBar';

describe('SubmitBar', () => {
  it('keeps the historical inline layout unless sticky is explicitly enabled', () => {
    const { container } = render(<SubmitBar onSubmit={() => undefined} />);

    expect(container.querySelector('[data-sticky="true"]')).toBeNull();
  });

  it('prevents duplicate submits while submitting and exposes the error summary', () => {
    const onSubmit = vi.fn();
    render(
      <SubmitBar
        sticky
        status="submitting"
        loading
        errorSummary="Correct the missing owner field"
        onSubmit={onSubmit}
      />,
    );

    const button = screen.getByRole('button');
    expect(button).toHaveProperty('disabled', true);
    expect(screen.getByRole('alert').textContent).toContain('Correct the missing owner field');
    fireEvent.click(button);
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
