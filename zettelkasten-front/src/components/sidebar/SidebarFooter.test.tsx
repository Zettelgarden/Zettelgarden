import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { SidebarFooter } from './SidebarFooter';
import { useAuth } from '../../contexts/AuthContext';

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

const mockUseAuth = vi.mocked(useAuth);

function renderFooter(isCollapsed = false) {
  return render(
    <MemoryRouter>
      <SidebarFooter isCollapsed={isCollapsed} onToggleCollapse={() => {}} />
    </MemoryRouter>,
  );
}

describe('SidebarFooter admin link', () => {
  beforeEach(() => {
    mockUseAuth.mockReset();
    mockUseAuth.mockReturnValue({ isAdmin: false } as any);
  });

  it('hides the Admin link for non-admin users', () => {
    renderFooter();
    expect(screen.queryByLabelText('Admin')).not.toBeInTheDocument();
    // Regular links still render.
    expect(screen.getByLabelText('Help')).toBeInTheDocument();
    expect(screen.getByLabelText('Settings')).toBeInTheDocument();
  });

  it('shows the Admin link for admins', () => {
    mockUseAuth.mockReturnValue({ isAdmin: true } as any);
    renderFooter();
    const adminLink = screen.getByLabelText('Admin');
    expect(adminLink).toBeInTheDocument();
    expect(adminLink.getAttribute('href')).toBe('/admin');
  });

  it('shows the Admin link in collapsed mode too', () => {
    mockUseAuth.mockReturnValue({ isAdmin: true } as any);
    renderFooter(true);
    expect(screen.getByLabelText('Admin')).toBeInTheDocument();
  });
});
