import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ViewCardContentSection } from './ViewCardContentSection';
import { UIStateProvider } from '../../contexts/UIStateContext';
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
    onCreateChildCard: vi.fn(),
    setViewCard: vi.fn(),
    setError: vi.fn(),
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

describe('ViewCardContentSection — desktop (no inline relationships)', () => {
  it('renders just the body and no inline relationships or footer buttons', () => {
    renderSection();
    // Inline relationship sections are absent on desktop (they live in the
    // rail; the ＋ Child / ＋ Link affordances live in the header now).
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
    expect(screen.queryByText('Linked references')).not.toBeInTheDocument();
    expect(screen.queryByText('＋ Child')).not.toBeInTheDocument();
    expect(screen.queryByText('＋ Link')).not.toBeInTheDocument();
    // The tabbed display (entities/files/history/summaries) lives in the
    // rail's Metadata tab on desktop, not inline here.
    expect(screen.queryByTestId('tabbed-display')).not.toBeInTheDocument();
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

  it('renders the tabbed display inline when showTabbedDisplay is set', () => {
    renderSection({
      showRelationships: true,
      showTabbedDisplay: true,
      categorizedReferences: { bidirectional: [], incoming: [], outgoing: [] },
      summaries: [],
    });
    // Mobile keeps entities/files/history/summaries inline.
    expect(screen.getByTestId('tabbed-display')).toBeInTheDocument();
  });
});
