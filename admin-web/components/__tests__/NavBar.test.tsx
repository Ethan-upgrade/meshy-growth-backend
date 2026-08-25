import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import NavBar from '../NavBar';

describe('NavBar', () => {
  it('should render the header with branding', () => {
    render(<NavBar />);
    expect(screen.getByText('Growth Admin')).toBeInTheDocument();
  });

  it('should render all navigation links', () => {
    render(<NavBar />);

    expect(screen.getByRole('link', { name: /events/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /conversions/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /experiments/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /campaigns/i })).toBeInTheDocument();
  });

  it('should have correct href for navigation links', () => {
    render(<NavBar />);

    expect(screen.getByRole('link', { name: /events/i })).toHaveAttribute('href', '/events');
    expect(screen.getByRole('link', { name: /conversions/i })).toHaveAttribute('href', '/conversions');
    expect(screen.getByRole('link', { name: /experiments/i })).toHaveAttribute('href', '/experiments');
    expect(screen.getByRole('link', { name: /campaigns/i })).toHaveAttribute('href', '/campaigns');
  });

  it('should render nav element with links inside header', () => {
    const { container } = render(<NavBar />);
    const header = container.querySelector('header');
    const nav = container.querySelector('nav');

    expect(header).toBeInTheDocument();
    expect(nav).toBeInTheDocument();
    expect(header?.contains(nav)).toBe(true);
  });
});
