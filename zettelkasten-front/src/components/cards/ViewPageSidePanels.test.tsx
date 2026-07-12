import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ViewPageSidePanels } from './ViewPageSidePanels';
import { UIStateProvider, useUIState } from '../../contexts/UIStateContext';
import { sampleCards, samplePartialCards } from '../../tests/data';

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
vi.mock('../tabs/EntitiesTab', () => ({
  EntitiesTab: ({ viewingCard }: any) => (
    <div data-testid="entities-tab">
      {viewingCard.entities?.map((e: any) => (
        <span key={e.id}>{e.name}</span>
      ))}
      {(!viewingCard.entities || viewingCard.entities.length === 0) && (
        <span>No entities available</span>
      )}
    </div>
  ),
}));
vi.mock('../tabs/FilesTab', () => ({
  FilesTab: ({ viewingCard }: any) => (
    <div data-testid="files-tab">{viewingCard.files.length} files</div>
  ),
}));
vi.mock('../tabs/HistoryTab', () => ({
  HistoryTab: ({ auditEvents }: any) => (
    <div data-testid="history-tab">{auditEvents.length} events</div>
  ),
}));
vi.mock('../tabs/RollbackConfirmDialog', () => ({
  RollbackConfirmDialog: () => null,
}));

type PanelProps = React.ComponentProps<typeof ViewPageSidePanels>;
const [viewingCard, parentCard] = sampleCards();

function baseProps(overrides: Partial<PanelProps> = {}): PanelProps {
  return {
    parentCard: null,
    prevSibling: null,
    nextSibling: null,
    onOpenEntity: vi.fn(),
    viewingCard,
    tags: [],
    onTagClick: vi.fn(),
    onRemoveTag: vi.fn(),
    onCreateChildCard: vi.fn(),
    categorizedReferences: { bidirectional: [], incoming: [], outgoing: [] },
    onAddBacklink: vi.fn(),
    setViewCard: vi.fn(),
    setError: vi.fn(),
    fileUploadRef: { current: null } as React.RefObject<HTMLInputElement>,
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
  beforeEach(() => {
    // URL-param tests mutate the location; reset between tests.
    window.history.replaceState({}, '', '/');
  });
  it('defaults to the Metadata tab when the card has no relationships', () => {
    renderPanel();
    // No children/references → Metadata is the smart default.
    expect(screen.getByText('Tags')).toBeInTheDocument();
    // The nested tabbed display is gone; History lives in a collapsible.
    expect(screen.queryByTestId('tabbed-display')).not.toBeInTheDocument();
    expect(screen.getByText('History')).toBeInTheDocument();
    // Links-only content is absent on the default tab.
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
  });

  it('defaults to the Links tab when the card has references', () => {
    const [ref] = samplePartialCards();
    renderPanel({
      categorizedReferences: { bidirectional: [ref], incoming: [], outgoing: [] },
    });
    // Smart default picks Links because there's relationship data.
    expect(screen.getByText('Children')).toBeInTheDocument();
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
    const [ref] = samplePartialCards();
    renderPanel({
      parentCard: null,
      categorizedReferences: { bidirectional: [ref], incoming: [], outgoing: [] },
    });
    // Smart default is Links (has references); Children + references show.
    expect(screen.getByText('Children')).toBeInTheDocument();
    expect(screen.getByText('Linked references')).toBeInTheDocument();
    // Parent header is absent when there is no parent.
    expect(screen.queryByText('Parent')).not.toBeInTheDocument();
  });

  it('switches to the Entities tab and shows the card entities', () => {
    renderPanel();
    fireEvent.click(screen.getByText('Entities'));
    expect(screen.getByText('Entity One')).toBeInTheDocument();
    // Metadata content is hidden
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });

  it('shows the empty hint on the Entities tab when the card has no entities', () => {
    renderPanel({ viewingCard: { ...viewingCard, entities: [] } });
    fireEvent.click(screen.getByText('Entities'));
    expect(screen.getByText('No entities available')).toBeInTheDocument();
  });

  it('switches to the Files tab and shows the files list', () => {
    renderPanel();
    fireEvent.click(screen.getByText('Files'));
    expect(screen.getByTestId('files-tab')).toBeInTheDocument();
    // Metadata content is hidden
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
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
    const [ref] = samplePartialCards();
    renderPanel({
      onCreateChildCard,
      categorizedReferences: { bidirectional: [ref], incoming: [], outgoing: [] },
    });
    // Smart default lands on Links because of the reference.
    fireEvent.click(screen.getByTitle('Add child'));
    expect(onCreateChildCard).toHaveBeenCalledTimes(1);
  });

  it('reflects the active tab in the ?pane= query param', () => {
    renderPanel({
      categorizedReferences: { bidirectional: [], incoming: [], outgoing: [] },
    });
    // Smart default → metadata for a relationship-less card.
    expect(window.location.search).toContain('pane=metadata');

    fireEvent.click(screen.getByText('Files'));
    expect(window.location.search).toContain('pane=files');
  });

  it('honors a valid ?pane= param on mount instead of the smart default', () => {
    window.history.replaceState({}, '', '/?pane=entities');
    renderPanel();
    // URL wins over the smart default (would otherwise be metadata).
    expect(screen.getByText('Entity One')).toBeInTheDocument();
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });

  it('falls back to the smart default for an invalid ?pane= value', () => {
    window.history.replaceState({}, '', '/?pane=bogus');
    renderPanel();
    // Invalid param ignored → smart default (metadata) applies.
    expect(screen.getByText('Tags')).toBeInTheDocument();
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
