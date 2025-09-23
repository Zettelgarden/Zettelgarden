import React, { useState, useEffect } from "react";
import { Dialog, Menu } from "@headlessui/react";
import { Link } from "react-router-dom"; // Import Link
import { Entity } from "../../models/Card";
import { PartialCard, SearchResult, defaultPartialCard } from "../../models/Card";
import { semanticSearchCards } from "../../api/cards";
import { CardList } from "../cards/CardList";
import { CardTag } from "../cards/CardTag"; // Import CardTag
import { Button } from "../Button";
import { FactWithCard } from "../../models/Fact";
import { getEntityFacts, getSimilarEntities, mergeEntities } from "../../api/entities";
import { useShortcutContext } from "../../contexts/ShortcutContext";

interface EntityDialogProps {
    onClose: () => void;
    onEdit?: (entity: Entity, event: React.MouseEvent) => void;
}

export function EntityDialog({ onClose, onEdit }: EntityDialogProps) {
    const [associatedCards, setAssociatedCards] = useState<PartialCard[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [facts, setFacts] = useState<FactWithCard[]>([]);
    const [factsError, setFactsError] = useState<string | null>(null);
    const [factsLoading, setFactsLoading] = useState(false);
    const [similarEntities, setSimilarEntities] = useState<Entity[]>([]);
    const [loadingSimilar, setLoadingSimilar] = useState(false);
    const [similarError, setSimilarError] = useState<string | null>(null);
    const [showConfirmDialog, setShowConfirmDialog] = useState(false);
    const [entityToMerge, setEntityToMerge] = useState<Entity | null>(null);
    const [isMerging, setIsMerging] = useState(false);
    const [mergeError, setMergeError] = useState<string | null>(null);
    const [mergeDirection, setMergeDirection] = useState<'into-current' | 'from-current'>('into-current');


    const {
        showEntityDialog,
        setShowEntityDialog,
        selectedEntity,
        setSelectedEntity,
        showFactDialog,
        setShowFactDialog,
        selectedFact,
        setSelectedFact,
    } = useShortcutContext();

    function handleFactClick(fact: FactWithCard) {
        setSelectedFact(fact)
        setShowFactDialog(true)
        setShowEntityDialog(false)
    }

    function handleEntityClick(entity: Entity) {
        setSelectedEntity(entity)
        setShowEntityDialog(true)
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
            } else {
                // The other entity was merged into the current one, so we just refresh the similar entities
                const updatedSimilar = await getSimilarEntities(selectedEntity.id);
                setSimilarEntities(updatedSimilar);
            }

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

    useEffect(() => {
        if (showEntityDialog && selectedEntity) {
            setIsLoading(true);
            setError(null);
            setAssociatedCards([]); // Clear previous cards

            semanticSearchCards(`@[${selectedEntity.name}]`, false, false, false)
                .then((results: SearchResult[]) => {
                    if (results === null) {
                        setAssociatedCards([]);
                        return;
                    }
                    const cards: PartialCard[] = results.map(result => ({
                        id: Number(result.metadata?.id) || 0,
                        card_id: result.metadata?.card_id || "", // This is the string ID like "1.1"
                        title: result.title,
                        body: result.preview || "", // Or handle if preview is not what's needed
                        link: "", // Or construct a link if necessary
                        is_deleted: false, // Assuming not deleted if shown
                        created_at: new Date(result.created_at),
                        updated_at: new Date(result.updated_at),
                        parent_id: result.metadata?.parent_id || 0,
                        user_id: 0, // Or get from metadata if available
                        // Ensure all fields of PartialCard are covered
                        // These might need to be fetched or are not relevant for this list
                        parent: defaultPartialCard,
                        files: [],
                        children_count: 0,
                        references_count: 0,
                        tags: result.tags || [],
                        tasks_count: 0,
                        is_public: false,
                        is_template: false,
                        is_pinned: false,
                        rating: 0,
                        card_type: result.metadata?.card_type || "note",
                        // entities: result.entities || [], // Removed as SearchResult doesn't have .entities directly and PartialCard doesn't store them
                    }));
                    setAssociatedCards(cards);
                })
                .catch((err) => {
                    console.error("Error fetching cards for entity:", err);
                    setError("Failed to load associated cards.");
                })
                .finally(() => {
                    setIsLoading(false);
                });
            // fetch facts
            setFacts([]);
            setFactsError(null);
            setFactsLoading(true);

            getEntityFacts(selectedEntity.id)
                .then((res) => setFacts(res ?? []))
                .catch((err) => {
                    console.error("Error fetching facts:", err);
                    setFactsError("Failed to load facts.");
                    setFacts([]); // keep array
                })
                .finally(() => setFactsLoading(false));

            setLoadingSimilar(true);
            setSimilarError(null);
            getSimilarEntities(selectedEntity.id)
                .then(setSimilarEntities)
                .catch(() => setSimilarError("Failed to load similar entities"))
                .finally(() => setLoadingSimilar(false));
        }
    }, [showEntityDialog, selectedEntity]);

    return (
        <>
            <Dialog open={showEntityDialog} onClose={onClose} className="relative z-50">
                <div className="fixed inset-0 bg-black/30" aria-hidden="true" />

                <div className="fixed inset-0 flex items-center justify-center p-4">
                    <Dialog.Panel className="w-full max-w-3xl transform overflow-y-auto max-h-[90vh] rounded-2xl bg-white p-6 shadow-xl transition-all">
                        <Dialog.Title className="text-lg font-medium leading-6 text-gray-900 mb-2">
                            {selectedEntity ? `Entity: ${selectedEntity.name}` : "Entity Details"}
                        </Dialog.Title>

                        {selectedEntity && (
                            <div className="mb-4 space-y-2 text-sm">
                                {selectedEntity.description && (
                                    <p className="text-gray-700">{selectedEntity.description}</p>
                                )}
                                <div className="text-xs text-gray-500">
                                    <p>Created: {new Date(selectedEntity.created_at).toLocaleDateString()}</p>
                                    <p>Updated: {new Date(selectedEntity.updated_at).toLocaleDateString()}</p>
                                </div>

                                <h4 className="text-md font-medium text-gray-800 mt-4 border-t pt-3">Facts:</h4>
                                <div className="min-h-[100px] max-h-[30vh] overflow-y-auto pr-2">
                                    {factsLoading && <p>Loading facts...</p>}
                                    {factsError && <p className="text-red-600">{factsError}</p>}
                                    {!factsLoading && !factsError && facts.length === 0 && (
                                        <p>No facts linked to this entity.</p>
                                    )}
                                    {!factsLoading && !factsError && facts.length > 0 && (
                                        <ul className="space-y-2">
                                            {facts.map((f) => (
                                                <li
                                                    key={f.id}
                                                    onClick={() => handleFactClick(f)}
                                                    className="cursor-pointer hover:bg-gray-100 p-1 rounded"
                                                >
                                                    <p className="text-sm text-gray-700">• {f.fact}</p>
                                                    {f.card && (
                                                        <span className="text-xs text-blue-600">
                                                            <CardTag card={f.card} showTitle={true} />
                                                        </span>
                                                    )}
                                                </li>
                                            ))}
                                        </ul>
                                    )}
                                </div>
                                {selectedEntity.card && selectedEntity.card.id > 0 && (
                                    <div className="mt-1">
                                        <span className="text-xs text-gray-600">Linked Card: </span>
                                        <Link
                                            to={`/app/card/${selectedEntity.card.id}`}
                                            className="text-blue-600 hover:text-blue-800 hover:underline"
                                            onClick={onClose}
                                        >
                                            <CardTag card={selectedEntity.card} showTitle={true} />
                                        </Link>
                                    </div>
                                )}
                            </div>
                        )}

                        <h4 className="text-md font-medium text-gray-800 mb-2 border-t pt-3">Associated Cards:</h4>
                        <div className="min-h-[150px] max-h-[50vh] overflow-y-auto pr-2">
                            {isLoading && <p>Loading cards...</p>}
                            {error && <p className="text-red-600">{error}</p>}
                            {!isLoading && !error && associatedCards.length === 0 && (
                                <p>No cards found for this entity.</p>
                            )}
                            {!isLoading && !error && associatedCards.length > 0 && (
                                <CardList cards={associatedCards} showAddButton={false} />
                            )}
                        </div>

                        <h4 className="text-md font-medium text-gray-800 mt-4 border-t pt-3">Similar Entities:</h4>
                        <div className="min-h-[100px] max-h-[30vh] overflow-y-auto pr-2">
                            {loadingSimilar && <p>Loading similar entities...</p>}
                            {similarError && <p className="text-red-600">{similarError}</p>}
                            {!loadingSimilar && similarEntities.length === 0 && <p>No similar entities.</p>}
                            {!loadingSimilar && similarEntities.length > 0 && (
                                <ul className="space-y-1 text-sm">
                                    {similarEntities.map((e) => (
                                        <li
                                            key={e.id}
                                            className="flex items-center justify-between hover:bg-gray-100 p-1 rounded"
                                        >
                                            <span
                                                onClick={() => handleEntityClick(e)}
                                                className="text-gray-700 cursor-pointer flex-grow"
                                            >
                                                • {e.name}
                                            </span>
                                            <div className="flex items-center ml-2">
                                                <Menu as="div" className="relative inline-block text-left">
                                                    <div>
                                                        <Menu.Button className="inline-flex justify-center w-full rounded-md border border-gray-300 shadow-sm px-3 py-1 bg-white text-xs font-medium text-gray-700 hover:bg-gray-50 focus:outline-none">
                                                            <svg xmlns="http://www.w3.org/2000/svg" className="h-5" viewBox="0 0 20 20" fill="currentColor">
                                                                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                                                            </svg>
                                                        </Menu.Button>
                                                    </div>
                                                    <Menu.Items className="absolute right-0 w-56 mt-2 origin-top-right bg-white divide-y divide-gray-100 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
                                                        <div className="px-1 py-1 ">
                                                            <Menu.Item>
                                                                {({ active }) => (
                                                                    <button
                                                                        onClick={() => handleInitiateMerge(e, 'into-current')}
                                                                        className={`${active ? 'bg-blue-500 text-white' : 'text-gray-900'
                                                                            } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                                                                    >
                                                                        Merge '{e.name}' into '{selectedEntity?.name}'
                                                                    </button>
                                                                )}
                                                            </Menu.Item>
                                                            <Menu.Item>
                                                                {({ active }) => (
                                                                    <button
                                                                        onClick={() => handleInitiateMerge(e, 'from-current')}
                                                                        className={`${active ? 'bg-blue-500 text-white' : 'text-gray-900'
                                                                            } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                                                                    >
                                                                        Merge '{selectedEntity?.name}' into '{e.name}'
                                                                    </button>
                                                                )}
                                                            </Menu.Item>
                                                        </div>
                                                    </Menu.Items>
                                                </Menu>
                                            </div>
                                        </li>
                                    ))}
                                </ul>
                            )}
                        </div>


                        <div className="mt-6 flex justify-end gap-3">
                            {selectedEntity && onEdit && (
                                <Button
                                    onClick={handleEditClick}
                                    className="bg-blue-500 text-white hover:bg-blue-600"
                                >
                                    Edit
                                </Button>
                            )}
                            <Button onClick={onClose}>
                                Close
                            </Button>
                        </div>
                    </Dialog.Panel>
                </div>
            </Dialog>
            {showConfirmDialog && entityToMerge && (
                <Dialog
                    open={showConfirmDialog}
                    onClose={() => setShowConfirmDialog(false)}
                    className="fixed inset-0 z-50 flex items-center justify-center"
                >
                    <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />
                    <Dialog.Panel className="bg-white p-6 rounded-lg max-w-md mx-auto relative">
                        <Dialog.Title className="text-lg font-semibold mb-4">
                            Confirm Merge
                        </Dialog.Title>
                        <div className="mb-4">
                            <p className="font-medium text-green-600 mb-2">
                                Primary Entity (will be kept):<br />
                                {mergeDirection === 'into-current' ? selectedEntity?.name : entityToMerge.name}
                            </p>
                            <p className="text-gray-600 mb-2">This entity will be merged into the primary:</p>
                            <p className="text-gray-800">• {mergeDirection === 'into-current' ? entityToMerge.name : selectedEntity?.name}</p>
                        </div>
                        <p className="text-red-600 text-sm mb-4">
                            This action cannot be undone. The merged entity will be deleted.
                        </p>
                        {mergeError && <p className="text-red-600 mb-2">{mergeError}</p>}
                        <div className="flex justify-end gap-4">
                            <button
                                onClick={() => setShowConfirmDialog(false)}
                                className="px-4 py-2 text-gray-600 hover:text-gray-800"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleConfirmMerge}
                                disabled={isMerging}
                                className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
                            >
                                {isMerging ? "Merging..." : "Merge"}
                            </button>
                        </div>
                    </Dialog.Panel>
                </Dialog>
            )}
        </>
    );
}
