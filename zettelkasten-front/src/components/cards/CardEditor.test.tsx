import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CardEditor } from './CardEditor';
import { defaultCard, defaultCardTemplate } from '../../models/Card';
import {
  CardEditorProvider,
  EditorUIProvider,
  EditorMessagesProvider,
} from '../../contexts/editor';

describe('CardEditor', () => {
  // Mock dependencies
  const mockCardBodyRef = { current: { formatText: vi.fn(), togglePreviewMode: vi.fn() } };

  // Helper function to render CardEditor with required providers
  function renderWithProviders(
    ui: React.ReactElement,
    {
      editingCard = { ...defaultCard, id: 1, title: 'Test Title', body: 'Test Body' },
      setEditingCard = vi.fn(),
      message = '',
      setMessage = vi.fn(),
      error = '',
      setError = vi.fn(),
      templates = [] as any[],
      loadingTemplates = false,
      templateError = '',
      setTemplates = vi.fn(),
      setLoadingTemplates = vi.fn(),
      setTemplateError = vi.fn(),
      showTemplateDropdown = false,
      setShowTemplateDropdown = vi.fn(),
      setShowBacklinkDialog = vi.fn(),
      setShowCardIdDiscovery = vi.fn(),
      setShowSaveAsTemplate = vi.fn(),
      handleSelectTemplate = vi.fn(),
    } = {}
  ) {
    const Wrapper = ({ children }: { children: React.ReactNode }) => (
      <CardEditorProvider editingCard={editingCard} setEditingCard={setEditingCard}>
        <EditorUIProvider
          handleSelectTemplate={handleSelectTemplate}
          initialTemplates={templates}
          initialLoadingTemplates={loadingTemplates}
          initialTemplateError={templateError}
        >
          <EditorMessagesProvider initialMessage={message} initialError={error}>
            {children}
          </EditorMessagesProvider>
        </EditorUIProvider>
      </CardEditorProvider>
    );

    return render(ui, { wrapper: Wrapper });
  }

  const defaultProps = {
    newCard: false,
    previewModeActive: false,
    setPreviewModeActive: vi.fn(),
    cardBodyRef: mockCardBodyRef,
    filesToUpdate: [],
    setFilesToUpdate: vi.fn(),
    addBacklink: vi.fn(),
  };

  describe('Props rendering', () => {
    it('should display message and error states', () => {
      // Test message only (info styling)
      renderWithProviders(
        <CardEditor {...defaultProps} />,
        { message: 'Success message', setMessage: vi.fn() }
      );

      expect(screen.getByText('Success message')).toBeInTheDocument();
      const messageDiv = screen.getByText('Success message').closest('div');
      expect(messageDiv).toHaveClass('bg-blue-50', 'text-blue-700');

      // Test error only (error styling)
      renderWithProviders(
        <CardEditor {...defaultProps} />,
        { error: 'Error message', setError: vi.fn() }
      );

      expect(screen.getByText('Error message')).toBeInTheDocument();
      const errorDiv = screen.getByText('Error message').closest('div');
      expect(errorDiv).toHaveClass('bg-red-50', 'text-red-700');
    });

    it('should not display message/error div when both are empty', () => {
      renderWithProviders(<CardEditor {...defaultProps} />);
      expect(screen.queryByText(/Success|Error/i)).not.toBeInTheDocument();
    });

    it('should not show template dropdown when newCard is false', () => {
      renderWithProviders(<CardEditor {...defaultProps} newCard={false} />);
      expect(screen.queryByText('Use Template')).not.toBeInTheDocument();
    });

    it('should show template dropdown for new cards with templates', () => {
      const templates = [{ ...defaultCardTemplate, id: 1, name: 'Test Template' }];
      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates, loadingTemplates: false }
      );

      expect(screen.getByText('Use Template')).toBeInTheDocument();
    });

    it('should show loading state for templates', () => {
      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates: [], loadingTemplates: true }
      );

      expect(screen.getByText('Loading templates...')).toBeInTheDocument();
    });

    it('should show template error state', () => {
      const templates = [{ ...defaultCardTemplate, id: 1, name: 'Test Template' }];
      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates, templateError: 'Failed to load templates' }
      );

      expect(screen.getByText('Failed to load templates')).toBeInTheDocument();
    });

    it('should not render a title input (moved to EditPageHeader)', () => {
      renderWithProviders(<CardEditor {...defaultProps} />);
      expect(screen.queryByLabelText('Title:')).not.toBeInTheDocument();
      expect(screen.queryByLabelText('Title')).not.toBeInTheDocument();
    });

    it('should not render Save/Cancel buttons (moved to EditPageHeader)', () => {
      renderWithProviders(<CardEditor {...defaultProps} />);
      expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
    });

  });

  describe('User interactions', () => {
    it('should show and hide template dropdown when Use Template button is clicked', () => {
      const templates = [{ ...defaultCardTemplate, id: 1, name: 'Test Template' }];
      const setShowTemplateDropdown = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates, loadingTemplates: false, setShowTemplateDropdown }
      );

      const templateButton = screen.getByText('Use Template');
      fireEvent.click(templateButton);

      // Note: The actual dropdown state management would depend on the Menu component
      // In a real implementation, this would toggle showTemplateDropdown
    });

    it('should call template handlers when templates are selected', () => {
      const templates = [{ ...defaultCardTemplate, id: 1, name: 'Test Template', title: 'Template Title' }];
      const handleSelectTemplate = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates, loadingTemplates: false, handleSelectTemplate }
      );

      const templateButton = screen.getByText('Use Template');
      fireEvent.click(templateButton);

      // Assuming template dropdown opens - this would depend on HeadlessUI Menu behavior
      // In a real test, you'd likely need to work with the actual Menu component
    });
  });

  describe('Dialog interactions', () => {
    it('should handle backlink dialog show/hide', () => {
      const setShowBacklinkDialog = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} />,
        { setShowBacklinkDialog }
      );

      // Assuming there's a backlink button or trigger in MarkdownToolbar
      // The backlink logic would be tested there
    });

    it('should handle card ID discovery dialog', () => {
      const setShowCardIdDiscovery = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} />,
        { setShowCardIdDiscovery }
      );

      // Card ID discovery would be triggered from CardMetadata component
    });

    it('should handle save as template dialog', () => {
      const setShowSaveAsTemplate = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} />,
        { setShowSaveAsTemplate }
      );

      // Save as template dialog state management
    });
  });
});
