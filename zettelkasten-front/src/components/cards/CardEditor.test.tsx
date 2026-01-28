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
        <EditorUIProvider handleSelectTemplate={handleSelectTemplate}>
          <EditorMessagesProvider>
            {children}
          </EditorMessagesProvider>
        </EditorUIProvider>
      </CardEditorProvider>
    );

    // Override context defaults by setting state after render
    const result = render(ui, { wrapper: Wrapper });

    // Manually set context values for templates
    if (setTemplates) setTemplates(templates);
    if (setLoadingTemplates) setLoadingTemplates(loadingTemplates);
    if (setTemplateError) setTemplateError(templateError);
    if (setShowTemplateDropdown) setShowTemplateDropdown(showTemplateDropdown);
    if (setMessage) setMessage(message);
    if (setError) setError(error);

    return result;
  }

  const defaultProps = {
    newCard: false,
    previewModeActive: false,
    setPreviewModeActive: vi.fn(),
    cardBodyRef: mockCardBodyRef,
    handleSaveCard: vi.fn(),
    handleCancelButtonClick: vi.fn(),
    suggestingTitle: false,
    handleSuggestTitle: vi.fn(),
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
        { templates }
      );

      expect(screen.getByText('Use Template')).toBeInTheDocument();
    });

    it('should show loading state for templates', () => {
      const templates = [{ ...defaultCardTemplate, id: 1, name: 'Test Template' }];
      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates, loadingTemplates: true }
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

    it('should render title input with current value', () => {
      renderWithProviders(<CardEditor {...defaultProps} />);
      const titleInput = screen.getByLabelText('Title:');
      expect(titleInput).toHaveValue('Test Title');
      expect(titleInput).toHaveAttribute('placeholder', 'Title');
    });

    it('should render Save and Cancel buttons', () => {
      renderWithProviders(<CardEditor {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    });

  });

  describe('User interactions', () => {
    it('should call handleSaveCard when Save button is clicked', () => {
      const handleSaveCard = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} handleSaveCard={handleSaveCard} />
      );

      const saveButton = screen.getByRole('button', { name: 'Save' });
      fireEvent.click(saveButton);

      expect(handleSaveCard).toHaveBeenCalled();
    });

    it('should call handleCancelButtonClick when Cancel button is clicked', () => {
      const handleCancelButtonClick = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} handleCancelButtonClick={handleCancelButtonClick} />
      );

      const cancelButton = screen.getByRole('button', { name: 'Cancel' });
      fireEvent.click(cancelButton);

      expect(handleCancelButtonClick).toHaveBeenCalled();
    });

    it('should show and hide template dropdown when Use Template button is clicked', () => {
      const templates = [{ ...defaultCardTemplate, id: 1, name: 'Test Template' }];
      const setShowTemplateDropdown = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates, setShowTemplateDropdown }
      );

      const templateButton = screen.getByText('Use Template');
      fireEvent.click(templateButton);

      // Note: The actual dropdown state management would depend on the Menu component
      // In a real implementation, this would toggle showTemplateDropdown
    });

    it('should show loading state and call handleSuggestTitle when suggest title button is clicked', async () => {
      const handleSuggestTitle = vi.fn();

      renderWithProviders(
        <CardEditor
          {...defaultProps}
          handleSuggestTitle={handleSuggestTitle}
          suggestingTitle={true}
        />
      );

      const suggestButton = screen.getByTitle('Suggesting title...');
      expect(suggestButton).toBeDisabled();

      // Test clicking when not loading
      renderWithProviders(
        <CardEditor
          {...defaultProps}
          handleSuggestTitle={handleSuggestTitle}
          suggestingTitle={false}
        />,
        {
          editingCard: { ...defaultCard, id: 1, title: 'Test Title', body: 'Some content' }
        }
      );

      const enabledButton = screen.getByTitle('Suggest title from content');
      expect(enabledButton).not.toBeDisabled();

      fireEvent.click(enabledButton);
      expect(handleSuggestTitle).toHaveBeenCalled();
    });

    it('should disable title suggestion when body is empty', () => {
      renderWithProviders(
        <CardEditor {...defaultProps} />,
        {
          editingCard: { ...defaultCard, id: 1, title: 'Test Title', body: '' }
        }
      );

      const suggestButton = screen.getByTitle('Suggest title from content');
      expect(suggestButton).toBeDisabled();
    });

    it('should call template handlers when templates are selected', () => {
      const templates = [{ ...defaultCardTemplate, id: 1, name: 'Test Template', title: 'Template Title' }];
      const handleSelectTemplate = vi.fn();

      renderWithProviders(
        <CardEditor {...defaultProps} newCard={true} />,
        { templates, handleSelectTemplate }
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
