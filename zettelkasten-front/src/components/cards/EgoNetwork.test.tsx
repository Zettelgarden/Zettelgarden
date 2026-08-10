// zettelkasten-front/src/components/cards/EgoNetwork.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { EgoNetwork } from './EgoNetwork';
import { PartialCard, RelatedCard } from '../../models/Card';

function makePartial(id: number, cardId: string, title: string): PartialCard {
  return {
    id,
    card_id: cardId,
    user_id: 1,
    title,
    parent_id: 0,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
  };
}

function makeRelated(id: number, cardId: string, title: string): RelatedCard {
  return { card: makePartial(id, cardId, title), score: 5, reasons: [] };
}

const center = makePartial(1, 'CENT', 'Center Card');

describe('EgoNetwork', () => {
  it('shows an empty-state message when there are no connections', () => {
    render(
      <EgoNetwork
        center={center}
        parent={null}
        children={[]}
        references={[]}
        relatedCards={[]}
        suggestions={[]}
        onCardClick={() => {}}
      />,
    );
    expect(screen.getByText(/no connections yet/i)).toBeInTheDocument();
    expect(screen.queryByTestId('ego-network')).not.toBeInTheDocument();
  });

  it('renders ring-1 nodes for parent, children, references, and related cards', () => {
    render(
      <EgoNetwork
        center={center}
        parent={makePartial(2, 'PAR', 'Parent Card')}
        children={[makePartial(3, 'CH1', 'Child One')]}
        references={[makePartial(4, 'REF1', 'Referenced Card')]}
        relatedCards={[makeRelated(5, 'REL1', 'Related Card')]}
        suggestions={[]}
        onCardClick={() => {}}
      />,
    );

    expect(screen.getByTestId('ego-node-1')).toBeInTheDocument(); // center
    expect(screen.getByTestId('ego-node-2')).toBeInTheDocument(); // parent
    expect(screen.getByTestId('ego-node-3')).toBeInTheDocument(); // child
    expect(screen.getByTestId('ego-node-4')).toBeInTheDocument(); // reference
    expect(screen.getByTestId('ego-node-5')).toBeInTheDocument(); // related
    // Edge type labels under each node.
    expect(screen.getByText('parent')).toBeInTheDocument();
    expect(screen.getByText('child')).toBeInTheDocument();
    expect(screen.getByText('reference')).toBeInTheDocument();
    expect(screen.getByText('related')).toBeInTheDocument();
  });

  it('calls onCardClick when a ring-1 node is clicked', () => {
    const onCardClick = vi.fn();
    render(
      <EgoNetwork
        center={center}
        parent={null}
        children={[]}
        references={[makePartial(4, 'REF1', 'Referenced Card')]}
        relatedCards={[]}
        suggestions={[]}
        onCardClick={onCardClick}
      />,
    );

    fireEvent.click(screen.getByTestId('ego-node-4'));
    expect(onCardClick).toHaveBeenCalledWith(4);
  });

  it('renders two-hop suggestion chips and navigates on click', () => {
    const onCardClick = vi.fn();
    render(
      <EgoNetwork
        center={center}
        parent={null}
        children={[]}
        references={[]}
        relatedCards={[]}
        suggestions={[makeRelated(9, 'SUG1', 'Two Hops Away')]}
        onCardClick={onCardClick}
      />,
    );

    expect(screen.getByText('Two hops out')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Two Hops Away'));
    expect(onCardClick).toHaveBeenCalledWith(9);
  });

  it('excludes suggestion cards already shown in ring 1', () => {
    render(
      <EgoNetwork
        center={center}
        parent={null}
        children={[]}
        references={[makePartial(9, 'SUG1', 'Two Hops Away')]}
        relatedCards={[]}
        suggestions={[makeRelated(9, 'SUG1', 'Two Hops Away')]}
        onCardClick={() => {}}
      />,
    );

    // Card 9 appears once, as a ring-1 reference; no duplicate chip.
    expect(screen.getByTestId('ego-node-9')).toBeInTheDocument();
    expect(screen.queryByText('Two hops out')).not.toBeInTheDocument();
  });
});
