import React from "react";
import { PartialCard, Card, Entity } from "../../models/Card";
import { CreateTaskWindow } from "../tasks/CreateTaskWindow";
import { QuickSearchWindow } from "../cards/QuickSearchWindow";
import { StarCardDialog } from "../cards/StarCardDialog";
import { EntityDialog } from "../entities/EntityDialog";
import { FactDialog } from "../facts/FactDialog";
import { TaskDialog } from "../tasks/TaskDialog";
import { EditEntityDialog } from "../entities/EditEntityDialog";
import { AddArticleDialog } from "../cards/AddArticleDialog";
import { GettingStartedPage } from "../../pages/GettingStartedPage";

interface SidebarModalsProps {
  showCreateTaskWindow: boolean;
  setShowCreateTaskWindow: (show: boolean) => void;
  showQuickSearchWindow: boolean;
  setShowQuickSearchWindow: (show: boolean) => void;
  showStarCardDialog: boolean;
  setShowStarCardDialog: (show: boolean) => void;
  showEntityDialog: boolean;
  setShowEntityDialog: (show: boolean) => void;
  selectedEntity: Entity | null;
  setSelectedEntity: (entity: Entity | null) => void;
  showFactDialog: boolean;
  setShowFactDialog: (show: boolean) => void;
  showTaskDialog: boolean;
  setShowTaskDialog: (show: boolean) => void;
  selectedTaskId: number | null;
  showEditEntityDialog: boolean;
  setShowEditEntityDialog: (show: boolean) => void;
  entityToEdit: Entity | null;
  setEntityToEdit: (entity: Entity | null) => void;
  showAddArticleDialog: boolean;
  setShowAddArticleDialog: (show: boolean) => void;
  showGettingStarted: boolean;
  setShowGettingStarted: (show: boolean) => void;
  getCurrentCard: () => PartialCard | Card | null;
  setMessage: (message: string) => void;
  handleCloseGettingStarted: () => void;
}

export function SidebarModals({
  showCreateTaskWindow,
  setShowCreateTaskWindow,
  showQuickSearchWindow,
  setShowQuickSearchWindow,
  showStarCardDialog,
  setShowStarCardDialog,
  showEntityDialog,
  setShowEntityDialog,
  selectedEntity,
  setSelectedEntity,
  showFactDialog,
  setShowFactDialog,
  showTaskDialog,
  setShowTaskDialog,
  selectedTaskId,
  showEditEntityDialog,
  setShowEditEntityDialog,
  entityToEdit,
  setEntityToEdit,
  showAddArticleDialog,
  setShowAddArticleDialog,
  showGettingStarted,
  setShowGettingStarted,
  getCurrentCard,
  setMessage,
  handleCloseGettingStarted,
}: SidebarModalsProps) {
  return (
    <>
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={getCurrentCard()}
          setShowTaskWindow={setShowCreateTaskWindow}
        />
      )}

      {showQuickSearchWindow && (
        <QuickSearchWindow setShowWindow={setShowQuickSearchWindow} />
      )}

      {showStarCardDialog && (
        <StarCardDialog
          onClose={() => setShowStarCardDialog(false)}
          onStarSuccess={() => {
            // StarCardDialog refresh - placeholder for now since StarredCardsSection manages its own data
            // The component watches location.pathname changes for refreshes
          }}
          setMessage={setMessage}
        />
      )}

      <EntityDialog
        onClose={() => { setShowEntityDialog(false) }}
        onEdit={(entity) => {
          setEntityToEdit(entity);
          setShowEditEntityDialog(true);
        }}
      />
      <FactDialog
        onClose={() => setShowFactDialog(false)}
        onFactDeleted={() => setShowFactDialog(false)}
      />
      <TaskDialog
        taskId={selectedTaskId}
        isOpen={showTaskDialog}
        onClose={() => setShowTaskDialog(false)}
      />
      <EditEntityDialog
        entity={entityToEdit}
        isOpen={showEditEntityDialog}
        onClose={() => {
          setShowEditEntityDialog(false);
          setEntityToEdit(null);
        }}
        onSuccess={() => {
          // Refresh the entity dialog if it's still open
          if (selectedEntity && entityToEdit && selectedEntity.id === entityToEdit.id) {
            // Force refresh of the entity dialog by toggling it
            setShowEntityDialog(false);
            setTimeout(() => {
              setSelectedEntity(entityToEdit);
              setShowEntityDialog(true);
            }, 100);
          }
        }}
        onDelete={(entity) => {
          // Close edit dialog and entity dialog
          setShowEditEntityDialog(false);
          setShowEntityDialog(false);
          setEntityToEdit(null);
          setSelectedEntity(null);
        }}
      />
      <AddArticleDialog
        show={showAddArticleDialog}
        onClose={() => setShowAddArticleDialog(false)}
        setMessage={setMessage}
      />
      {showGettingStarted && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-[1000]"
          onClick={() => handleCloseGettingStarted()}
        >
          <div
            className="bg-white p-5 rounded-md shadow-lg max-w-4xl w-[90%] max-h-[90vh] overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <GettingStartedPage setShowGettingStarted={setShowGettingStarted} />
          </div>
        </div>
      )}
    </>
  );
}