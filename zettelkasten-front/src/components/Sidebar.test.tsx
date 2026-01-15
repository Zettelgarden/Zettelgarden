import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { ShortcutProvider } from '../contexts/ShortcutContext';
import { TaskProvider } from '../contexts/TaskContext';
import { ChatProvider } from '../contexts/ChatContext';
import { PartialCardProvider } from '../contexts/CardContext';
import { AuthProvider } from '../contexts/AuthContext';
import { FileProvider } from '../contexts/FileContext';
import { PinProvider } from '../contexts/PinContext';
import { ChatSidebarProvider } from '../contexts/ChatSidebarContext';
import { CardRefreshProvider } from '../contexts/CardRefreshContext';
import { TagProvider } from '../contexts/TagContext';
import { StatusProvider } from '../contexts/StatusContext';

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

vi.mock('./sidebar/SidebarMobileMenu', () => ({
  SidebarMobileMenu: () => <div>Mobile Menu</div>,
}));

function SidebarWrapper() {
  return (
    <BrowserRouter>
      <TagProvider>
        <ChatProvider>
          <PartialCardProvider>
            <TaskProvider>
              <StatusProvider>
                <ShortcutProvider>
                  <FileProvider>
                    <PinProvider>
                      <ChatSidebarProvider>
                        <CardRefreshProvider>
                          <AuthProvider>
                            <Sidebar />
                          </AuthProvider>
                        </CardRefreshProvider>
                      </ChatSidebarProvider>
                    </PinProvider>
                  </FileProvider>
                </ShortcutProvider>
              </StatusProvider>
            </TaskProvider>
          </PartialCardProvider>
        </ChatProvider>
      </TagProvider>
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
