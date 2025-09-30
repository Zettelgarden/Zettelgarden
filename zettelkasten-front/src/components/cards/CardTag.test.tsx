import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CardTag } from './CardTag';
import { PartialCard } from '../../models/Card';

describe('CardTag', () => {
  const mockCard: PartialCard = {
    id: 1,
    card_id: '1/A.1',
    user_id: 1,
    title: 'Test Card Title',
    parent_id: 0,
    created_at: new Date(),
    updated_at: new Date(),
    tags: [],
  };

  it('renders card ID', () => {
    render(<CardTag card={mockCard} showTitle={false} />);
    expect(screen.getByText('[1/A.1]')).toBeInTheDocument();
  });

  it('shows title when showTitle is true', () => {
    render(<CardTag card={mockCard} showTitle={true} />);
    expect(screen.getByText('[1/A.1]')).toBeInTheDocument();
    expect(screen.getByText(/Test Card Title/)).toBeInTheDocument();
  });

  it('hides title when showTitle is false', () => {
    render(<CardTag card={mockCard} showTitle={false} />);
    expect(screen.getByText('[1/A.1]')).toBeInTheDocument();
    expect(screen.queryByText(/Test Card Title/)).not.toBeInTheDocument();
  });

  it('renders with different card IDs', () => {
    const card: PartialCard = {
      id: 2,
      card_id: '10/B.5/C',
      user_id: 1,
      title: 'Another Card',
      parent_id: 0,
      created_at: new Date(),
      updated_at: new Date(),
      tags: [],
    };
    render(<CardTag card={card} showTitle={false} />);
    expect(screen.getByText('[10/B.5/C]')).toBeInTheDocument();
  });
});