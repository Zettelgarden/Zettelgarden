import React from 'react';
import { Menu } from '@headlessui/react';
import { PartialCard } from '../../models/Card';
import { File } from '../../models/File';
import { CardBodyTextArea, CardBodyTextAreaHandle } from './CardBodyTextArea';
import { MarkdownToolbar } from './MarkdownToolbar';
import { EditorLinkSuggestions } from './EditorLinkSuggestions';
import { BacklinkDialog } from './BacklinkDialog';
import { SaveAsTemplateDialog } from './SaveAsTemplateDialog';
import { CardIdDiscoveryDialog } from './CardIdDiscoveryDialog';
import { CardBodyHelpPopover } from './CardBodyHelpPopover';
import { InsertSchemaTableButton } from './InsertSchemaTableButton';
import { useCardEditorContext } from '../../contexts/editor';
import { useEditorUIContext } from '../../contexts/editor';
import { useEditorMessagesContext } from '../../contexts/editor';

interface CardEditorProps {
  newCard: boolean;
  previewModeActive: boolean;
  setPreviewModeActive: (active: boolean) => void;
  cardBodyRef: React.RefObject<CardBodyTextAreaHandle>;
  filesToUpdate: File[];
  setFilesToUpdate: (files: File[]) => void;
  addBacklink: (selectedCard: PartialCard) => void;
}

export function CardEditor({
  newCard,
  previewModeActive,
  setPreviewModeActive,
  cardBodyRef,
  filesToUpdate,
  setFilesToUpdate,
  addBacklink,
}: CardEditorProps) {
  const { editingCard, setEditingCard } = useCardEditorContext();
  const {
    showSaveAsTemplate,
    setShowSaveAsTemplate,
    showBacklinkDialog,
    setShowBacklinkDialog,
    showCardIdDiscovery,
    setShowCardIdDiscovery,
    templates,
    loadingTemplates,
    templateError,
    showTemplateDropdown,
    setShowTemplateDropdown,
    handleSelectTemplate,
  } = useEditorUIContext();
  const { message, setMessage, error, setError } = useEditorMessagesContext();

  return (
    <>
      {(message || error) && (
        <div
          className={`p-4 rounded-md ${
            error ? 'bg-red-50 text-red-700' : 'bg-blue-50 text-blue-700'
          }`}
        >
          {message || error}
        </div>
      )}

      {newCard &&
        (loadingTemplates || templateError || templates.length > 0) && (
          <div className="">
            {loadingTemplates ? (
              <div className="text-xs text-gray-500">Loading templates...</div>
            ) : templateError ? (
              <div className="text-xs text-red-600">{templateError}</div>
            ) : (
              <Menu as="div" className="relative inline-block">
                <Menu.Button className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 bg-white border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-1 focus:ring-offset-1 focus:ring-blue-500">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-3.5 w-3.5"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                  >
                    <path d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zM3 10a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1v-6zM14 9a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z" />
                  </svg>
                  Use Template
                </Menu.Button>
                <Menu.Items className="absolute z-10 mt-1 w-56 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none max-h-60 overflow-auto">
                  <div className="py-1">
                    {templates.map((template) => (
                      <Menu.Item key={template.id}>
                        {({ active }) => (
                          <button
                            className={`${
                              active
                                ? 'bg-gray-100 text-gray-900'
                                : 'text-gray-700'
                            } w-full text-left px-3 py-1.5 text-sm`}
                            onClick={() => handleSelectTemplate(template)}
                          >
                            {template.name || template.title}
                          </button>
                        )}
                      </Menu.Item>
                    ))}
                  </div>
                </Menu.Items>
              </Menu>
            )}
          </div>
        )}

      <div className="space-y-2">
        <label
          htmlFor="body"
          className="block text-sm font-medium text-gray-700"
        >
          Body:
          <CardBodyHelpPopover />
        </label>
        <MarkdownToolbar
          onFormatText={(formatType) => {
            cardBodyRef.current?.formatText(formatType);
          }}
          onBacklinkClick={() => setShowBacklinkDialog(true)}
          onTogglePreview={() => {
            cardBodyRef.current?.togglePreviewMode();
            setPreviewModeActive(!previewModeActive);
          }}
          isPreviewActive={previewModeActive}
        />
        <InsertSchemaTableButton
          onInsert={(syntax) => {
            setEditingCard({ ...editingCard, body: editingCard.body + syntax });
          }}
        />
        <CardBodyTextArea
          ref={cardBodyRef}
          editingCard={editingCard}
          setEditingCard={(card) => {
            setEditingCard(card);
            console.log('saving', newCard);
            if (newCard) {
              localStorage.setItem('newCardBodyDraft', card.body);
            }
          }}
          setMessage={setMessage}
          newCard={newCard}
          filesToUpdate={filesToUpdate}
          setFilesToUpdate={setFilesToUpdate}
        />

        <EditorLinkSuggestions
          card={editingCard}
          newCard={newCard}
          onInsertLink={(cardId, title) => {
            const link = `[[${cardId}|${title}]]`;
            setEditingCard({
              ...editingCard,
              body: editingCard.body + (editingCard.body ? '\n\n' : '') + link,
            });
          }}
        />
      </div>

      {showBacklinkDialog && (
        <BacklinkDialog
          onClose={() => setShowBacklinkDialog(false)}
          onSelect={addBacklink}
          excludeCardId={editingCard.id}
        />
      )}

      {showSaveAsTemplate && (
        <SaveAsTemplateDialog
          body={editingCard.body}
          title={editingCard.title}
          onClose={() => setShowSaveAsTemplate(false)}
          onSuccess={setMessage}
        />
      )}

      {showCardIdDiscovery && (
        <CardIdDiscoveryDialog
          onClose={() => setShowCardIdDiscovery(false)}
          onSelectId={(cardId) => {
            setEditingCard({ ...editingCard, card_id: cardId });
            setShowCardIdDiscovery(false);
          }}
        />
      )}
    </>
  );
}
