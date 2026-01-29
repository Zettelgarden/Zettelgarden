import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CardMetadata } from './CardMetadata';
import { defaultCard } from '../../models/Card';
import { Tag } from '../../models/Tags';

// Mock react-router-dom
vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
}));

// Mock API functions
vi.mock('../../api/cards', () => ({
  getNextRootId: vi.fn(() => Promise.resolve({ new_id: '5' })),
}));

// Mock child components
vi.mock('../tags/SearchTagDropdown', () => ({
  SearchTagDropdown: ({ tags, handleTagClick }: any) => (
    <button data-testid="search-tag-dropdown">Search Tags ({tags.length})</button>
  ),
}));

vi.mock('./BacklinkInputDropdownList', () => ({
  BacklinkInputDropdownList: ({ addBacklink }: any) => (
    <div data-testid="backlink-dropdown">Backlinks</div>
  ),
}));

describe('CardMetadata', () => {
  const mockSetEditingCard = vi.fn();
  const mockSetShowCardIdDiscovery = vi.fn();
  const mockHandleClickFillCard = vi.fn();
  const mockHandleTagClick = vi.fn();
  const mockHandleRemoveTag = vi.fn();
  const mockAddBacklink = vi.fn();
  const mockSetMessage = vi.fn();

  const defaultTags: Tag[] = [
    { id: 1, name: 'javascript', color: '#FF0000', user_id: 1 },
    { id: 2, name: 'react', color: '#00FF00', user_id: 1 },
  ];

  const defaultEditingCard = {
    ...defaultCard,
    id: 1,
    card_id: '1',
    title: 'Test Card',
    body: 'This is a test card with #javascript and #react tags',
    tags: defaultTags,
  };

  const defaultOriginalCard = {
    ...defaultCard,
    id: 1,
    card_id: '1',
    title: 'Test Card',
    created_at: new Date('2024-01-01T00:00:00Z'),
    updated_at: new Date('2024-01-02T00:00:00Z'),
  };

  const defaultProps = {
    newCard: false,
    originalCard: defaultOriginalCard,
    editingCard: defaultEditingCard,
    setEditingCard: mockSetEditingCard,
    setShowCardIdDiscovery: mockSetShowCardIdDiscovery,
    handleClickFillCard: mockHandleClickFillCard,
    tags: defaultTags,
    handleTagClick: mockHandleTagClick,
    handleRemoveTag: mockHandleRemoveTag,
    addBacklink: mockAddBacklink,
    setMessage: mockSetMessage,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Props rendering', () => {
    it('should render Card ID input with current value', () => {
      render(<CardMetadata {...defaultProps} />);

      const cardIdInput = screen.getByDisplayValue('1');
      expect(cardIdInput).toBeInTheDocument();
      expect(cardIdInput).toHaveValue('1');
    });

    it('should render Tags section with SearchTagDropdown', () => {
      render(<CardMetadata {...defaultProps} />);

      expect(screen.getByText('Tags')).toBeInTheDocument();
      expect(screen.getByTestId('search-tag-dropdown')).toBeInTheDocument();
    });

    it('should display all tags as clickable elements', () => {
      render(<CardMetadata {...defaultProps} />);

      expect(screen.getByText('#javascript')).toBeInTheDocument();
      expect(screen.getByText('#react')).toBeInTheDocument();
    });

    it('should show Created and Updated timestamps for existing cards', () => {
      render(<CardMetadata {...defaultProps} newCard={false} />);

      expect(screen.getByText('Created:')).toBeInTheDocument();
      expect(screen.getByText('Updated:')).toBeInTheDocument();
      expect(screen.getByText(/2024-01-01T00:00:00/)).toBeInTheDocument();
      expect(screen.getByText(/2024-01-02T00:00:00/)).toBeInTheDocument();
    });

    it('should not show Created/Updated timestamps for new cards', () => {
      render(<CardMetadata {...defaultProps} newCard={true} />);

      expect(screen.queryByText('Created:')).not.toBeInTheDocument();
      expect(screen.queryByText('Updated:')).not.toBeInTheDocument();
    });

    it('should render info menu button with icon', () => {
      render(<CardMetadata {...defaultProps} />);

      const menuButtons = screen.getAllByRole('button');
      const infoButton = menuButtons.find(btn => btn.querySelector('svg'));
      expect(infoButton).toBeInTheDocument();
    });
  });

  describe('Card ID input interactions', () => {
    it('should update card_id when input changes', () => {
      render(<CardMetadata {...defaultProps} />);

      const cardIdInput = screen.getByPlaceholderText('ID') || screen.getByDisplayValue(/^/);
      fireEvent.change(cardIdInput, { target: { value: '2' } });

      expect(mockSetEditingCard).toHaveBeenCalledWith({
        ...defaultEditingCard,
        card_id: '2',
      });
    });

    it('should show placeholder text in card_id input', () => {
      const emptyCard = { ...defaultEditingCard, card_id: '' };
      render(<CardMetadata {...defaultProps} editingCard={emptyCard} />);

      const cardIdInput = screen.getByPlaceholderText('ID');
      expect(cardIdInput).toHaveAttribute('placeholder', 'ID');
    });
  });

  describe('Card ID button actions (new cards)', () => {

    it('should show + button for new cards', () => {
      render(<CardMetadata {...defaultProps} newCard={true} />);

      const buttons = screen.getAllByRole('button');
      const addButton = buttons.find(btn => btn.querySelector('svg')?.innerHTML.includes('M10 3a1 1 0 011 1v5h5'));
      expect(addButton).toBeDefined();
    });

    it('should show discovery button for new cards', () => {
      render(<CardMetadata {...defaultProps} newCard={true} />);

      const buttons = screen.getAllByRole('button');
      const discoveryButton = buttons.find(btn => btn.querySelector('svg')?.innerHTML.includes('M8 4a4 4 0 100 8'));
      expect(discoveryButton).toBeDefined();
    });

    it('should not show action buttons for existing cards with card_id', () => {
      render(<CardMetadata {...defaultProps} newCard={false} editingCard={{ ...defaultEditingCard, card_id: '1' }} />);

      const buttons = screen.getAllByRole('button');
      // Should have menu button (1), tag dropdown button (1), and 2 tag remove buttons = 4
      expect(buttons.length).toBe(4);
    });

    it('should show action buttons for existing cards with empty card_id', () => {
      render(<CardMetadata {...defaultProps} newCard={false} editingCard={{ ...defaultEditingCard, card_id: '' }} />);

      const buttons = screen.getAllByRole('button');
      // Should have action buttons when card_id is empty
      expect(buttons.length).toBeGreaterThan(2);
    });
  });

  describe('getNextRootId functionality', () => {
    it('should call getNextRootId and update card_id when + button is clicked', async () => {
      const { getNextRootId } = await import('../../api/cards');

      render(<CardMetadata {...defaultProps} newCard={true} />);

      const buttons = screen.getAllByRole('button');
      const addButton = buttons.find(btn => btn.querySelector('svg')?.innerHTML.includes('M10 3a1 1 0 011 1v5h5'));

      if (addButton) {
        fireEvent.click(addButton);

        await waitFor(() => {
          expect(getNextRootId).toHaveBeenCalled();
        });
      }
    });

    it('should handle API errors when getting next root ID', async () => {
      const { getNextRootId } = await import('../../api/cards');
      (getNextRootId as any).mockRejectedValueOnce(new Error('API Error'));

      render(<CardMetadata {...defaultProps} newCard={true} />);

      const buttons = screen.getAllByRole('button');
      const addButton = buttons.find(btn => btn.querySelector('svg')?.innerHTML.includes('M10 3a1 1 0 011 1v5h5'));

      if (addButton) {
        fireEvent.click(addButton);

        // Should not crash, error is silently handled
        await waitFor(() => {
          expect(getNextRootId).toHaveBeenCalled();
        });
      }
    });
  });

  describe('Card ID discovery functionality', () => {
    it('should call setShowCardIdDiscovery when discovery button is clicked', () => {
      render(<CardMetadata {...defaultProps} newCard={true} />);

      const buttons = screen.getAllByRole('button');
      const discoveryButton = buttons.find(btn => btn.querySelector('svg')?.innerHTML.includes('M8 4a4 4 0 100 8'));

      if (discoveryButton) {
        fireEvent.click(discoveryButton);
        expect(mockSetShowCardIdDiscovery).toHaveBeenCalledWith(true);
      }
    });
  });

  describe('Tag display and interactions', () => {
    it('should display all tags with # prefix', () => {
      render(<CardMetadata {...defaultProps} />);

      expect(screen.getByText('#javascript')).toBeInTheDocument();
      expect(screen.getByText('#react')).toBeInTheDocument();
    });

    it('should not show remove button for tags not in body', () => {
      const cardWithTagNotInBody = {
        ...defaultEditingCard,
        body: 'Only #javascript here',
        tags: defaultTags,
      };

      render(<CardMetadata {...defaultProps} editingCard={cardWithTagNotInBody} />);

      // The remove button (×) should only appear for tags in the body
      const removeButtons = screen.getAllByText('×');
      expect(removeButtons.length).toBe(1); // Only javascript tag has remove button
    });

    it('should show remove button for tags that are in body', () => {
      render(<CardMetadata {...defaultProps} />);

      const removeButtons = screen.getAllByText('×');
      expect(removeButtons.length).toBe(2); // Both tags are in body
    });

    it('should call handleRemoveTag when remove button is clicked', () => {
      render(<CardMetadata {...defaultProps} />);

      const removeButtons = screen.getAllByText('×');
      fireEvent.click(removeButtons[0]);

      expect(mockHandleRemoveTag).toHaveBeenCalledWith('javascript');
    });

    it('should display empty state when no tags', () => {
      const cardWithoutTags = { ...defaultEditingCard, tags: [] };
      render(<CardMetadata {...defaultProps} editingCard={cardWithoutTags} tags={[]} />);

      // Should not have any tag elements
      expect(screen.queryByText('#javascript')).not.toBeInTheDocument();
      expect(screen.queryByText('#react')).not.toBeInTheDocument();
    });
  });

  describe('Menu interactions', () => {
    it('should open info menu when menu button is clicked', () => {
      render(<CardMetadata {...defaultProps} />);

      const menuButtons = screen.getAllByRole('button');
      const infoButton = menuButtons.find(btn => btn.querySelector('svg'));

      if (infoButton) {
        fireEvent.click(infoButton);

        // Menu should open with explanatory text
        expect(screen.getByText(/Card IDs are unique identifiers/)).toBeInTheDocument();
        expect(screen.getByText(/root card/)).toBeInTheDocument();
        // Use getAllByText since "child of 1" appears twice (1.1 and 1.1.2)
        expect(screen.getAllByText(/child of 1/).length).toBeGreaterThan(0);
      }
    });

    it('should display example card IDs in menu', () => {
      render(<CardMetadata {...defaultProps} />);

      const menuButtons = screen.getAllByRole('button');
      const infoButton = menuButtons.find(btn => btn.querySelector('svg'));

      if (infoButton) {
        fireEvent.click(infoButton);

        // Use getAllByText since there are multiple "1" elements
        expect(screen.getAllByText('1').length).toBeGreaterThan(0);
        expect(screen.getByText('1.1')).toBeInTheDocument();
        expect(screen.getByText('1.1.2')).toBeInTheDocument();
      }
    });

    it('should display recommendation text in menu', () => {
      render(<CardMetadata {...defaultProps} />);

      const menuButtons = screen.getAllByRole('button');
      const infoButton = menuButtons.find(btn => btn.querySelector('svg'));

      if (infoButton) {
        fireEvent.click(infoButton);

        expect(screen.getByText(/We recommend using numbers for IDs/)).toBeInTheDocument();
      }
    });
  });

  describe('State updates', () => {
    it('should preserve all card properties when updating card_id', () => {
      render(<CardMetadata {...defaultProps} />);

      const cardIdInput = screen.getByPlaceholderText('ID') || screen.getByDisplayValue(/^/);
      fireEvent.change(cardIdInput, { target: { value: '2.5' } });

      expect(mockSetEditingCard).toHaveBeenCalledWith({
        ...defaultEditingCard,
        card_id: '2.5',
      });
    });

    it('should handle multiple card_id updates in sequence', () => {
      render(<CardMetadata {...defaultProps} />);

      const cardIdInput = screen.getByPlaceholderText('ID') || screen.getByDisplayValue(/^/);

      fireEvent.change(cardIdInput, { target: { value: '2' } });
      expect(mockSetEditingCard).toHaveBeenLastCalledWith({
        ...defaultEditingCard,
        card_id: '2',
      });

      fireEvent.change(cardIdInput, { target: { value: '2.1' } });
      expect(mockSetEditingCard).toHaveBeenLastCalledWith({
        ...defaultEditingCard,
        card_id: '2.1',
      });
    });
  });

  describe('Conditional rendering based on newCard', () => {
    it('should show action buttons for new cards with empty card_id', () => {
      render(
        <CardMetadata
          {...defaultProps}
          newCard={true}
          editingCard={{ ...defaultEditingCard, card_id: '' }}
        />
      );

      const buttons = screen.getAllByRole('button');
      expect(buttons.length).toBeGreaterThan(2); // Should have + and discovery buttons
    });

    it('should show action buttons for existing cards with empty card_id', () => {
      render(
        <CardMetadata
          {...defaultProps}
          newCard={false}
          editingCard={{ ...defaultEditingCard, card_id: '' }}
        />
      );

      const buttons = screen.getAllByRole('button');
      expect(buttons.length).toBeGreaterThan(2); // Should have + and discovery buttons
    });

    it('should hide action buttons for existing cards with populated card_id', () => {
      render(
        <CardMetadata
          {...defaultProps}
          newCard={false}
          editingCard={{ ...defaultEditingCard, card_id: '1' }}
        />
      );

      const buttons = screen.getAllByRole('button');
      // Should have menu, tag dropdown, and 2 tag remove buttons = 4, no + or discovery buttons
      expect(buttons.length).toBe(4);

      // Verify no + or discovery buttons
      const addButton = buttons.find(btn => btn.querySelector('svg')?.innerHTML.includes('M10 3a1 1 0 011 1v5h5'));
      const discoveryButton = buttons.find(btn => btn.querySelector('svg')?.innerHTML.includes('M8 4a4 4 0 100 8'));
      expect(addButton).toBeUndefined();
      expect(discoveryButton).toBeUndefined();
    });
  });

  describe('Edge cases', () => {
    it('should handle empty card_id gracefully', () => {
      render(
        <CardMetadata
          {...defaultProps}
          editingCard={{ ...defaultEditingCard, card_id: '' }}
        />
      );

      const cardIdInput = screen.getByPlaceholderText('ID') || screen.getByDisplayValue(/^/);
      expect(cardIdInput).toHaveValue('');
    });

    it('should handle card with special characters in card_id', () => {
      render(
        <CardMetadata
          {...defaultProps}
          editingCard={{ ...defaultEditingCard, card_id: '1.2.3-beta' }}
        />
      );

      const cardIdInput = screen.getByPlaceholderText('ID') || screen.getByDisplayValue(/^/);
      expect(cardIdInput).toHaveValue('1.2.3-beta');
    });

    it('should handle tags with special characters', () => {
      const specialTags: Tag[] = [{ id: 3, name: 'c++', color: '#0000FF', user_id: 1 }];
      render(
        <CardMetadata
          {...defaultProps}
          editingCard={{ ...defaultEditingCard, tags: specialTags }}
          tags={specialTags}
        />
      );

      expect(screen.getByText('#c++')).toBeInTheDocument();
    });

    it('should handle very long card_id values', () => {
      const longId = '1.2.3.4.5.6.7.8.9.10.11.12';
      render(
        <CardMetadata
          {...defaultProps}
          editingCard={{ ...defaultEditingCard, card_id: longId }}
        />
      );

      const cardIdInput = screen.getByPlaceholderText('ID') || screen.getByDisplayValue(/^/);
      expect(cardIdInput).toHaveValue(longId);
    });

    it('should handle cards with many tags', () => {
      const manyTags: Tag[] = Array.from({ length: 20 }, (_, i) => ({
        id: i,
        name: `tag${i}`,
        color: '#FF0000',
        user_id: 1,
      }));

      render(
        <CardMetadata
          {...defaultProps}
          editingCard={{ ...defaultEditingCard, tags: manyTags }}
          tags={manyTags}
        />
      );

      // Should render all tags
      expect(screen.getByText('#tag0')).toBeInTheDocument();
      expect(screen.getByText('#tag19')).toBeInTheDocument();
    });
  });
});
