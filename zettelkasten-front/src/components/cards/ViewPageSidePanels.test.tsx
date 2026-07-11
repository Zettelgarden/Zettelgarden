import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ViewPageSidePanels } from './ViewPageSidePanels';
import { UIStateProvider, useUIState } from '../../contexts/UIStateContext';
import { sampleCards, sampleEntityData } from '../../tests/data';

type PanelProps = React.ComponentProps<typeof ViewPageSidePanels>;
const [viewingCard, parentCard] = sampleCards();

function baseProps(overrides: Partial<PanelProps> = {}): PanelProps {
  return {
    parentCard: null,
    prevSibling: null,
    nextSibling: null,
    linkedEntities: [],
    onOpenEntity: vi.fn(),
    viewingCard,
    tags: [],
    onTagClick: vi.fn(),
    onRemoveTag: vi.fn(),
    ...overrides,
  };
}

function renderPanel(overrides: Partial<PanelProps> = {}) {
  return render(
    <BrowserRouter>
      <UIStateProvider>
        <ViewPageSidePanels {...baseProps(overrides)} />
      </UIStateProvider>
    </BrowserRouter>,
  );
}

describe('ViewPageSidePanels — tabs', () => {
  it('defaults to the Metadata tab and shows Tags + Details', () => {
    renderPanel();
    expect(screen.getByText('Tags')).toBeInTheDocument();
    expect(screen.getByText('Created:')).toBeInTheDocument();
    // Links-only content (Parent) is absent on the default tab
    expect(screen.queryByText('Parent')).not.toBeInTheDocument();
  });

  it('switches to the Links tab and shows the Parent section', () => {
    renderPanel({ parentCard });
    fireEvent.click(screen.getByText('Links'));
    expect(screen.getByText('Parent')).toBeInTheDocument();
    // Metadata content is now hidden
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });

  it('shows a calm hint on the Links tab when there is no parent', () => {
    renderPanel({ parentCard: null });
    fireEvent.click(screen.getByText('Links'));
    expect(screen.getByText('No links for this card yet.')).toBeInTheDocument();
  });

  it('switches to the Entities tab and shows linked entities', () => {
    renderPanel({ linkedEntities: sampleEntityData });
    fireEvent.click(screen.getByText('Entities'));
    expect(screen.getByText('Entity One')).toBeInTheDocument();
    // Metadata content is hidden
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });

  it('shows a calm hint on the Entities tab when empty', () => {
    renderPanel({ linkedEntities: [] });
    fireEvent.click(screen.getByText('Entities'));
    expect(screen.getByText('No linked entities.')).toBeInTheDocument();
  });

  it('renders a close button that collapses the rail via context', () => {
    function Harness() {
      const { rightPaneOpen } = useUIState();
      return (
        <>
          <ViewPageSidePanels {...baseProps()} />
          <div data-testid="rail-state">{rightPaneOpen ? 'open' : 'closed'}</div>
        </>
      );
    }
    render(
      <BrowserRouter>
        <UIStateProvider>
          <Harness />
        </UIStateProvider>
      </BrowserRouter>,
    );
    expect(screen.getByTestId('rail-state').textContent).toBe('open');
    fireEvent.click(screen.getByRole('button', { name: 'Close info pane' }));
    expect(screen.getByTestId('rail-state').textContent).toBe('closed');
  });
});
