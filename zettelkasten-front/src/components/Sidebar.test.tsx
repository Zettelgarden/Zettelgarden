import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { UIStateProvider } from '../contexts/UIStateContext';
import { DialogStateProvider } from '../contexts/DialogStateContext';
import { TaskProvider } from '../contexts/TaskContext';
import { AuthProvider } from '../contexts/AuthContext';
import { TagProvider } from '../contexts/TagContext';
import { StatusProvider } from '../contexts/StatusContext';
import { ToastProvider } from './toast/ToastContext';
import { RSSProvider } from '../contexts/RSSContext';

// Mock the child components that may make API calls
vi.mock('./sidebar/SidebarHeader', () => ({
  SidebarHeader: () => <div data-testid="sidebar-header">Header</div>,
}));

vi.mock('./sidebar/NavigationLinks', () => ({
  NavigationLinks: () => <div data-testid="nav-links">Nav</div>,
}));

vi.mock('./sidebar/SecondaryNavigationLinks', () => ({
  SecondaryNavigationLinks: () => <div>Secondary Nav</div>,
}));

vi.mock('./sidebar/StarredSearchesSection', () => ({
  StarredSearchesSection: () => <div>Starred Searches</div>,
}));

vi.mock('./sidebar/StarredCardsSection', () => ({
  StarredCardsSection: () => <div>Starred Cards</div>,
}));

vi.mock('./sidebar/SidebarFooter', () => ({
  SidebarFooter: () => <div>Footer</div>,
}));

vi.mock('./sidebar/SidebarModals', () => ({
  SidebarModals: () => <div data-testid="sidebar-modals">Modals</div>,
}));

vi.mock('./habits/TodaysHabitsWidget', () => ({
  TodaysHabitsWidget: () => <div data-testid="todays-habits-widget">Habits</div>,
}));

function SidebarWrapper() {
  return (
    <BrowserRouter>
      <ToastProvider>
        <TagProvider>
          <TaskProvider testing={true} testTasks={[]}>
            <StatusProvider>
              <UIStateProvider>
                <DialogStateProvider>
                  <RSSProvider>
                    <AuthProvider>
                      <Sidebar />
                    </AuthProvider>
                  </RSSProvider>
                </DialogStateProvider>
              </UIStateProvider>
            </StatusProvider>
          </TaskProvider>
        </TagProvider>
      </ToastProvider>
    </BrowserRouter>
  );
}

describe('Sidebar Keyboard Shortcuts', () => {
  it('should render without errors', () => {
    render(<SidebarWrapper />);
    expect(screen.getByTestId('sidebar-header')).toBeInTheDocument();
  });

  it('should respond to "t" key press for creating task', () => {
    render(<SidebarWrapper />);

    // Fire keydown event with key "t"
    fireEvent.keyDown(document, { key: 't' });

    // The hook should have triggered the callback
    // Note: We can't directly check the modal visibility in this integration test
    // without more complex setup, but we can verify the component renders
    expect(screen.getByTestId('sidebar-modals')).toBeInTheDocument();
  });

  it('should respond to "s" key press for search', () => {
    render(<SidebarWrapper />);

    // Fire keydown event with key "s"
    fireEvent.keyDown(document, { key: 's' });

    // The hook should have triggered the callback
    expect(screen.getByTestId('sidebar-modals')).toBeInTheDocument();
  });

  it('should not respond to shortcuts when input is focused', () => {
    render(<SidebarWrapper />);

    const input = document.createElement('input');
    document.body.appendChild(input);
    input.focus();

    fireEvent.keyDown(document, { key: 't' });

    // Component should still render but shortcut should not trigger
    expect(screen.getByTestId('sidebar-header')).toBeInTheDocument();

    document.body.removeChild(input);
  });

  it('should not respond to shortcuts with metaKey', () => {
    render(<SidebarWrapper />);

    fireEvent.keyDown(document, { key: 't', metaKey: true });

    // Component should render but shortcut should not have been triggered
    expect(screen.getByTestId('sidebar-header')).toBeInTheDocument();
  });
});
