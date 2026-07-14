import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ViewCardContentSection } from './ViewCardContentSection';
import { UIStateProvider } from '../../contexts/UIStateContext';
import { sampleCards } from '../../tests/data';

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

type Props = React.ComponentProps<typeof ViewCardContentSection>;
const [viewingCard] = sampleCards();

function baseProps(overrides: Partial<Props> = {}): Props {
  return {
    viewingCard,
    latestSummary: null,
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
    // Inline relationship sections are absent (they live in the rail on
    // desktop and in ViewMobileLayout accordions on mobile); the ＋ Child /
    // ＋ Link affordances live in the header now.
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
    expect(screen.queryByText('Linked references')).not.toBeInTheDocument();
    expect(screen.queryByText('＋ Child')).not.toBeInTheDocument();
    expect(screen.queryByText('＋ Link')).not.toBeInTheDocument();
    // The tabbed display (entities/files/history/summaries) lives in the
    // rail's Metadata tab on desktop, not inline here.
    expect(screen.queryByTestId('tabbed-display')).not.toBeInTheDocument();
  });

  it('renders the tabbed display inline when showTabbedDisplay is set', () => {
    renderSection({
      showTabbedDisplay: true,
      summaries: [],
    });
    // Mobile keeps entities/files/history/summaries inline.
    expect(screen.getByTestId('tabbed-display')).toBeInTheDocument();
  });
});
