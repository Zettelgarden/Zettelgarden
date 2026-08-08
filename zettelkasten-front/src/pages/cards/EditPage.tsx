import React, { useState, useEffect, useRef } from 'react';
import { MobileTopBar } from '../../components/layout/MobileTopBar';
import { isCardIdUnique } from '../../utils/cards';
import { uploadFile } from '../../api/files';
import { parseURL } from '../../api/references';
import {
  saveNewCard,
  saveExistingCard,
  getCard,
  getNextRootId,
  getCardReferences,
  getCardChildren,
  getCardFiles,
  getCardTags,
  getCardTasks,
  getCardEntities,
  suggestCardTitle,
} from '../../api/cards';
import { editFile } from '../../api/files';
import { getTemplates } from '../../api/templates';
import { FileListItem } from '../../components/files/FileListItem';
import { BacklinkDialog } from '../../components/cards/BacklinkDialog';
import { CardIdDiscoveryDialog } from '../../components/cards/CardIdDiscoveryDialog';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  Card,
  PartialCard,
  defaultCard,
  CardTemplate,
} from '../../models/Card';
import { File } from '../../models/File';
import { useUIState, RightPaneTab } from '../../contexts/UIStateContext';
import { Button } from '../../components/Button';
import {
  CardBodyTextArea,
  CardBodyTextAreaHandle,
} from '../../components/cards/CardBodyTextArea';
import { processTemplateVariables } from '../../utils/templateVariables';
import { useRightPaneTab } from '../../hooks/useRightPaneTab';

import { useTagContext } from '../../contexts/TagContext';
import { setDocumentTitle } from '../../utils/title';
import { isErrorResponse } from '../../models/common';
import { getMissingRequiredFields } from '../../utils/schemaValidation';
import { fetchSchema } from '../../api/schemas';

// New component imports
import { EditPageHeader } from '../../components/cards/EditPageHeader';
import { CardEditor } from '../../components/cards/CardEditor';
import { CardMetadata } from '../../components/cards/CardMetadata';
import { CardSchemaSection } from '../../components/schemas/CardSchemaSection';

// Editor context imports
import {
  CardEditorProvider,
  EditorUIProvider,
  EditorMessagesProvider,
  useCardEditorContext,
  useEditorUIContext,
  useEditorMessagesContext,
} from '../../contexts/editor';

interface EditPageProps {
  newCard: boolean;
}

// Read the optional ?schema= query param used by the schema table's
// "Add Card" action to pre-attach a schema to a new card.
function getNewCardSchemaId(
  searchParams: URLSearchParams,
  newCard: boolean,
): number | undefined {
  if (!newCard) return undefined;
  const raw = searchParams.get('schema');
  if (!raw) return undefined;
  const parsed = parseInt(raw, 10);
  return Number.isNaN(parsed) ? undefined : parsed;
}

// Edit rail tabs. Metadata holds Card ID/Tags/Schema/Details; Links (the
// backlink input) arrives in PR 3.
const EDIT_TABS: { id: RightPaneTab; label: string }[] = [
  { id: 'metadata', label: 'Metadata' },
  { id: 'links', label: 'Links' },
  { id: 'files', label: 'Files' },
];

function onFileDelete(file_id: number) {}

function renderWarningLabel(cards: PartialCard[], editingCard: Card) {
  if (!editingCard.card_id) return null;
  if (!isCardIdUnique(cards, editingCard.card_id)) {
    return (
      <span className="text-red-600 text-sm font-medium">
        Card ID is not unique!
      </span>
    );
  }
  return null;
}

function EditPageContent({ newCard }: EditPageProps) {
  const [searchParams] = useSearchParams();
  const [originalCard, setOriginalCard] = useState<Card>(defaultCard);
  const [previewModeActive, setPreviewModeActive] = useState(false);
  const {
    lastCard,
    nextCardId,
    setNextCardId,
    toggleMobileSidebar,
    rightPaneOpen,
    toggleRightPane,
    rightPaneTab,
    setRightPaneTab,
  } = useUIState();
  const [filesToUpdate, setFilesToUpdate] = useState<File[]>([]);
  const cardBodyRef = useRef<CardBodyTextAreaHandle>(null);
  const [suggestingTitle, setSuggestingTitle] = useState(false);
  const hasInitializedRef = useRef(false);

  // Use contexts for shared state
  const { editingCard, setEditingCard } = useCardEditorContext();
  const {
    templates,
    setTemplates,
    setLoadingTemplates,
    setTemplateError,
    showCardIdDiscovery,
    setShowCardIdDiscovery,
  } = useEditorUIContext();
  const { message, setMessage, error, setError } = useEditorMessagesContext();

  const [fileFilterString, setFileFilterString] = useState<string>('');
  const { tags } = useTagContext();

  // Edit rail defaults to Metadata (an editor almost always wants Card ID/Tags
  // first); the hook shares ?pane= sync with the view page.
  useRightPaneTab({ hasRelationships: false });

  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  // Fetch templates
  useEffect(() => {
    if (newCard) {
      fetchTemplates();
    }
  }, [newCard]);

  async function fetchTemplates() {
    try {
      const fetchedTemplates = await getTemplates();
      setTemplates(fetchedTemplates);
      setLoadingTemplates(false);
    } catch (err) {
      setTemplateError('Failed to load templates');
      setLoadingTemplates(false);
    }
  }

  async function fetchCard(cardId: string) {
    try {
      let refreshed = await getCard(cardId);

      if (isErrorResponse(refreshed)) {
        setError(refreshed.error);
        return;
      }

      const [refs, kids, files, cardTags, tasks, entities] = await Promise.all([
        getCardReferences(cardId),
        getCardChildren(cardId),
        getCardFiles(cardId),
        getCardTags(cardId),
        getCardTasks(cardId),
        getCardEntities(cardId),
      ]);

      // Combine categorized references into a single array for backward compatibility
      refreshed.references = [
        ...refs.bidirectional,
        ...refs.outgoing,
        ...refs.incoming,
      ];
      refreshed.children = kids;
      refreshed.files = files;
      refreshed.tags = cardTags;
      refreshed.tasks = tasks;
      refreshed.entities = entities;

      setEditingCard(refreshed);
      setOriginalCard(refreshed);
      setDocumentTitle(refreshed.card_id + ' - Edit');
    } catch (err: any) {
      setError((err as Error).message || 'Failed to fetch card');
    }
  }

  // clear draft on save
  async function handleSaveCard() {
    // Enforce required schema fields before save (bead s2l) — mirrors the
    // backend rule so users get inline feedback, not a server error.
    if (editingCard.schema_id) {
      try {
        const schema = await fetchSchema(editingCard.schema_id);
        const missing = getMissingRequiredFields(
          schema.fields,
          editingCard.structured_data,
        );
        if (missing.length > 0) {
          setError(
            `Please fill required field${
              missing.length > 1 ? 's' : ''
            }: ${missing.join(', ')}`,
          );
          return;
        }
      } catch (err) {
        console.error('Failed to load schema for validation:', err);
        // Fall through to the server — it enforces the same rule.
      }
    }
    try {
      let response;
      if (newCard) {
        response = await saveNewCard(editingCard);
      } else {
        response = await saveExistingCard(editingCard);
      }

      if (!('error' in response)) {
        if (newCard) {
          localStorage.removeItem('newCardBodyDraft');
        }
        filesToUpdate.map((file) =>
          editFile(file['id'].toString(), {
            name: file.name,
            card_pk: response.id,
          }),
        );

        navigate(`/app/card/${response.id}`);
      } else {
        setError('Unable to save card, something has gone wrong.');
      }
    } catch (error: any) {
      console.error('Error saving card:', error);

      // Check for specific error messages from the backend
      let errorMessage =
        error.message || 'Failed to save card. Please try again.';

      if (errorMessage.includes('card_id already exists')) {
        errorMessage = `The Card ID "${editingCard.card_id}" is already in use by another card. Please choose a different ID.`;
      }

      setError(errorMessage);
    }
  }

  // on mount, restore draft if newCard
  useEffect(() => {
    if (!newCard) {
      fetchCard(id!);
    } else if (!hasInitializedRef.current) {
      setDocumentTitle('New Card');
      const draft = localStorage.getItem('newCardBodyDraft') || '';
      console.log(nextCardId, lastCard?.card_id);
      const cardId = nextCardId || (lastCard ? lastCard.card_id : '');
      setEditingCard({
        ...defaultCard,
        card_id: cardId,
        body: draft,
        schema_id: getNewCardSchemaId(searchParams, newCard),
        process_entities_and_facts: true,
      });
      hasInitializedRef.current = true;
      if (nextCardId) {
        setNextCardId(null);
      }
    }
  }, [id, newCard, nextCardId, lastCard, setNextCardId, setEditingCard]);

  // clear draft on cancel
  function handleCancelButtonClick() {
    if (newCard) {
      localStorage.removeItem('newCardBodyDraft');
      console.log(lastCard);
      if (lastCard) {
        navigate(`/app/card/${lastCard.id}`);
      } else {
        navigate(`/`);
      }
    } else {
      if (editingCard.id) {
        navigate(`/app/card/${editingCard.id}`);
      } else {
        navigate(`/`);
      }
    }
  }

  function addBacklink(selectedCard: PartialCard) {
    let text = '';
    if (selectedCard) {
      text = '\n\n[[' + selectedCard.card_id + '|*|]]';
    } else {
      text = '';
    }
    setEditingCard((prevEditingCard) => ({
      ...prevEditingCard,
      body: prevEditingCard.body + text,
    }));
  }

  function handleTagClick(tagName: string) {
    setEditingCard((prevEditingCard) => ({
      ...prevEditingCard,
      body: prevEditingCard.body + '\n\n#' + tagName,
    }));
  }

  function handleRemoveTag(tagName: string) {
    const tagRegex = new RegExp(`\\n*#${tagName}\\b`, 'g');
    setEditingCard((prevEditingCard) => ({
      ...prevEditingCard,
      body: prevEditingCard.body.replace(tagRegex, ''),
    }));
  }

  async function handleClickFillCard() {
    if (!editingCard.link) {
      // Handle case where there's no link
      console.log('No link provided');
      return;
    }

    try {
      const result = await parseURL(editingCard.link);
      setEditingCard((prev) => ({
        ...prev,
        // Only update title if it's empty/blank
        title:
          !prev.title || prev.title.trim() === '' ? result.title : prev.title,
        // Only update body if it's empty/blank
        body:
          !prev.body || prev.body.trim() === '' ? result.content : prev.body,
      }));
    } catch (error) {
      console.error('Failed to parse URL:', error);
      // Handle error - maybe show a notification to the user
    }
  }

  async function handleDisplayFileOnCardClick(file: File) {
    setEditingCard((prevEditingCard) => ({
      ...prevEditingCard,
      body: prevEditingCard.body + '\n\n![](' + file.id + ')',
    }));
  }

  async function handleSuggestTitle() {
    if (!editingCard.body.trim()) {
      setError(
        'Please add some content to the card body before suggesting a title.',
      );
      return;
    }

    setSuggestingTitle(true);
    setError('');
    setMessage('');

    try {
      const suggestedTitle = await suggestCardTitle(editingCard.body);
      setEditingCard((prevCard) => ({
        ...prevCard,
        title: suggestedTitle,
      }));
      setMessage('Title suggestion applied successfully!');
    } catch (error: any) {
      console.error('Error suggesting title:', error);
      setError(error.message || 'Failed to suggest title. Please try again.');
    } finally {
      setSuggestingTitle(false);
    }
  }

  return (
    <div className="pb-10">
      {editingCard && (
        <MobileTopBar
          title={editingCard.title || (newCard ? 'New Card' : 'Edit Card')}
          onMenuClick={toggleMobileSidebar}
        />
      )}
      <div className="space-y-6">
        <EditPageHeader
          newCard={newCard}
          originalCard={originalCard}
          suggestingTitle={suggestingTitle}
          handleSuggestTitle={handleSuggestTitle}
          handleSaveCard={handleSaveCard}
          handleCancelButtonClick={handleCancelButtonClick}
          onDeleteSuccess={() => {
            if (lastCard && lastCard.id !== editingCard.id) {
              navigate(`/app/card/${lastCard.id}`);
            } else {
              navigate('/');
            }
          }}
        />

        <div className="">
          {editingCard && (
            <div className="flex flex-col md:flex-row gap-4 px-4">
              <div
                className={`${rightPaneOpen ? 'md:w-2/3' : 'w-full'} space-y-6`}
              >
                <CardEditor
                  newCard={newCard}
                  previewModeActive={previewModeActive}
                  setPreviewModeActive={setPreviewModeActive}
                  cardBodyRef={cardBodyRef}
                  filesToUpdate={filesToUpdate}
                  setFilesToUpdate={setFilesToUpdate}
                  addBacklink={addBacklink}
                />
              </div>
              {rightPaneOpen && (
                <div className="md:w-1/3">
                  {/* Tab strip + close affordance */}
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex flex-wrap">
                      {EDIT_TABS.map((tab) => (
                        <span
                          key={tab.id}
                          onClick={() => setRightPaneTab(tab.id)}
                          className={`
                            cursor-pointer font-medium py-1 px-2 flex items-center text-sm
                            ${
                              rightPaneTab === tab.id
                                ? 'text-blue-600 border-b-2 border-blue-600'
                                : 'text-gray-600 hover:text-gray-800 hover:bg-gray-100 rounded-md'
                            }
                          `}
                        >
                          {tab.label}
                        </span>
                      ))}
                    </div>
                    <button
                      type="button"
                      onClick={toggleRightPane}
                      title="Close info pane"
                      aria-label="Close info pane"
                      className="p-1 rounded-md text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors"
                    >
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M6 18L18 6M6 6l12 12"
                        />
                      </svg>
                    </button>
                  </div>

                  <div className="space-y-4">
                    {rightPaneTab === 'links' && (
                      <CardMetadata
                        newCard={newCard}
                        originalCard={originalCard}
                        editingCard={editingCard}
                        setEditingCard={setEditingCard}
                        setShowCardIdDiscovery={setShowCardIdDiscovery}
                        handleClickFillCard={handleClickFillCard}
                        tags={tags}
                        handleTagClick={handleTagClick}
                        handleRemoveTag={handleRemoveTag}
                        addBacklink={addBacklink}
                        tab="links"
                      />
                    )}

                    {rightPaneTab === 'metadata' && (
                      <>
                        <CardMetadata
                          newCard={newCard}
                          originalCard={originalCard}
                          editingCard={editingCard}
                          setEditingCard={setEditingCard}
                          setShowCardIdDiscovery={setShowCardIdDiscovery}
                          handleClickFillCard={handleClickFillCard}
                          tags={tags}
                          handleTagClick={handleTagClick}
                          handleRemoveTag={handleRemoveTag}
                          addBacklink={addBacklink}
                          tab="metadata"
                        />

                        <div className="bg-white rounded-lg p-4 shadow-sm">
                          <CardSchemaSection
                            schemaId={editingCard.schema_id}
                            structuredData={editingCard.structured_data}
                            onSchemaChange={(schemaId) =>
                              setEditingCard({
                                ...editingCard,
                                schema_id: schemaId,
                                structured_data: {},
                              })
                            }
                            onDataChange={(data) =>
                              setEditingCard({
                                ...editingCard,
                                structured_data: data,
                              })
                            }
                          />
                        </div>
                      </>
                    )}

                    {rightPaneTab === 'files' && (
                      <div>
                        <h4 className="text-sm font-medium text-gray-900 mb-3">
                          Files ({editingCard.files.length})
                        </h4>
                        {editingCard.files.length > 0 ? (
                          <ul className="divide-y divide-gray-200 border border-gray-200 rounded-md overflow-hidden">
                            {editingCard.files.map((file, index) => (
                              <FileListItem
                                key={index}
                                file={file}
                                onDelete={onFileDelete}
                                setRefreshFiles={(refresh: boolean) => {}}
                                displayFileOnCard={(file: File) =>
                                  handleDisplayFileOnCardClick(file)
                                }
                                filterString={fileFilterString}
                                setFilterString={setFileFilterString}
                              />
                            ))}
                          </ul>
                        ) : (
                          <p className="text-sm text-gray-400">
                            No files attached.
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {showCardIdDiscovery && (
        <CardIdDiscoveryDialog
          onClose={() => setShowCardIdDiscovery(false)}
          onSelectId={(cardId) => {
            setEditingCard({ ...editingCard, card_id: cardId });
            setShowCardIdDiscovery(false);
          }}
        />
      )}
    </div>
  );
}

export function EditPage({ newCard }: EditPageProps) {
  const { lastCard, nextCardId, setNextCardId } = useUIState();
  const params = useParams();
  const [searchParams] = useSearchParams();
  const id: string | undefined = params.id;

  // Initialize editingCard state for the provider
  const [editingCard, setEditingCard] = React.useState<Card>(() => {
    if (newCard) {
      const draft = localStorage.getItem('newCardBodyDraft') || '';
      const cardId = nextCardId || (lastCard ? lastCard.card_id : '');
      return {
        ...defaultCard,
        card_id: cardId,
        body: draft,
        schema_id: getNewCardSchemaId(searchParams, newCard),
        process_entities_and_facts: true,
      };
    }
    return defaultCard;
  });

  function handleSelectTemplate(template: CardTemplate) {
    // Process template variables in both title and body
    const processedTitle = processTemplateVariables(template.title);
    const processedBody = processTemplateVariables(template.body);

    setEditingCard((prevCard) => ({
      ...prevCard,
      title: processedTitle,
      body: processedBody,
    }));
  }

  return (
    <CardEditorProvider
      editingCard={editingCard}
      setEditingCard={setEditingCard}
    >
      <EditorUIProvider handleSelectTemplate={handleSelectTemplate}>
        <EditorMessagesProvider>
          <EditPageContent newCard={newCard} />
        </EditorMessagesProvider>
      </EditorUIProvider>
    </CardEditorProvider>
  );
}
