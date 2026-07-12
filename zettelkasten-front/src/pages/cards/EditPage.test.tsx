import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { EditPage } from './EditPage';
import { UIStateProvider, useUIState } from '../../contexts/UIStateContext';
import { TagProvider } from '../../contexts/TagContext';

// Templates are the only network call made on mount for a new card.
vi.mock('../../api/templates', () => ({
  getTemplates: vi.fn(() => Promise.resolve([])),
}));

// Card body + its heavy sub-deps are irrelevant to the rail layout; stub them
// so the test stays focused and fast. FileListItem pulls in react-pdf which
// needs DOMMatrix (absent in happy-dom).
vi.mock('../../components/cards/CardBodyTextArea', () => ({
  CardBodyTextArea: () => <textarea data-testid="card-body" />,
  CardBodyTextAreaHandle: {},
}));

vi.mock('../../components/files/FileListItem', () => ({
  FileListItem: () => <div data-testid="file-list-item">file</div>,
}));

function renderEditPage() {
  return render(
    <BrowserRouter>
      <TagProvider testing testTags={[]}>
        <UIStateProvider>
          <EditPage newCard={true} />
        </UIStateProvider>
      </TagProvider>
    </BrowserRouter>,
  );
}

describe('EditPage — closable rail', () => {
  beforeEach(() => {
    // The hook writes ?pane= to the URL; reset between tests.
    window.history.replaceState({}, '', '/');
  });

  it('renders the Metadata rail by default', () => {
    renderEditPage();
    expect(screen.getByText('Metadata')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Close info pane' })).toBeInTheDocument();
  });

  it('hides the rail and reclaims full width when the pane is closed', () => {
    function Harness() {
      const { rightPaneOpen } = useUIState();
      return (
        <>
          <EditPage newCard={true} />
          <div data-testid="rail-state">{rightPaneOpen ? 'open' : 'closed'}</div>
        </>
      );
    }
    render(
      <BrowserRouter>
        <TagProvider testing testTags={[]}>
          <UIStateProvider>
            <Harness />
          </UIStateProvider>
        </TagProvider>
      </BrowserRouter>,
    );

    expect(screen.getByTestId('rail-state').textContent).toBe('open');
    expect(screen.getByText('Metadata')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Close info pane' }));

    expect(screen.getByTestId('rail-state').textContent).toBe('closed');
    // The rail tab strip + close affordance disappear with the panel.
    expect(screen.queryByText('Metadata')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Close info pane' })).not.toBeInTheDocument();
  });
});
