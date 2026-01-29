import React, { useState, useEffect, useRef } from "react";
import { isCardIdUnique } from "../../utils/cards";
import { uploadFile } from "../../api/files";
import { parseURL } from "../../api/references";
import { saveNewCard, saveExistingCard, getCard, getNextRootId, getCardReferences, getCardChildren, getCardFiles, getCardTags, getCardTasks, getCardEntities, suggestCardTitle } from "../../api/cards";
import { editFile } from "../../api/files";
import { getTemplates } from "../../api/templates";
import { FileListItem } from "../../components/files/FileListItem";
import { BacklinkDialog } from "../../components/cards/BacklinkDialog";
import { useNavigate, useParams } from "react-router-dom";
import { Card, PartialCard, defaultCard, CardTemplate } from "../../models/Card";
import { File } from "../../models/File";
import { useUIState } from "../../contexts/UIStateContext";
import { Button } from "../../components/Button";
import { ButtonCardDelete } from "../../components/cards/ButtonCardDelete";
import { CardBodyTextArea, CardBodyTextAreaHandle } from "../../components/cards/CardBodyTextArea";
import { processTemplateVariables } from "../../utils/templateVariables";
import { HeaderSubSection } from "../../components/Header";
import { BacklinkInputDropdownList } from "../../components/cards/BacklinkInputDropdownList";



import { useTagContext } from "../../contexts/TagContext";
import { setDocumentTitle } from "../../utils/title";
import { isErrorResponse } from "../../models/common";

// New component imports
import { EditorToolbar } from "../../components/cards/EditorToolbar";
import { CardEditor } from "../../components/cards/CardEditor";
import { CardMetadata } from "../../components/cards/CardMetadata";
import { CardSchemaSection } from "../../components/schemas/CardSchemaSection";

// Editor context imports
import {
  CardEditorProvider,
  EditorUIProvider,
  EditorMessagesProvider,
  useCardEditorContext,
  useEditorUIContext,
  useEditorMessagesContext,
} from "../../contexts/editor";

interface EditPageProps {
  newCard: boolean;
}

function onFileDelete(file_id: number) { }

function renderWarningLabel(cards: PartialCard[], editingCard: Card) {
  if (!editingCard.card_id) return null;
  if (!isCardIdUnique(cards, editingCard.card_id)) {
    return <span className="text-red-600 text-sm font-medium">Card ID is not unique!</span>;
  }
  return null;
}

function EditPageContent({ newCard }: EditPageProps) {
  const [originalCard, setOriginalCard] = useState<Card>(defaultCard);
  const [previewModeActive, setPreviewModeActive] = useState(false);
  const { lastCard, nextCardId, setNextCardId } = useUIState();
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
  } = useEditorUIContext();
  const { message, setMessage, error, setError } = useEditorMessagesContext();

  const [fileFilterString, setFileFilterString] = useState<string>("");
  const { tags } = useTagContext();

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
      setTemplateError("Failed to load templates");
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
        getCardEntities(cardId)
      ]);

      // Combine categorized references into a single array for backward compatibility
      refreshed.references = [...refs.bidirectional, ...refs.outgoing, ...refs.incoming];
      refreshed.children = kids;
      refreshed.files = files;
      refreshed.tags = cardTags;
      refreshed.tasks = tasks;
      refreshed.entities = entities;

      setEditingCard(refreshed);
      setOriginalCard(refreshed);
      setDocumentTitle(refreshed.card_id + " - Edit");
    } catch (err: any) {
      setError((err as Error).message || "Failed to fetch card");
    }
  }

  // clear draft on save
  async function handleSaveCard() {
    try {
      let response;
      if (newCard) {
        response = await saveNewCard(editingCard);
      } else {
        response = await saveExistingCard(editingCard);
      }

      if (!("error" in response)) {
        if (newCard) {
          localStorage.removeItem('newCardBodyDraft');
        }
        filesToUpdate.map((file) =>
          editFile(file["id"].toString(), {
            name: file.name,
            card_pk: response.id,
          }),
        );

        navigate(`/app/card/${response.id}`);
      } else {
        setError("Unable to save card, something has gone wrong.");
      }
    } catch (error: any) {
      console.error('Error saving card:', error);

      // Check for specific error messages from the backend
      let errorMessage = error.message || "Failed to save card. Please try again.";

      if (errorMessage.includes("card_id already exists")) {
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
      setDocumentTitle("New Card");
      const draft = localStorage.getItem('newCardBodyDraft') || "";
      console.log(nextCardId, lastCard?.card_id)
      const cardId = nextCardId || (lastCard ? lastCard.card_id : "");
      setEditingCard({
        ...defaultCard,
        card_id: cardId,
        body: draft,
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
    let text = "";
    if (selectedCard) {
      text = "\n\n[" + selectedCard.card_id + "] - " + selectedCard.title;
    } else {
      text = "";
    }
    setEditingCard((prevEditingCard) => ({
      ...prevEditingCard,
      body: prevEditingCard.body + text,
    }));
  }

  function handleTagClick(tagName: string) {
    setEditingCard((prevEditingCard) => ({
      ...prevEditingCard,
      body: prevEditingCard.body + "\n\n#" + tagName,
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
      console.log("No link provided");
      return;
    }

    try {
      const result = await parseURL(editingCard.link);
      setEditingCard((prev) => ({
        ...prev,
        // Only update title if it's empty/blank
        title:
          !prev.title || prev.title.trim() === "" ? result.title : prev.title,
        // Only update body if it's empty/blank
        body:
          !prev.body || prev.body.trim() === "" ? result.content : prev.body,
      }));
    } catch (error) {
      console.error("Failed to parse URL:", error);
      // Handle error - maybe show a notification to the user
    }
  }

  async function handleDisplayFileOnCardClick(file: File) {
    setEditingCard((prevEditingCard) => ({
      ...prevEditingCard,
      body: prevEditingCard.body + "\n\n![](" + file.id + ")",
    }));
  }

  async function handleSuggestTitle() {
    if (!editingCard.body.trim()) {
      setError("Please add some content to the card body before suggesting a title.");
      return;
    }

    setSuggestingTitle(true);
    setError("");
    setMessage("");

    try {
      const suggestedTitle = await suggestCardTitle(editingCard.body);
      setEditingCard((prevCard) => ({
        ...prevCard,
        title: suggestedTitle,
      }));
      setMessage("Title suggestion applied successfully!");
    } catch (error: any) {
      console.error("Error suggesting title:", error);
      setError(error.message || "Failed to suggest title. Please try again.");
    } finally {
      setSuggestingTitle(false);
    }
  }

  return (

    <div className="pb-10">
      <div className="space-y-6">

        <EditorToolbar
          newCard={newCard}
          originalCard={originalCard}
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
              <div className="md:w-2/3 space-y-6">
                <CardEditor
                  newCard={newCard}
                  previewModeActive={previewModeActive}
                  setPreviewModeActive={setPreviewModeActive}
                  cardBodyRef={cardBodyRef}
                  handleSaveCard={handleSaveCard}
                  handleCancelButtonClick={handleCancelButtonClick}
                  suggestingTitle={suggestingTitle}
                  handleSuggestTitle={handleSuggestTitle}
                  filesToUpdate={filesToUpdate}
                  setFilesToUpdate={setFilesToUpdate}
                  addBacklink={addBacklink}
                />
                <hr className="my-4" />

                <div className="py-2">
                  <HeaderSubSection text="References" />

                  <BacklinkInputDropdownList
                    onSelect={addBacklink}
                    onSearch={() => { }}
                    placeholder="Add Backlink"
                    className="max-w-md"
                    excludeCardId={editingCard.id}
                  />
                </div>
                <hr className="my-4" />
                <div className="space-y-2">

                  <HeaderSubSection text="Link" />
                  <div className="relative">
                    <input
                      type="text"
                      id="link"
                      value={editingCard.link}
                      onChange={(e) =>
                        setEditingCard({ ...editingCard, link: e.target.value })
                      }
                      placeholder="Source"
                      className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm pr-10"
                    />
                    <button
                      onClick={handleClickFillCard}
                      className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded"
                      type="button"
                      title="Fill card from URL"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clipRule="evenodd" />
                      </svg>
                    </button>
                  </div>
                </div>

                {!newCard && (
                  <div className="mt-8">
                    <h4 className="text-lg font-medium text-gray-900 mb-4">Files:</h4>
                    <ul className="divide-y divide-gray-200 border border-gray-200 rounded-md overflow-hidden">
                      {editingCard.files.map((file, index) => (
                        <FileListItem
                          key={index}
                          file={file}
                          onDelete={onFileDelete}
                          setRefreshFiles={(refresh: boolean) => { }}
                          displayFileOnCard={(file: File) =>
                            handleDisplayFileOnCardClick(file)
                          }
                          filterString={fileFilterString}
                          setFilterString={setFileFilterString}
                        />
                      ))}
                    </ul>
                  </div>
                )}
              </div>
              <div className="md:w-1/3 space-y-4">
                <CardMetadata
                  newCard={newCard}
                  originalCard={originalCard}
                  editingCard={editingCard}
                  setEditingCard={setEditingCard}
                  setShowCardIdDiscovery={() => {}}
                  handleClickFillCard={handleClickFillCard}
                  tags={tags}
                  handleTagClick={handleTagClick}
                  handleRemoveTag={handleRemoveTag}
                  addBacklink={addBacklink}
                  setMessage={() => {}}
                />

                <div className="bg-white rounded-lg p-4 shadow-sm">
                  <CardSchemaSection
                    schemaId={editingCard.schema_id}
                    structuredData={editingCard.structured_data}
                    onSchemaChange={(schemaId) =>
                      setEditingCard({ ...editingCard, schema_id: schemaId, structured_data: {} })
                    }
                    onDataChange={(data) =>
                      setEditingCard({ ...editingCard, structured_data: data })
                    }
                  />
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

    </div >
  );
}

export function EditPage({ newCard }: EditPageProps) {
  const { lastCard, nextCardId, setNextCardId } = useUIState();
  const params = useParams();
  const id: string | undefined = params.id;

  // Initialize editingCard state for the provider
  const [editingCard, setEditingCard] = React.useState<Card>(() => {
    if (newCard) {
      const draft = localStorage.getItem('newCardBodyDraft') || "";
      const cardId = nextCardId || (lastCard ? lastCard.card_id : "");
      return {
        ...defaultCard,
        card_id: cardId,
        body: draft,
        process_entities_and_facts: true,
      };
    }
    return defaultCard;
  });

  function handleSelectTemplate(template: CardTemplate) {
    // Process template variables in both title and body
    const processedTitle = processTemplateVariables(template.title);
    const processedBody = processTemplateVariables(template.body);

    setEditingCard(prevCard => ({
      ...prevCard,
      title: processedTitle,
      body: processedBody
    }));
  }

  return (
    <CardEditorProvider editingCard={editingCard} setEditingCard={setEditingCard}>
      <EditorUIProvider handleSelectTemplate={handleSelectTemplate}>
        <EditorMessagesProvider>
          <EditPageContent newCard={newCard} />
        </EditorMessagesProvider>
      </EditorUIProvider>
    </CardEditorProvider>
  );
}
