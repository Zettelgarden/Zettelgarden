import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ViewCardContentSection } from './ViewCardContentSection';
import { sampleCards } from '../../tests/data';

// CardBody pulls in markdown rendering; mock it to keep this focused.
vi.mock('./CardBody', () => ({
  CardBody: () => <div data-testid="card-body">body</div>,
}));

type Props = React.ComponentProps<typeof ViewCardContentSection>;
const [viewingCard] = sampleCards();

function baseProps(overrides: Partial<Props> = {}): Props {
  return {
    viewingCard,
    latestSummary: null,
    ...overrides,
  };
}

function renderSection(overrides: Partial<Props> = {}) {
  return render(
    <BrowserRouter>
      <ViewCardContentSection {...baseProps(overrides)} />
    </BrowserRouter>,
  );
}

describe('ViewCardContentSection', () => {
  it('renders the card body and no inline relationships or footer buttons', () => {
    renderSection();
    expect(screen.getByTestId('card-body')).toBeInTheDocument();
    // Relationship/tabbed sections live elsewhere now: desktop renders them in
    // the rail, mobile in ViewMobileLayout accordions. The ＋ Child / ＋ Link
    // affordances live in the header.
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
    expect(screen.queryByText('Linked references')).not.toBeInTheDocument();
    expect(screen.queryByText('＋ Child')).not.toBeInTheDocument();
    expect(screen.queryByText('＋ Link')).not.toBeInTheDocument();
    expect(screen.queryByTestId('tabbed-display')).not.toBeInTheDocument();
  });
});
