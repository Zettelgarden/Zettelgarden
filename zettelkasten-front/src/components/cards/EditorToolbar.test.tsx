import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { EditorToolbar } from './EditorToolbar';
import { defaultCard } from '../../models/Card';
import {
  CardEditorProvider,
  EditorUIProvider,
  EditorMessagesProvider,
} from '../../contexts/editor';

describe('EditorToolbar', () => {
  // Helper function to render EditorToolbar with required providers
  function renderWithProviders(
    ui: React.ReactElement,
    {
      editingCard = { ...defaultCard, id: 1, card_id: '1', title: 'Test Card' },
      setEditingCard = vi.fn(),
      setShowSaveAsTemplate = vi.fn(),
      setMessage = vi.fn(),
    } = {}
  ) {
    const Wrapper = ({ children }: { children: React.ReactNode }) => (
      <CardEditorProvider editingCard={editingCard} setEditingCard={setEditingCard}>
        <EditorUIProvider handleSelectTemplate={vi.fn()}>
          <EditorMessagesProvider>
            {children}
          </EditorMessagesProvider>
        </EditorUIProvider>
      </CardEditorProvider>
    );

    return render(ui, { wrapper: Wrapper });
  }

  const defaultProps = {
    newCard: false,
    originalCard: { ...defaultCard, id: 1, card_id: '1', title: 'Test Card' },
    onDeleteSuccess: vi.fn(),
  };

  describe('Props rendering', () => {
    it('should display "New Card" when newCard is true', () => {
      renderWithProviders(<EditorToolbar {...defaultProps} newCard={true} />);
      expect(screen.getByText('New Card')).toBeInTheDocument();
      expect(screen.queryByText('[1]')).not.toBeInTheDocument();
    });

    it('should display card ID and title when newCard is false', () => {
      renderWithProviders(<EditorToolbar {...defaultProps} />);
      expect(screen.getByText('[1]')).toBeInTheDocument();
      expect(screen.getByText((content, element) => {
        return content.includes('Test Card');
      })).toBeInTheDocument();
      expect(screen.queryByText('New Card')).not.toBeInTheDocument();
    });

    it('should render the menu button with three dots icon', () => {
      renderWithProviders(<EditorToolbar {...defaultProps} />);
      const menuButton = screen.getByRole('button');
      expect(menuButton).toBeInTheDocument();
    });
  });

  describe('Menu interactions', () => {
    it('should show "Process Entities & Facts" checkbox only when not newCard', () => {
      renderWithProviders(<EditorToolbar {...defaultProps} />);

      // Open menu
      const menuButton = screen.getByRole('button');
      fireEvent.click(menuButton);

      // Should show checkbox for existing cards
      expect(screen.getByLabelText(/Process Entities & Facts/i)).toBeInTheDocument();
    });

    it('should not show "Process Entities & Facts" checkbox when newCard is true', () => {
      renderWithProviders(<EditorToolbar {...defaultProps} newCard={true} />);

      // Open menu
      const menuButton = screen.getByRole('button');
      fireEvent.click(menuButton);

      // Should not show checkbox for new cards
      expect(screen.queryByLabelText(/Process Entities & Facts/i)).not.toBeInTheDocument();
    });

    it('should toggle process_entities_and_facts value when checkbox is changed', () => {
      const setEditingCard = vi.fn();
      const editingCard = { ...defaultCard, id: 1, process_entities_and_facts: false };

      renderWithProviders(
        <EditorToolbar {...defaultProps} />,
        { editingCard, setEditingCard }
      );

      // Open menu
      const menuButton = screen.getByRole('button');
      fireEvent.click(menuButton);

      // Click checkbox
      const checkbox = screen.getByLabelText(/Process Entities & Facts/i);
      fireEvent.click(checkbox);

      // Should call setEditingCard with updated value
      expect(setEditingCard).toHaveBeenCalledWith({
        ...editingCard,
        process_entities_and_facts: true,
      });
    });

    it('should call setShowSaveAsTemplate when "Save as Template" menu item is clicked', () => {
      const setShowSaveAsTemplate = vi.fn();

      renderWithProviders(
        <EditorToolbar {...defaultProps} />,
        { setShowSaveAsTemplate }
      );

      // Open menu
      const menuButton = screen.getByRole('button');
      fireEvent.click(menuButton);

      // Click "Save as Template"
      const saveAsTemplateItem = screen.getByText('Save as Template');
      fireEvent.click(saveAsTemplateItem);

      expect(setShowSaveAsTemplate).toHaveBeenCalledWith(true);
    });

    it('should update active state styling on menu items', () => {
      renderWithProviders(<EditorToolbar {...defaultProps} />);

      // Open menu
      const menuButton = screen.getByRole('button');
      fireEvent.click(menuButton);

      // Verify menu items exist and have proper aria attributes
      const menuItems = screen.getAllByRole('menuitem');
      expect(menuItems.length).toBeGreaterThan(0);
    });
  });

  describe('State updates', () => {
    it('should maintain current editingCard state when toggling checkboxes', () => {
      const setEditingCard = vi.fn();
      const editingCard = {
        ...defaultCard,
        id: 1,
        title: 'Existing Title',
        body: 'Existing Body',
        process_entities_and_facts: false
      };

      renderWithProviders(
        <EditorToolbar {...defaultProps} />,
        { editingCard, setEditingCard }
      );

      // Open menu
      const menuButton = screen.getByRole('button');
      fireEvent.click(menuButton);

      // Toggle checkbox
      const checkbox = screen.getByLabelText(/Process Entities & Facts/i);
      fireEvent.click(checkbox);

      // Should preserve all existing properties
      expect(setEditingCard).toHaveBeenCalledWith({
        ...editingCard,
        process_entities_and_facts: true,
      });
    });
  });
});
