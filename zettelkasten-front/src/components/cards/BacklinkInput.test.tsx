import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BacklinkInput } from './BacklinkInput';

// Mock the BacklinkInputDropdownList component
vi.mock('./BacklinkInputDropdownList', () => ({
  BacklinkInputDropdownList: ({ placeholder, onSelect }: any) => (
    <div>
      <input placeholder={placeholder} />
      <button onClick={() => onSelect({ card_id: '1', title: 'Test Card' })}>
        Select Test Card
      </button>
    </div>
  ),
}));

describe('BacklinkInput', () => {
  it('renders placeholder text', () => {
    render(<BacklinkInput addBacklink={vi.fn()} />);
    expect(screen.getByPlaceholderText('Add Backlink')).toBeInTheDocument();
  });

  it('calls addBacklink when card is selected', async () => {
    const handleAddBacklink = vi.fn();
    render(<BacklinkInput addBacklink={handleAddBacklink} />);

    const selectButton = screen.getByText('Select Test Card');
    selectButton.click();

    expect(handleAddBacklink).toHaveBeenCalledWith({
      card_id: '1',
      title: 'Test Card',
    });
  });
});
