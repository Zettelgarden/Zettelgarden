import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '../tests/utils';
import { SidebarLink } from './SidebarLink';

describe('SidebarLink', () => {
  it('renders with text and icon', () => {
    renderWithProviders(
      <SidebarLink to="/test">
        <span>Icon</span>
        <span>Link Text</span>
      </SidebarLink>,
    );

    expect(screen.getByText('Icon')).toBeInTheDocument();
    expect(screen.getByText('Link Text')).toBeInTheDocument();
  });

  it('applies active class when location matches', () => {
    const { container } = renderWithProviders(
      <SidebarLink to="/">
        <span>Home</span>
      </SidebarLink>,
    );

    const link = container.querySelector('a');
    expect(link).toHaveClass('bg-gray-100');
  });

  it('renders as a link with correct href', () => {
    renderWithProviders(
      <SidebarLink to="/cards">
        <span>Cards</span>
      </SidebarLink>,
    );

    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', '/cards');
  });

  it('wraps first child in icon container', () => {
    const { container } = renderWithProviders(
      <SidebarLink to="/test">
        <svg data-testid="icon">Icon</svg>
        <span>Text</span>
      </SidebarLink>,
    );

    const iconWrapper = container.querySelector('.w-6.h-6');
    expect(iconWrapper).toBeInTheDocument();
    expect(
      iconWrapper?.querySelector('[data-testid="icon"]'),
    ).toBeInTheDocument();
  });
});
