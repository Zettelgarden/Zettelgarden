import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ViewPageSidePanels } from './ViewPageSidePanels';
import { UIStateProvider, useUIState } from '../../contexts/UIStateContext';
import { sampleCards, sampleEntityData, samplePartialCards } from '../../tests/data';

// Heavy children are mocked to keep this a focused unit test and avoid
// pulling TagContext / react-pdf transitively.
vi.mock('./CardList', () => ({
  CardList: ({ cards }: any) => <div data-testid="card-list">{cards.length}</div>,
}));
vi.mock('./ChildrenCards', () => ({
  ChildrenCards: ({ allChildren }: any) => <div data-testid="children-cards">{allChildren.length}</div>,
}));
vi.mock('./BacklinkInput', () => ({
  BacklinkInput: () => <div data-testid="backlink-input">Add Backlink</div>,
}));

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
    onCreateChildCard: vi.fn(),
    categorizedReferences: { bidirectional: [], incoming: [], outgoing: [] },
    onAddBacklink: vi.fn(),
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
  it('defaults to the Links tab and shows the Children section', () => {
    renderPanel();
    // Links is the default; its Children header is visible.
    expect(screen.getByText('Children')).toBeInTheDocument();
    // Metadata-only content is absent on the default tab.
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });

  it('switches to the Links tab and shows the Parent section', () => {
    renderPanel({ parentCard });
    fireEvent.click(screen.getByText('Links'));
    expect(screen.getByText('Parent')).toBeInTheDocument();
    // Metadata content is now hidden
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });

  it('renders the Children + References structure on the Links tab without a parent', () => {
    renderPanel({ parentCard: null });
    // No parent, but the Links tab still offers Children + references.
    expect(screen.getByText('Children')).toBeInTheDocument();
    expect(screen.getByText('Linked references')).toBeInTheDocument();
    // Parent header is absent when there is no parent.
    expect(screen.queryByText('Parent')).not.toBeInTheDocument();
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

  it('shows Children, Linked references, and the backlink input on the Links tab', () => {
    const [ref] = samplePartialCards();
    renderPanel({
      categorizedReferences: { bidirectional: [ref], incoming: [], outgoing: [] },
    });
    // Default tab is Links.
    expect(screen.getByText('Children')).toBeInTheDocument();
    expect(screen.getByText('Linked references')).toBeInTheDocument();
    // With a reference present the collapsible is open, revealing the input.
    expect(screen.getByTestId('backlink-input')).toBeInTheDocument();
  });

  it('calls onCreateChildCard from the Links tab add-child button', () => {
    const onCreateChildCard = vi.fn();
    renderPanel({ onCreateChildCard });
    fireEvent.click(screen.getByTitle('Add child'));
    expect(onCreateChildCard).toHaveBeenCalledTimes(1);
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
