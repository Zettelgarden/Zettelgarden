import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { EditPageHeader } from './EditPageHeader';
import { defaultCard } from '../../models/Card';
import { UIStateProvider } from '../../contexts/UIStateContext';
import {
  CardEditorProvider,
  EditorUIProvider,
  EditorMessagesProvider,
} from '../../contexts/editor';

describe('EditPageHeader', () => {
  // Render EditPageHeader with the full set of providers it reads from.
  function renderWithProviders(
    ui: React.ReactElement,
    {
      editingCard = { ...defaultCard, id: 1, card_id: 'abc', title: 'Test Card', body: 'body' },
      setEditingCard = vi.fn(),
      setShowSaveAsTemplate = vi.fn(),
      setMessage = vi.fn(),
      message = '',
      error = '',
    } = {},
  ) {
    const Wrapper = ({ children }: { children: React.ReactNode }) => (
      <UIStateProvider>
        <CardEditorProvider editingCard={editingCard} setEditingCard={setEditingCard}>
          <EditorUIProvider handleSelectTemplate={vi.fn()}>
            <EditorMessagesProvider initialMessage={message} initialError={error}>
              {children}
            </EditorMessagesProvider>
          </EditorUIProvider>
        </CardEditorProvider>
      </UIStateProvider>
    );

    return render(ui, { wrapper: Wrapper });
  }

  const defaultProps = {
    newCard: false,
    originalCard: { ...defaultCard, id: 1, card_id: 'abc', title: 'Test Card' },
    suggestingTitle: false,
    handleSuggestTitle: vi.fn(),
    handleSaveCard: vi.fn(),
    handleCancelButtonClick: vi.fn(),
    onDeleteSuccess: vi.fn(),
  };

  describe('Breadcrumb + title', () => {
    it('shows the original card_id in the breadcrumb for existing cards', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} />);
      expect(screen.getByText('[abc]')).toBeInTheDocument();
    });

    it('shows the proposed card_id in the breadcrumb for new cards', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} newCard={true} />, {
        editingCard: { ...defaultCard, id: 0, card_id: 'new-id', title: '', body: '' },
      });
      expect(screen.getByText('[new-id]')).toBeInTheDocument();
    });

    it('falls back to [new] in the breadcrumb when a new card has no id yet', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} newCard={true} />, {
        editingCard: { ...defaultCard, id: 0, card_id: '', title: '', body: '' },
      });
      expect(screen.getByText('[new]')).toBeInTheDocument();
    });

    it('renders the editable title input with the current value', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} />);
      const titleInput = screen.getByLabelText('Title');
      expect(titleInput).toHaveValue('Test Card');
      expect(titleInput).toHaveAttribute('placeholder', 'Untitled');
    });

    it('updates the title via setEditingCard on change', () => {
      const setEditingCard = vi.fn();
      renderWithProviders(<EditPageHeader {...defaultProps} />, {
        editingCard: { ...defaultCard, id: 1, card_id: 'abc', title: 'Old', body: 'body' },
        setEditingCard,
      });
      const titleInput = screen.getByLabelText('Title');
      fireEvent.change(titleInput, { target: { value: 'New Title' } });
      expect(setEditingCard).toHaveBeenCalledWith(
        expect.objectContaining({ title: 'New Title' }),
      );
    });
  });

  describe('Save / Cancel actions', () => {
    it('renders Save and Cancel buttons', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    });

    it('calls handleSaveCard when Save is clicked', () => {
      const handleSaveCard = vi.fn();
      renderWithProviders(
        <EditPageHeader {...defaultProps} handleSaveCard={handleSaveCard} />,
      );
      fireEvent.click(screen.getByRole('button', { name: 'Save' }));
      expect(handleSaveCard).toHaveBeenCalled();
    });

    it('calls handleCancelButtonClick when Cancel is clicked', () => {
      const handleCancelButtonClick = vi.fn();
      renderWithProviders(
        <EditPageHeader {...defaultProps} handleCancelButtonClick={handleCancelButtonClick} />,
      );
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      expect(handleCancelButtonClick).toHaveBeenCalled();
    });
  });

  describe('Suggest title button', () => {
    it('calls handleSuggestTitle when clicked and enabled', () => {
      const handleSuggestTitle = vi.fn();
      renderWithProviders(
        <EditPageHeader {...defaultProps} handleSuggestTitle={handleSuggestTitle} />,
      );
      const button = screen.getByTitle('Suggest title from content');
      fireEvent.click(button);
      expect(handleSuggestTitle).toHaveBeenCalled();
    });

    it('is disabled when the body is empty', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} />, {
        editingCard: { ...defaultCard, id: 1, card_id: 'abc', title: 'T', body: '' },
      });
      expect(screen.getByTitle('Suggest title from content')).toBeDisabled();
    });

    it('is disabled while suggesting', () => {
      renderWithProviders(
        <EditPageHeader {...defaultProps} suggestingTitle={true} />,
      );
      expect(screen.getByTitle('Suggesting title...')).toBeDisabled();
    });
  });

  describe('Rail toggle', () => {
    it('renders the desktop-only info pane toggle', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} />);
      expect(screen.getByTitle('Toggle info pane')).toBeInTheDocument();
    });

    it('reflects the open state via aria-pressed', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} />);
      const toggle = screen.getByRole('button', { name: 'Toggle info pane' });
      expect(toggle).toHaveAttribute('aria-pressed', 'true');
    });
  });

  describe('Overflow menu', () => {
    function openMenu() {
      fireEvent.click(screen.getByTitle('More actions'));
    }

    it('shows Process Entities & Facts checkbox only for existing cards', () => {
      const { rerender } = renderWithProviders(<EditPageHeader {...defaultProps} />);
      openMenu();
      expect(screen.getByLabelText(/Process Entities & Facts/i)).toBeInTheDocument();
    });

    it('hides Process Entities & Facts checkbox for new cards', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} newCard={true} />, {
        editingCard: { ...defaultCard, id: 0, card_id: 'new-id', title: '', body: '' },
      });
      openMenu();
      expect(screen.queryByLabelText(/Process Entities & Facts/i)).not.toBeInTheDocument();
    });

    it('toggles process_entities_and_facts via the checkbox', () => {
      const setEditingCard = vi.fn();
      renderWithProviders(<EditPageHeader {...defaultProps} />, {
        editingCard: {
          ...defaultCard,
          id: 1,
          card_id: 'abc',
          title: 'T',
          body: 'b',
          process_entities_and_facts: false,
        },
        setEditingCard,
      });
      openMenu();
      fireEvent.click(screen.getByLabelText(/Process Entities & Facts/i));
      expect(setEditingCard).toHaveBeenCalledWith(
        expect.objectContaining({ process_entities_and_facts: true }),
      );
    });

    it('renders Save as Template for both new and existing cards', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} newCard={true} />, {
        editingCard: { ...defaultCard, id: 0, card_id: 'new-id', title: '', body: '' },
      });
      openMenu();
      expect(screen.getByText('Save as Template')).toBeInTheDocument();
    });

    it('shows Delete Card only for existing cards', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} />);
      openMenu();
      expect(screen.getByText('Delete Card')).toBeInTheDocument();
    });

    it('hides Delete Card for new cards', () => {
      renderWithProviders(<EditPageHeader {...defaultProps} newCard={true} />, {
        editingCard: { ...defaultCard, id: 0, card_id: 'new-id', title: '', body: '' },
      });
      openMenu();
      expect(screen.queryByText('Delete Card')).not.toBeInTheDocument();
    });
  });
});
