import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ViewCardContentSection } from './ViewCardContentSection';
import { UIStateProvider, useUIState } from '../../contexts/UIStateContext';
import { sampleCards, samplePartialCards } from '../../tests/data';

// Heavy children are mocked to keep this a focused unit test and avoid
// pulling TagContext / react-pdf transitively.
vi.mock('./CardBody', () => ({
  CardBody: () => <div data-testid="card-body">body</div>,
}));
vi.mock('./ViewCardTabbedDisplay', () => ({
  ViewCardTabbedDisplay: () => <div data-testid="tabbed-display">tabs</div>,
}));
vi.mock('./CardList', () => ({
  CardList: ({ cards }: any) => <div data-testid="card-list">{cards.length}</div>,
}));
vi.mock('./ChildrenCards', () => ({
  ChildrenCards: ({ allChildren }: any) => <div data-testid="children-cards">{allChildren.length}</div>,
}));
vi.mock('./BacklinkInput', () => ({
  BacklinkInput: () => <div data-testid="backlink-input">Add Backlink</div>,
}));

type Props = React.ComponentProps<typeof ViewCardContentSection>;
const [viewingCard] = sampleCards();

function baseProps(overrides: Partial<Props> = {}): Props {
  return {
    viewingCard,
    latestSummary: null,
    analysis: null,
    onCreateChildCard: vi.fn(),
    setViewCard: vi.fn(),
    setError: vi.fn(),
    handleOpenEntity: vi.fn(),
    summaries: null,
    fileUploadRef: { current: null } as React.RefObject<HTMLInputElement>,
    ...overrides,
  };
}

function renderSection(overrides: Partial<Props> = {}) {
  return render(
    <BrowserRouter>
      <UIStateProvider>
        <ViewCardContentSection {...baseProps(overrides)} />
      </UIStateProvider>
    </BrowserRouter>,
  );
}

describe('ViewCardContentSection — desktop (footer affordance)', () => {
  it('shows the +Child and +Link footer buttons and no inline relationships', () => {
    renderSection();
    expect(screen.getByText('＋ Child')).toBeInTheDocument();
    expect(screen.getByText('＋ Link')).toBeInTheDocument();
    // Inline relationship sections are gone on desktop.
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
    expect(screen.queryByText('Linked references')).not.toBeInTheDocument();
  });

  it('opens the rail to the Links tab when +Link is clicked', () => {
    function Harness() {
      const { rightPaneOpen, rightPaneTab } = useUIState();
      return (
        <>
          <ViewCardContentSection {...baseProps()} />
          <div data-testid="rail-open">{rightPaneOpen ? 'open' : 'closed'}</div>
          <div data-testid="rail-tab">{rightPaneTab}</div>
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
    // Default state: rail may be open already; the tab should flip to links.
    fireEvent.click(screen.getByText('＋ Link'));
    expect(screen.getByTestId('rail-open').textContent).toBe('open');
    expect(screen.getByTestId('rail-tab').textContent).toBe('links');
  });

  it('triggers child creation and opens the rail when +Child is clicked', () => {
    const onCreateChildCard = vi.fn();
    function Harness() {
      const { rightPaneOpen, rightPaneTab } = useUIState();
      return (
        <>
          <ViewCardContentSection {...baseProps({ onCreateChildCard })} />
          <div data-testid="rail-open">{rightPaneOpen ? 'open' : 'closed'}</div>
          <div data-testid="rail-tab">{rightPaneTab}</div>
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
    fireEvent.click(screen.getByText('＋ Child'));
    expect(onCreateChildCard).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('rail-open').textContent).toBe('open');
    expect(screen.getByTestId('rail-tab').textContent).toBe('links');
  });
});

describe('ViewCardContentSection — mobile (showRelationships)', () => {
  it('renders inline Children + Linked references and no footer buttons', () => {
    const [ref] = samplePartialCards();
    renderSection({
      showRelationships: true,
      categorizedReferences: { bidirectional: [ref], incoming: [], outgoing: [] },
      onAddBacklink: vi.fn(),
    });
    expect(screen.getByText('Children')).toBeInTheDocument();
    expect(screen.getByText('Linked references')).toBeInTheDocument();
    // Footer affordance is desktop-only.
    expect(screen.queryByText('＋ Child')).not.toBeInTheDocument();
    expect(screen.queryByText('＋ Link')).not.toBeInTheDocument();
  });
});
