import React, { useState } from "react";
import { Dialog } from "@headlessui/react";
import { useNavigate } from "react-router-dom";
import { Entity } from "../../models/Card";
import { mergeEntities, updateEntity, UpdateEntityRequest, addEntityToCard } from "../../api/entities";
import { useDialogState } from "../../contexts/DialogStateContext";
import { CreateCardDialog } from "../cards/CreateCardDialog";
import { useEntityData } from "../../hooks/useEntityData";
import { EntityHeader } from "./EntityHeader";
import { EntityCardsSection } from "./EntityCardsSection";
import { EntitySimilarSection } from "./EntitySimilarSection";
import { EntityMergeDialog } from "./EntityMergeDialog";

interface EntityDialogProps {
    onClose: () => void;
    onEdit?: (entity: Entity, event: React.MouseEvent) => void;
}

export function EntityDialog({ onClose, onEdit }: EntityDialogProps) {
    const [showConfirmDialog, setShowConfirmDialog] = useState(false);
    const [entityToMerge, setEntityToMerge] = useState<Entity | null>(null);
    const [isMerging, setIsMerging] = useState(false);
    const [mergeError, setMergeError] = useState<string | null>(null);
    const [mergeDirection, setMergeDirection] = useState<'into-current' | 'from-current'>('into-current');
    const [showConvertDialog, setShowConvertDialog] = useState(false);
    const [cardTitle, setCardTitle] = useState("");
    const [cardBody, setCardBody] = useState("");

    const navigate = useNavigate();

    const {
        showEntityDialog,
        setShowEntityDialog,
        selectedEntity,
        setSelectedEntity,
    } = useDialogState();

    const entityData = useEntityData(showEntityDialog, selectedEntity);

    function handleEntityClick(entity: Entity) {
        setSelectedEntity(entity);
        setShowEntityDialog(true);
    }

    function handleInitiateMerge(entity: Entity, direction: 'into-current' | 'from-current') {
        setMergeDirection(direction);
        setEntityToMerge(entity);
        setShowConfirmDialog(true);
    }

    async function handleConfirmMerge() {
        if (!selectedEntity || !entityToMerge) return;

        setIsMerging(true);
        setMergeError(null);

        const primaryEntityId = mergeDirection === 'into-current' ? selectedEntity.id : entityToMerge.id;
        const secondaryEntityId = mergeDirection === 'into-current' ? entityToMerge.id : selectedEntity.id;

        try {
            await mergeEntities(primaryEntityId, secondaryEntityId);

            if (mergeDirection === 'from-current') {
                // The current entity was merged, so we need to update the dialog to show the new primary entity
                setSelectedEntity(entityToMerge);
            }
            // Note: similar entities will be refreshed on next dialog open

            setShowConfirmDialog(false);
            setEntityToMerge(null);
        } catch (err) {
            setMergeError("Failed to merge entities");
            console.error(err);
        } finally {
            setIsMerging(false);
        }
    }

    const handleEditClick = () => {
        if (selectedEntity && onEdit) {
            onEdit(selectedEntity, {} as React.MouseEvent);
            onClose();
        }
    };

    const handleTurnIntoCard = () => {
        if (!selectedEntity) return;
        const truncatedTitle = selectedEntity.name.length > 50
            ? selectedEntity.name.slice(0, 50) + "..."
            : selectedEntity.name;
        setCardTitle(truncatedTitle);
        setCardBody(selectedEntity.description || selectedEntity.name);
        setShowConvertDialog(true);
    };

    const handleCardCreated = async (newCardId: number) => {
        if (!selectedEntity) return;

        try {
            // Update the entity to link it to the new card
            const updateRequest: UpdateEntityRequest = {
                name: selectedEntity.name,
                description: selectedEntity.description,
                type: selectedEntity.type,
                card_pk: newCardId,
            };

            await updateEntity(selectedEntity.id, updateRequest);

            // Link the entity to the card
            await addEntityToCard(selectedEntity.id, newCardId);

            // Update the selectedEntity to reflect the new card link
            setSelectedEntity({
                ...selectedEntity,
                card_pk: newCardId,
                card: {
                    id: newCardId,
                    card_id: "", // Will be filled by the actual card data
                    title: cardTitle,
                    user_id: 0,
                    parent_id: 0,
                    created_at: new Date(),
                    updated_at: new Date(),
                    tags: [],
                },
            });

            navigate(`/app/card/${newCardId}`);
        } catch (err) {
            console.error(err);
        }
    };

    return (
        <>
            <Dialog open={showEntityDialog} onClose={onClose} className="relative z-50">
                <div className="fixed inset-0 bg-black/30" aria-hidden="true" />

                <div className="fixed inset-0 flex items-center justify-center p-4">
                    <Dialog.Panel className="w-full max-w-3xl transform overflow-y-auto max-h-[90vh] rounded-2xl bg-white p-6 shadow-xl transition-all">
                        {selectedEntity && (
                            <>
                                <EntityHeader
                                    entity={selectedEntity}
                                    onClose={onClose}
                                    onEdit={onEdit ? handleEditClick : undefined}
                                    onTurnIntoCard={handleTurnIntoCard}
                                />

                                <EntityCardsSection
                                    cards={entityData.associatedCards}
                                    isLoading={entityData.isLoading}
                                    error={entityData.error}
                                />

                                <EntitySimilarSection
                                    similarEntities={entityData.similarEntities}
                                    isLoading={entityData.loadingSimilar}
                                    error={entityData.similarError}
                                    currentEntityName={selectedEntity.name}
                                    onEntityClick={handleEntityClick}
                                    onInitiateMerge={handleInitiateMerge}
                                />
                            </>
                        )}
                    </Dialog.Panel>
                </div>
            </Dialog>

            <EntityMergeDialog
                isOpen={showConfirmDialog}
                onClose={() => setShowConfirmDialog(false)}
                onConfirm={handleConfirmMerge}
                currentEntity={selectedEntity}
                entityToMerge={entityToMerge}
                mergeDirection={mergeDirection}
                isMerging={isMerging}
                mergeError={mergeError}
            />

            <CreateCardDialog
                isOpen={showConvertDialog}
                onClose={() => setShowConvertDialog(false)}
                onCardCreated={handleCardCreated}
                title="Convert Entity to Card"
                initialTitle={cardTitle}
                initialBody={cardBody}
                processEntitiesAndFacts={true}
            />
        </>
    );
}
