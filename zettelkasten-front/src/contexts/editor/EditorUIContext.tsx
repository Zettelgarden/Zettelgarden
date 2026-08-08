import React, { createContext, useContext, useState } from 'react';
import { CardTemplate, PartialCard } from '../../models/Card';

interface EditorUIContextValue {
  // Dialog visibility states
  showSaveAsTemplate: boolean;
  setShowSaveAsTemplate: (show: boolean) => void;
  showBacklinkDialog: boolean;
  setShowBacklinkDialog: (show: boolean) => void;
  showCardIdDiscovery: boolean;
  setShowCardIdDiscovery: (show: boolean) => void;
  // Template states
  templates: CardTemplate[];
  setTemplates: (templates: CardTemplate[]) => void;
  loadingTemplates: boolean;
  setLoadingTemplates: (loading: boolean) => void;
  templateError: string;
  setTemplateError: (error: string) => void;
  showTemplateDropdown: boolean;
  setShowTemplateDropdown: (show: boolean) => void;
  // Handler for template selection (passed from parent)
  handleSelectTemplate: (template: CardTemplate) => void;
}

const EditorUIContext = createContext<EditorUIContextValue | undefined>(
  undefined,
);

export function useEditorUIContext() {
  const context = useContext(EditorUIContext);
  if (!context) {
    throw new Error('useEditorUIContext must be used within EditorUIProvider');
  }
  return context;
}

interface EditorUIProviderProps {
  children: React.ReactNode;
  handleSelectTemplate: (template: CardTemplate) => void;
  // Optional initial values for testing
  initialTemplates?: CardTemplate[];
  initialLoadingTemplates?: boolean;
  initialTemplateError?: string;
}

export function EditorUIProvider({
  children,
  handleSelectTemplate,
  initialTemplates = [],
  initialLoadingTemplates = true,
  initialTemplateError = '',
}: EditorUIProviderProps) {
  // Dialog states
  const [showSaveAsTemplate, setShowSaveAsTemplate] = useState(false);
  const [showBacklinkDialog, setShowBacklinkDialog] = useState(false);
  const [showCardIdDiscovery, setShowCardIdDiscovery] = useState(false);

  // Template states
  const [templates, setTemplates] = useState<CardTemplate[]>(initialTemplates);
  const [loadingTemplates, setLoadingTemplates] = useState(
    initialLoadingTemplates,
  );
  const [templateError, setTemplateError] = useState(initialTemplateError);
  const [showTemplateDropdown, setShowTemplateDropdown] = useState(false);

  return (
    <EditorUIContext.Provider
      value={{
        // Dialog states
        showSaveAsTemplate,
        setShowSaveAsTemplate,
        showBacklinkDialog,
        setShowBacklinkDialog,
        showCardIdDiscovery,
        setShowCardIdDiscovery,
        // Template states
        templates,
        setTemplates,
        loadingTemplates,
        setLoadingTemplates,
        templateError,
        setTemplateError,
        showTemplateDropdown,
        setShowTemplateDropdown,
        handleSelectTemplate,
      }}
    >
      {children}
    </EditorUIContext.Provider>
  );
}
