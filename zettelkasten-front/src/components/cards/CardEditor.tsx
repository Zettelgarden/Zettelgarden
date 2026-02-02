import React from "react";
import { Menu } from "@headlessui/react";
import { PartialCard } from "../../models/Card";
import { File } from "../../models/File";
import { CardBodyTextArea, CardBodyTextAreaHandle } from "./CardBodyTextArea";
import { MarkdownToolbar } from "./MarkdownToolbar";
import { BacklinkDialog } from "./BacklinkDialog";
import { SaveAsTemplateDialog } from "./SaveAsTemplateDialog";
import { CardIdDiscoveryDialog } from "./CardIdDiscoveryDialog";
import { CardBodyHelpPopover } from "./CardBodyHelpPopover";
import { Button } from "../Button";
import { useCardEditorContext } from "../../contexts/editor";
import { useEditorUIContext } from "../../contexts/editor";
import { useEditorMessagesContext } from "../../contexts/editor";

interface CardEditorProps {
  newCard: boolean;
  previewModeActive: boolean;
  setPreviewModeActive: (active: boolean) => void;
  cardBodyRef: React.RefObject<CardBodyTextAreaHandle>;
  handleSaveCard: () => void;
  handleCancelButtonClick: () => void;
  suggestingTitle: boolean;
  handleSuggestTitle: () => void;
  filesToUpdate: File[];
  setFilesToUpdate: (files: File[]) => void;
  addBacklink: (selectedCard: PartialCard) => void;
}

export function CardEditor({
  newCard,
  previewModeActive,
  setPreviewModeActive,
  cardBodyRef,
  handleSaveCard,
  handleCancelButtonClick,
  suggestingTitle,
  handleSuggestTitle,
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
        <div className={`p-4 rounded-md ${error ? 'bg-red-50 text-red-700' : 'bg-blue-50 text-blue-700'}`}>
          {message || error}
        </div>
      )}

      {newCard && (loadingTemplates || templateError || templates.length > 0) && (
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
                            active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
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

      <div className="">
        <label htmlFor="title" className="block text-sm font-medium text-gray-700">
          Title:
        </label>
        <div className="relative">
          <input
            type="text"
            id="title"
            value={editingCard.title}
            onChange={(e) =>
              setEditingCard({ ...editingCard, title: e.target.value })
            }
            placeholder="Title"
            className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm pr-10"
          />
          <button
            onClick={handleSuggestTitle}
            disabled={suggestingTitle || !editingCard.body.trim()}
            className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded disabled:text-gray-400 disabled:cursor-not-allowed disabled:hover:bg-transparent"
            type="button"
            title={suggestingTitle ? "Suggesting title..." : "Suggest title from content"}
          >
            {suggestingTitle ? (
              <svg className="animate-spin h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            ) : (
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
              </svg>
            )}
          </button>
        </div>
      </div>

      <div className="space-y-2">
        <label htmlFor="body" className="block text-sm font-medium text-gray-700">
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
        <CardBodyTextArea
          ref={cardBodyRef}
          editingCard={editingCard}
          setEditingCard={(card) => {
            setEditingCard(card);
            console.log("saving", newCard);
            if (newCard) {
              localStorage.setItem('newCardBodyDraft', card.body);
            }
          }}
          setMessage={setMessage}
          newCard={newCard}
          filesToUpdate={filesToUpdate}
          setFilesToUpdate={setFilesToUpdate}
        />
      </div>

      <div className="flex flex-wrap gap-3 pt-4">
        <Button onClick={handleSaveCard} variant="primary">Save</Button>
        <Button onClick={handleCancelButtonClick} variant="outline">Cancel</Button>
      </div>

      {showBacklinkDialog && (
        <BacklinkDialog
          onClose={() => setShowBacklinkDialog(false)}
          onSelect={addBacklink}
          setMessage={setMessage}
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
