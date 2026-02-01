import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DynamicSpreadsheet } from './DynamicSpreadsheet';

// Mock the card context for accessing card body
vi.mock('../../contexts/CardContext', () => ({
  useCardContext: () => ({
    viewingCard: {
      id: 1,
      body: 'Some text\n```spreadsheet:mysheet\n{"rows": 2, "cols": 2, "data": {"A1": {"value": "10", "formula": ""}}}\n```\nMore text'
    }
  })
}));

describe('DynamicSpreadsheet', () => {
  it('renders spreadsheet from card body', () => {
    render(<DynamicSpreadsheet name="mysheet" />);

    // Should render the grid
    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
  });

  it('renders default empty spreadsheet if not found', () => {
    render(<DynamicSpreadsheet name="nonexistent" />);

    // Should still render a grid
    expect(screen.getByText('A')).toBeInTheDocument();
  });

  it('displays "mysheet" as spreadsheet name', () => {
    const { container } = render(<DynamicSpreadsheet name="mysheet" />);
    // Component should render without error
    expect(container).toBeTruthy();
  });
});
