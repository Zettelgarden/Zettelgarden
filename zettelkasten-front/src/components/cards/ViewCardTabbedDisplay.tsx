import React, { useState, useEffect } from "react";
import { Card, Entity } from "../../models/Card";
import { File } from "../../models/File";
import { removeEntityFromCard } from "../../api/entities";
import {
  saveExistingCard,
  getCardAuditEvents,
  restoreCardToAuditEvent,
} from "../../api/cards";

// Import tab components
import { EntitiesTab } from "../tabs/EntitiesTab";
import { HistoryTab } from "../tabs/HistoryTab";
import { SummariesTab } from "../tabs/SummariesTab";
import { FilesTab } from "../tabs/FilesTab";
import { RollbackConfirmDialog } from "../tabs/RollbackConfirmDialog";

// Props interface
import { SummarizeJobResponse } from "../../api/summarizer";

interface ViewCardTabbedDisplayProps {
  viewingCard: Card;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
  handleOpenEntity: (entity: Entity) => void;
  summaries: SummarizeJobResponse[] | null;
  fileUploadRef: React.RefObject<HTMLInputElement>;
}


export function ViewCardTabbedDisplay({
  viewingCard,
  setViewCard,
  setError,
  handleOpenEntity,
  summaries,
  fileUploadRef,
}: ViewCardTabbedDisplayProps) {
  const [activeTab, setActiveTab] = useState<string>("Entities");
  const [auditEvents, setAuditEvents] = useState<any[]>([]);
  const [fileFilterString, setFileFilterString] = useState<string>("");
  const [entityFilterString, setEntityFilterString] = useState<string>("");
  const [showAddEntityDialog, setShowAddEntityDialog] = useState<boolean>(false);
  const [showRollbackDialog, setShowRollbackDialog] = useState<boolean>(false);
  const [pendingRestoreEvent, setPendingRestoreEvent] = useState<any>(null);
  const [isRestoring, setIsRestoring] = useState<boolean>(false);

  const tabs = [
    { label: "Entities" },
    { label: "History" },
    { label: "Summaries" },
    { label: "Files" },
  ];


  function handleTabClick(label: string) {
    setActiveTab(label);
  }

  async function handleDisplayFileOnCardClick(file: File) {
    if (viewingCard === null) {
      return;
    }

    let editedCard = {
      ...viewingCard,
      body: viewingCard.body + "\n\n![](" + file.id + ")",
    };
    let response = await saveExistingCard(editedCard);
    setViewCard(editedCard);
  }

  async function handleRemoveEntity(entityId: number) {
    try {
      await removeEntityFromCard(entityId, viewingCard.id);
      // Update the viewingCard by removing the entity
      setViewCard({
        ...viewingCard,
        entities: viewingCard.entities?.filter(entity => entity.id !== entityId) || []
      });
    } catch (error) {
      setError("Failed to remove entity from card");
    }
  }

  function handleEntityAdded(entity: Entity) {
    // Update the viewingCard by adding the entity
    setViewCard({
      ...viewingCard,
      entities: [...(viewingCard.entities || []), entity]
    });
  }

  useEffect(() => {
    if (activeTab === "History") {
      getCardAuditEvents(viewingCard.id.toString())
        .then(events => setAuditEvents(events))
        .catch(error => setError("Failed to load audit events"));
    }
  }, [activeTab, viewingCard.id]);

  // Handle restore click - show confirmation dialog
  const handleRestoreClick = (event: any) => {
    setPendingRestoreEvent(event);
    setShowRollbackDialog(true);
  };

  // Confirm restore - call the API
  const handleConfirmRestore = async () => {
    if (!pendingRestoreEvent) return;

    setIsRestoring(true);
    try {
      const restoredCard = await restoreCardToAuditEvent(
        viewingCard.id.toString(),
        pendingRestoreEvent.id
      );
      setViewCard(restoredCard);
      // Refresh audit events to show the new restore event
      const events = await getCardAuditEvents(viewingCard.id.toString());
      setAuditEvents(events);
      setShowRollbackDialog(false);
      setPendingRestoreEvent(null);
    } catch (error) {
      setError("Failed to restore card");
    } finally {
      setIsRestoring(false);
    }
  };

  // Cancel restore
  const handleCancelRestore = () => {
    setShowRollbackDialog(false);
    setPendingRestoreEvent(null);
  };


  return (
    <div>
      <div className="flex flex-wrap">
        {tabs.map((tab) => (
          <span
            key={tab.label}
            onClick={() => handleTabClick(tab.label)}
            className={`
            cursor-pointer font-medium py-1 px-2 rounded-md flex items-center text-sm
            ${activeTab === tab.label
                ? "text-blue-600 border-b-2 border-blue-600"
                : "text-gray-600 hover:text-gray-800 hover:bg-gray-100"
              }
          `}
          >
            {tab.label}
            {tab.label !== "History" &&
              <span className="ml-1 text-xs font-semibold bg-gray-200 rounded-full px-1.5 py-0.5 text-gray-700">
                {tab.label === "Files" && viewingCard.files.length}
                {tab.label === "Entities" && viewingCard.entities && viewingCard.entities.length}
                {tab.label === "Summaries" && summaries && summaries.length}
              </span>
            }
          </span>
        ))}
      </div>

      {activeTab === "Files" && (
        <FilesTab
          viewingCard={viewingCard}
          fileUploadRef={fileUploadRef}
          handleDisplayFileOnCardClick={handleDisplayFileOnCardClick}
          fileFilterString={fileFilterString}
          setFileFilterString={setFileFilterString}
          setError={setError}
        />
      )}
      {activeTab === "Entities" && (
        <EntitiesTab
          viewingCard={viewingCard}
          entityFilterString={entityFilterString}
          setEntityFilterString={setEntityFilterString}
          showAddEntityDialog={showAddEntityDialog}
          setShowAddEntityDialog={setShowAddEntityDialog}
          handleOpenEntity={handleOpenEntity}
          handleRemoveEntity={handleRemoveEntity}
          handleEntityAdded={handleEntityAdded}
          setError={setError}
        />
      )}
      {activeTab === "History" && (
        <HistoryTab auditEvents={auditEvents} onRestore={handleRestoreClick} />
      )}
      {activeTab === "Summaries" && (
        <SummariesTab summaries={summaries} />
      )}

      {/* Rollback Confirmation Dialog */}
      <RollbackConfirmDialog
        isOpen={showRollbackDialog}
        onClose={handleCancelRestore}
        onConfirm={handleConfirmRestore}
        cardTitle={viewingCard.title || viewingCard.card_id || 'Untitled Card'}
        auditEvent={pendingRestoreEvent}
        isLoading={isRestoring}
      />

    </div>
  );
}
