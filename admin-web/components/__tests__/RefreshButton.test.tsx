import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import RefreshButton from '../RefreshButton';

// Mock next/navigation
const mockRefresh = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    refresh: mockRefresh,
  }),
}));

describe('RefreshButton', () => {
  beforeEach(() => {
    mockRefresh.mockClear();
  });

  it('should render a button with text "Refresh"', () => {
    render(<RefreshButton />);
    const button = screen.getByRole('button', { name: /refresh/i });
    expect(button).toBeInTheDocument();
  });

  it('should call router.refresh when clicked', async () => {
    const user = userEvent.setup();

    render(<RefreshButton />);
    const button = screen.getByRole('button', { name: /refresh/i });

    await user.click(button);

    expect(mockRefresh).toHaveBeenCalled();
  });

  it('should have proper styling classes', () => {
    render(<RefreshButton />);
    const button = screen.getByRole('button', { name: /refresh/i });

    expect(button).toHaveClass(
      'rounded',
      'border',
      'border-slate-300',
      'px-3',
      'py-1',
      'text-sm',
      'hover:bg-slate-100'
    );
  });
});
