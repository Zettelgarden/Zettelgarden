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
import { usePartialCardContext } from "../../contexts/CardContext";
import { Button } from "../../components/Button";
import { ButtonCardDelete } from "../../components/cards/ButtonCardDelete";
import { CardBodyTextArea, CardBodyTextAreaHandle } from "../../components/cards/CardBodyTextArea";
import { processTemplateVariables } from "../../utils/templateVariables";

import { useTagContext } from "../../contexts/TagContext";
import { setDocumentTitle } from "../../utils/title";
import { isErrorResponse } from "../../models/common";

// New component imports
import { EditorToolbar } from "../../components/cards/EditorToolbar";
import { CardEditor } from "../../components/cards/CardEditor";
import { CardMetadata } from "../../components/cards/CardMetadata";

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

export function EditPage({ newCard }: EditPageProps) {
  const [error, setError] = useState<string>("");
  const [message, setMessage] = useState<string>("");
  const [originalCard, setOriginalCard] = useState<Card>(defaultCard);
  const [editingCard, setEditingCard] = useState<Card>(defaultCard);
  const [showSaveAsTemplate, setShowSaveAsTemplate] = useState(false);
  const [showBacklinkDialog, setShowBacklinkDialog] = useState(false);
  const [showCardIdDiscovery, setShowCardIdDiscovery] = useState(false);
  const [previewModeActive, setPreviewModeActive] = useState(false); // Added for preview toggle
  const { lastCard, nextCardId, setNextCardId } =
    usePartialCardContext();
  const [filesToUpdate, setFilesToUpdate] = useState<File[]>([]);
  const cardBodyRef = useRef<CardBodyTextAreaHandle>(null);

  // Template selector state
  const [templates, setTemplates] = useState<CardTemplate[]>([]);
  const [loadingTemplates, setLoadingTemplates] = useState(true);
  const [templateError, setTemplateError] = useState("");
  const [showTemplateDropdown, setShowTemplateDropdown] = useState(false);
  const { tags } = useTagContext();

  const [fileFilterString, setFileFilterString] = useState<string>("");
  const [suggestingTitle, setSuggestingTitle] = useState(false);

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

  function handleSelectTemplate(template: CardTemplate) {
    // Process template variables in both title and body
    const processedTitle = processTemplateVariables(template.title);
    const processedBody = processTemplateVariables(template.body);

    setEditingCard(prevCard => ({
      ...prevCard,
      title: processedTitle,
      body: processedBody
    }));
    setMessage("Template applied successfully");
    setShowTemplateDropdown(false);
  }

  async function fetchCard(id: string) {
    try {
      let refreshed = await getCard(id);

      if (isErrorResponse(refreshed)) {
        setError(refreshed.error);
        return;
      }

      const [refs, kids, files, tags, tasks, entities] = await Promise.all([
        getCardReferences(id),
        getCardChildren(id),
        getCardFiles(id),
        getCardTags(id),
        getCardTasks(id),
        getCardEntities(id)
      ]);

      // Combine categorized references into a single array for backward compatibility
      refreshed.references = [...refs.bidirectional, ...refs.outgoing, ...refs.incoming];
      refreshed.children = kids;
      refreshed.files = files;
      refreshed.tags = tags;
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
    } else {
      setDocumentTitle("New Card");
      const draft = localStorage.getItem('newCardBodyDraft') || "";
      setEditingCard({
        ...defaultCard,
        card_id: nextCardId || (lastCard ? lastCard.card_id : ""),
        body: draft,
        process_entities_and_facts: true,
      });
      if (nextCardId) {
        setNextCardId(null);
      }
    }
  }, [id]);

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
      let body =
        // Assuming you have a function to update the card
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
          editingCard={editingCard}
          setEditingCard={setEditingCard}
          setShowSaveAsTemplate={setShowSaveAsTemplate}
          setMessage={setMessage}
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
                  editingCard={editingCard}
                  setEditingCard={setEditingCard}
                  newCard={newCard}
                  message={message}
                  setMessage={setMessage}
                  error={error}
                  setError={setError}
                  templates={templates}
                  loadingTemplates={loadingTemplates}
                  templateError={templateError}
                  handleSelectTemplate={handleSelectTemplate}
                  showTemplateDropdown={showTemplateDropdown}
                  setShowTemplateDropdown={setShowTemplateDropdown}
                  previewModeActive={previewModeActive}
                  setPreviewModeActive={setPreviewModeActive}
                  cardBodyRef={cardBodyRef}
                  handleSaveCard={handleSaveCard}
                  handleCancelButtonClick={handleCancelButtonClick}
                  suggestingTitle={suggestingTitle}
                  handleSuggestTitle={handleSuggestTitle}
                  filesToUpdate={filesToUpdate}
                  setFilesToUpdate={setFilesToUpdate}
                  showSaveAsTemplate={showSaveAsTemplate}
                  showBacklinkDialog={showBacklinkDialog}
                  showCardIdDiscovery={showCardIdDiscovery}
                  setShowBacklinkDialog={setShowBacklinkDialog}
                  setShowSaveAsTemplate={setShowSaveAsTemplate}
                  setShowCardIdDiscovery={setShowCardIdDiscovery}
                  addBacklink={addBacklink}
                />

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
                setMessage={setMessage}
              />
            </div>
          )}
        </div>
      </div>

    </div >
  );
}
