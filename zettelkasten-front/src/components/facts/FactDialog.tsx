import React, { useEffect, useState } from "react";
import { Dialog } from "@headlessui/react";
import { Link } from "react-router-dom";
import { FactWithCard } from "../../models/Fact";
import { CardTag } from "../cards/CardTag";
import { Button } from "../Button";
import { Entity } from "../../models/Card";
import { getFactEntities } from "../../api/entities";
import { getFactCards, getSimilarFacts, linkFactToCard, mergeFacts, deleteFact, updateFact } from "../../api/facts";
import { CardIcon } from "../../assets/icons/CardIcon";
import { BacklinkInputDropdownList } from "../cards/BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";
import { useShortcutContext } from "../../contexts/ShortcutContext";

interface FactDialogProps {
    onClose: () => void;
    onFactDeleted: () => void;
}

export function FactDialog({ onClose, onFactDeleted }: FactDialogProps) {
    const [entities, setEntities] = useState<Entity[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [cards, setCards] = useState<any[]>([]);
    const [loadingCards, setLoadingCards] = useState(false);
    const [cardsError, setCardsError] = useState<string | null>(null);

    const {
        setShowEntityDialog,
        setSelectedEntity,
        showFactDialog,
        setShowFactDialog,
        selectedFact,
        setSelectedFact,
    } = useShortcutContext();

    function handleEntityClick(entity: Entity) {
        setShowFactDialog(false);
        setSelectedEntity(entity)
        setShowEntityDialog(true);
    }

    const [similarFacts, setSimilarFacts] = useState<FactWithCard[]>([]);
    const [loadingSimilar, setLoadingSimilar] = useState(false);
    const [similarError, setSimilarError] = useState<string | null>(null);

    function handleFactClick(fact: FactWithCard) {
        setSelectedFact(fact);
        setShowFactDialog(true);
    }

    useEffect(() => {
        if (selectedFact) {
            setLoading(true);
            setError(null);
            getFactEntities(selectedFact.id)
                .then(setEntities)
                .catch(() => setError("Failed to load entities"))
                .finally(() => setLoading(false));

            setLoadingCards(true);
            setCardsError(null);
            getFactCards(selectedFact.id)
                .then(setCards)
                .catch(() => setCardsError("Failed to load cards"))
                .finally(() => setLoadingCards(false));

            setLoadingSimilar(true);
            setSimilarError(null);
            getSimilarFacts(selectedFact.id)
                .then(setSimilarFacts)
                .catch(() => setSimilarError("Failed to load similar facts"))
                .finally(() => setLoadingSimilar(false));
        } else {
            setEntities([]);
            setCards([]);
            setSimilarFacts([]);
        }
    }, [selectedFact]);

    async function handleCardSelect(card: PartialCard) {
        if (!selectedFact) return;
        try {
            await linkFactToCard(selectedFact.id, card.id);
            const updatedCards = await getFactCards(selectedFact.id);
            setCards(updatedCards);
        } catch (err) {
            setCardsError("Failed to link card");
            console.error(err);
        }
    }

    const [showConfirmDialog, setShowConfirmDialog] = useState(false);
    const [factToMerge, setFactToMerge] = useState<FactWithCard | null>(null);
    const [isMerging, setIsMerging] = useState(false);
    const [mergeError, setMergeError] = useState<string | null>(null);

    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteError, setDeleteError] = useState<string | null>(null);

    const [isEditing, setIsEditing] = useState(false);
    const [editedFact, setEditedFact] = useState("");

    function handleStartEditing() {
        if (!selectedFact) return;
        setIsEditing(true);
        setEditedFact(selectedFact.fact);
    }

    async function handleSave() {
        if (!selectedFact) return;
        try {
            await updateFact(selectedFact.id, editedFact);
            setSelectedFact({ ...selectedFact, fact: editedFact });
            setIsEditing(false);
        } catch (err) {
            console.error(err);
            setError("Failed to update fact");
        }
    }


    function handleInitiateMerge(fact: FactWithCard) {
        setFactToMerge(fact);
        setShowConfirmDialog(true);
    }

    async function handleConfirmMerge() {
        if (!selectedFact || !factToMerge) return;
        setIsMerging(true);
        setMergeError(null);
        try {
            await mergeFacts(selectedFact.id, factToMerge.id);
            const updatedEntities = await getFactEntities(selectedFact.id);
            const updatedCards = await getFactCards(selectedFact.id);
            const updatedSimilar = await getSimilarFacts(selectedFact.id);
            setEntities(updatedEntities);
            setCards(updatedCards);
            setSimilarFacts(updatedSimilar);
            setShowConfirmDialog(false);
            setFactToMerge(null);
        } catch (err) {
            setMergeError("Failed to merge facts");
            console.error(err);
        } finally {
            setIsMerging(false);
        }
    }

    async function handleConfirmDelete() {
        if (!selectedFact) return;
        setIsDeleting(true);
        setDeleteError(null);
        try {
            await deleteFact(selectedFact.id);
            setShowDeleteConfirm(false);
            onClose();
            onFactDeleted();
        } catch (err) {
            setDeleteError("Failed to delete fact");
            console.error(err);
        } finally {
            setIsDeleting(false);
        }
    }

    return (
        <Dialog open={showFactDialog} onClose={onClose} className="relative z-50">
            <div className="fixed inset-0 bg-black/30" aria-hidden="true" />

            <div className="fixed inset-0 flex items-center justify-center p-4">
                <Dialog.Panel className="w-full max-w-3xl transform overflow-y-auto max-h-[90vh] rounded-2xl bg-white p-6 shadow-xl transition-all">
                    <Dialog.Title className="text-lg font-medium leading-6 text-gray-900 mb-2">
                        {selectedFact && !isEditing ? `Fact: ${selectedFact.fact.slice(0, 50)}...` : "Fact Details"}
                    </Dialog.Title>

                    {selectedFact ? (
                        <div className="mb-4 space-y-2 text-sm text-gray-700">
                            {isEditing ? (

                                <textarea
                                    value={editedFact}
                                    onChange={(e) => setEditedFact(e.target.value)}
                                    className="w-full h-40 p-2 border rounded"
                                />
                            ) : (
                                <p onClick={handleStartEditing} className="cursor-pointer hover:bg-gray-100 p-2 rounded">
                                    {selectedFact.fact}
                                </p>
                            )}
                            {selectedFact.card && (
                                <div>
                                    <span className="text-xs text-gray-600">From: </span>
                                    <Link
                                        to={`/app/card/${selectedFact.card.id}`}
                                        className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800 hover:underline"
                                    >
                                        <div className="w-4 h-4 mr-1 text-gray-400">
                                            <CardIcon />
                                        </div>
                                        [{selectedFact.card.card_id}] {selectedFact.card.title}
                                    </Link>
                                </div>
                            )}
                        </div>
                    ) : (
                        <p className="text-sm text-gray-500">No fact selected.</p>
                    )}

                    <h4 className="text-md font-medium text-gray-800 mt-4 border-t pt-3">Linked Entities:</h4>
                    <div className="min-h-[100px] max-h-[30vh] overflow-y-auto pr-2">
                        {loading && <p>Loading entities...</p>}
                        {error && <p className="text-red-600">{error}</p>}
                        {!loading && entities.length === 0 && <p>No entities linked.</p>}
                        {!loading && entities.length > 0 && (
                            <ul className="space-y-1 text-sm">
                                {entities.map((e) => (
                                    <li key={e.id} onClick={() => handleEntityClick(e)}>
                                        <span className="text-xs text-blue-600 cursor-pointer">
                                            {e.name}
                                        </span>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>

                    <h4 className="text-md font-medium text-gray-800 mt-4 border-t pt-3">Linked Cards:</h4>

                    <BacklinkInputDropdownList
                        onSelect={handleCardSelect}
                        onSearch={() => { }}
                        placeholder="Link a card..."
                        className="mb-2"
                    />

                    <div className="min-h-[100px] max-h-[30vh] overflow-y-auto pr-2">
                        {loadingCards && <p>Loading cards...</p>}
                        {cardsError && <p className="text-red-600">{cardsError}</p>}
                        {!loadingCards && cards.length === 0 && <p>No cards linked.</p>}
                        {!loadingCards && cards.length > 0 && (
                            <ul className="space-y-1 text-sm">
                                {cards.map((c) => (
                                    <li key={c.id}>
                                        <Link
                                            to={`/app/card/${c.id}`}
                                            className="text-blue-600 hover:text-blue-800 hover:underline"
                                        >
                                            <CardTag card={c} showTitle={true} />
                                        </Link>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>

                    <h4 className="text-md font-medium text-gray-800 mt-4 border-t pt-3">Similar Facts:</h4>
                    <div className="min-h-[100px] max-h-[30vh] overflow-y-auto pr-2">
                        {loadingSimilar && <p>Loading similar facts...</p>}
                        {similarError && <p className="text-red-600">{similarError}</p>}
                        {!loadingSimilar && similarFacts.length === 0 && <p>No similar facts.</p>}
                        {!loadingSimilar && similarFacts.length > 0 && (
                            <ul className="space-y-1 text-sm">
                                {similarFacts.map((f) => (
                                    <li
                                        key={f.id}
                                        className="flex items-center justify-between hover:bg-gray-100 p-1 rounded"
                                    >
                                        <span
                                            onClick={() => handleFactClick(f)}
                                            className="text-gray-700 cursor-pointer"
                                        >
                                            • {f.fact}
                                        </span>
                                        <Button
                                            className="ml-2 text-xs bg-green-500 text-white px-2 py-1 rounded"
                                            onClick={() => handleInitiateMerge(f)}
                                        >
                                            Merge
                                        </Button>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>

                    <div className="mt-6 flex justify-between items-center">
                        <div>
                            <Button
                                onClick={() => setShowDeleteConfirm(true)}
                                className="bg-red-500 text-white"
                            >
                                Delete Fact
                            </Button>
                        </div>
                        <div className="flex justify-end gap-3">
                            {isEditing && (
                                <>
                                    <Button onClick={handleSave}>Save</Button>
                                    <Button onClick={() => setIsEditing(false)} className="bg-gray-300">Cancel</Button>
                                </>
                            )}
                            <Button onClick={onClose}>Close</Button>
                        </div>
                    </div>
                </Dialog.Panel>
            </div>
            {showDeleteConfirm && (
                <Dialog
                    open={showDeleteConfirm}
                    onClose={() => setShowDeleteConfirm(false)}
                    className="fixed inset-0 z-50 flex items-center justify-center"
                >
                    <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />
                    <Dialog.Panel className="bg-white p-6 rounded-lg max-w-md mx-auto relative">
                        <Dialog.Title className="text-lg font-semibold mb-4">
                            Confirm Delete
                        </Dialog.Title>
                        <p className="mb-4">
                            Are you sure you want to delete this fact? This action cannot be undone.
                        </p>
                        <p className="text-gray-800 font-medium italic mb-4">{selectedFact?.fact}</p>
                        {deleteError && <p className="text-red-600 mb-2">{deleteError}</p>}
                        <div className="flex justify-end gap-4">
                            <button
                                onClick={() => setShowDeleteConfirm(false)}
                                className="px-4 py-2 text-gray-600 hover:text-gray-800"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleConfirmDelete}
                                disabled={isDeleting}
                                className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"
                            >
                                {isDeleting ? "Deleting..." : "Delete"}
                            </button>
                        </div>
                    </Dialog.Panel>
                </Dialog>
            )}
            {showConfirmDialog && factToMerge && (
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
                                Primary Fact (will be kept):<br />
                                {selectedFact?.fact}
                            </p>
                            <p className="text-gray-600 mb-2">This fact will be merged into the primary:</p>
                            <p className="text-gray-800">• {factToMerge.fact}</p>
                        </div>
                        <p className="text-red-600 text-sm mb-4">
                            This action cannot be undone. The merged fact will be deleted.
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
        </Dialog>
    );
}
