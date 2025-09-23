import React, { useEffect, useState } from "react";
import { Dialog, Menu } from "@headlessui/react";
import { Link } from "react-router-dom";
import { FactWithCard } from "../../models/Fact";
import { CardTag } from "../cards/CardTag";
import { Button } from "../Button";
import { Entity } from "../../models/Card";
import { getFactEntities } from "../../api/entities";
import { getFactCards, getSimilarFacts, linkFactToCard, mergeFacts, deleteFact, updateFact } from "../../api/facts";
import { getNextRootId, saveNewCard, suggestCardTitle } from "../../api/cards";
import { defaultCard } from "../../models/Card";
import { CardIcon } from "../../assets/icons/CardIcon";
import { BacklinkInputDropdownList } from "../cards/BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";
import { useShortcutContext } from "../../contexts/ShortcutContext";
import { useNavigate } from "react-router-dom";
import { CardIdDiscoveryDialog } from "../cards/CardIdDiscoveryDialog";

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
    const [mergeDirection, setMergeDirection] = useState<'into-current' | 'from-current'>('into-current');

    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteError, setDeleteError] = useState<string | null>(null);

    const [isEditing, setIsEditing] = useState(false);
    const [editedFact, setEditedFact] = useState("");

    const [showConvertDialog, setShowConvertDialog] = useState(false);
    const [isConverting, setIsConverting] = useState(false);
    const [convertError, setConvertError] = useState<string | null>(null);
    const [cardTitle, setCardTitle] = useState("");
    const [cardBody, setCardBody] = useState("");
    const [cardId, setCardId] = useState("");
    const [showCardIdDiscovery, setShowCardIdDiscovery] = useState(false);
    const [suggestingTitle, setSuggestingTitle] = useState(false);

    const navigate = useNavigate();

    function handleStartEditing() {
        if (!selectedFact) return;
        setIsEditing(true);
        setEditedFact(selectedFact.fact);
    }

    function handleStartConversion() {
        if (!selectedFact) return;
        const truncatedTitle = selectedFact.fact.length > 50
            ? selectedFact.fact.slice(0, 50) + "..."
            : selectedFact.fact;
        setCardTitle(truncatedTitle);
        setCardBody(selectedFact.fact);
        setCardId("");
        setShowConvertDialog(true);
        setConvertError(null);
    }

    async function handleSuggestTitle() {
        if (!cardBody.trim()) {
            setConvertError("Please add some content to the card body before suggesting a title.");
            return;
        }

        setSuggestingTitle(true);
        setConvertError(null);

        try {
            const suggestedTitle = await suggestCardTitle(cardBody);
            setCardTitle(suggestedTitle);
        } catch (error: any) {
            console.error("Error suggesting title:", error);
            setConvertError(error.message || "Failed to suggest title. Please try again.");
        } finally {
            setSuggestingTitle(false);
        }
    }

    async function handleConvertToCard() {
        if (!selectedFact) return;
        setIsConverting(true);
        setConvertError(null);

        try {

            const newCard = await saveNewCard({
                ...defaultCard,
                card_id: cardId,
                title: cardTitle,
                body: cardBody,
                process_entities_and_facts: false,
            });

            if ("error" in newCard) {
                throw new Error("Failed to create card");
            }

            await linkFactToCard(selectedFact.id, newCard.id);

            const updatedCards = await getFactCards(selectedFact.id);
            setCards(updatedCards);

            setShowConvertDialog(false);
            navigate(`/app/card/${newCard.id}`);
        } catch (err) {
            console.error(err);
            setConvertError("Failed to convert fact to card");
        } finally {
            setIsConverting(false);
        }
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


    function handleInitiateMerge(fact: FactWithCard, direction: 'into-current' | 'from-current') {
        setMergeDirection(direction);
        setFactToMerge(fact);
        setShowConfirmDialog(true);
    }

    async function handleConfirmMerge() {
        if (!selectedFact || !factToMerge) return;
        setIsMerging(true);
        setMergeError(null);

        const primaryFactId = mergeDirection === 'into-current' ? selectedFact.id : factToMerge.id;
        const secondaryFactId = mergeDirection === 'into-current' ? factToMerge.id : selectedFact.id;

        try {
            await mergeFacts(primaryFactId, secondaryFactId);

            if (mergeDirection === 'from-current') {
                setSelectedFact(factToMerge);
            } else {
                const updatedEntities = await getFactEntities(selectedFact.id);
                const updatedCards = await getFactCards(selectedFact.id);
                const updatedSimilar = await getSimilarFacts(selectedFact.id);
                setEntities(updatedEntities);
                setCards(updatedCards);
                setSimilarFacts(updatedSimilar);
            }
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
                <Dialog.Panel className="w-full max-w-4xl transform max-h-[90vh] rounded-2xl bg-white p-6 shadow-xl transition-all flex flex-col">
                    <Dialog.Title className="text-lg font-medium leading-6 text-gray-900 mb-2">
                        {selectedFact && !isEditing ? `Fact: ${selectedFact.fact.slice(0, 50)}...` : "Fact Details"}
                    </Dialog.Title>
                    <div className="overflow-y-auto">
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
                                            onClick={onClose}
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
                                                onClick={onClose}
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
                                                className="text-gray-700 cursor-pointer flex-grow"
                                            >
                                                • {f.fact}
                                            </span>
                                            <div className="flex items-center ml-2">
                                                <span className="text-xs text-blue-600 mr-2">
                                                    [{f.card.card_id}]
                                                </span>
                                                <Menu as="div" className="relative inline-block text-left">
                                                    <div>
                                                        <Menu.Button className="inline-flex justify-center w-full rounded-md border border-gray-300 shadow-sm px-3 py-1 bg-white text-xs font-medium text-gray-700 hover:bg-gray-50 focus:outline-none">
                                                            <svg xmlns="http://www.w3.org/2000/svg" className="h-5" viewBox="0 0 20 20" fill="currentColor">
                                                                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                                                            </svg>
                                                        </Menu.Button>
                                                    </div>
                                                    <Menu.Items className="absolute right-0 w-64 mt-2 origin-top-right bg-white divide-y divide-gray-100 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
                                                        <div className="px-1 py-1 ">
                                                            <Menu.Item>
                                                                {({ active }) => (
                                                                    <button
                                                                        onClick={() => handleInitiateMerge(f, 'into-current')}
                                                                        className={`${active ? 'bg-blue-500 text-white' : 'text-gray-900'
                                                                            } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                                                                    >
                                                                        Merge '{f.fact.slice(0, 20)}...' into current
                                                                    </button>
                                                                )}
                                                            </Menu.Item>
                                                            <Menu.Item>
                                                                {({ active }) => (
                                                                    <button
                                                                        onClick={() => handleInitiateMerge(f, 'from-current')}
                                                                        className={`${active ? 'bg-blue-500 text-white' : 'text-gray-900'
                                                                            } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                                                                    >
                                                                        Merge current into '{f.fact.slice(0, 20)}...'
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
                    </div>

                    <div className="mt-6 flex justify-between items-center border-t pt-4">
                        <div className="flex gap-3">
                            <Button
                                onClick={() => setShowDeleteConfirm(true)}
                                className="bg-red-500 text-white"
                            >
                                Delete Fact
                            </Button>
                            <Button
                                onClick={handleStartConversion}
                                disabled={!selectedFact}
                            >
                                Create Card
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
                                {mergeDirection === 'into-current' ? selectedFact?.fact : factToMerge.fact}
                            </p>
                            <p className="text-gray-600 mb-2">This fact will be merged into the primary:</p>
                            <p className="text-gray-800">• {mergeDirection === 'into-current' ? factToMerge.fact : selectedFact?.fact}</p>
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
            {showConvertDialog && (
                <Dialog
                    open={showConvertDialog}
                    onClose={() => setShowConvertDialog(false)}
                    className="fixed inset-0 z-50 flex items-center justify-center"
                >
                    <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />
                    <Dialog.Panel className="bg-white p-6 rounded-lg max-w-2xl mx-auto relative w-full">
                        <Dialog.Title className="text-lg font-semibold mb-4">
                            Convert Fact to Card
                        </Dialog.Title>
                        <div className="space-y-4 mb-6">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    Card ID
                                </label>
                                <div className="relative">
                                    <input
                                        type="text"
                                        value={cardId}
                                        onChange={(e) => setCardId(e.target.value)}
                                        className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm pr-24"
                                        placeholder="Enter card ID..."
                                    />
                                    <button
                                        onClick={() => setShowCardIdDiscovery(true)}
                                        className="absolute right-2 top-1/2 -translate-y-1/2 text-sm text-blue-600 hover:text-blue-800"
                                        type="button"
                                    >
                                        Discover ID
                                    </button>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    Card Title
                                </label>
                                <div className="relative">
                                    <input
                                        type="text"
                                        value={cardTitle}
                                        onChange={(e) => setCardTitle(e.target.value)}
                                        className="w-full p-2 border rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500 pr-24"
                                        placeholder="Enter card title..."
                                    />
                                    <button
                                        onClick={handleSuggestTitle}
                                        disabled={suggestingTitle || !cardBody.trim()}
                                        className="absolute right-2 top-1/2 -translate-y-1/2 text-sm text-blue-600 hover:text-blue-800 disabled:text-gray-400 disabled:cursor-not-allowed"
                                        type="button"
                                    >
                                        {suggestingTitle ? "Suggesting..." : "Suggest"}
                                    </button>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    Card Content
                                </label>
                                <textarea
                                    value={cardBody}
                                    onChange={(e) => setCardBody(e.target.value)}
                                    className="w-full h-32 p-2 border rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="Enter card content..."
                                />
                            </div>
                        </div>
                        {convertError && <p className="text-red-600 mb-4">{convertError}</p>}
                        <div className="flex justify-end gap-4">
                            <button
                                onClick={() => setShowConvertDialog(false)}
                                className="px-4 py-2 text-gray-600 hover:text-gray-800"
                                disabled={isConverting}
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleConvertToCard}
                                disabled={isConverting || !cardTitle.trim() || !cardBody.trim()}
                                className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
                            >
                                {isConverting ? "Converting..." : "Create Card"}
                            </button>
                        </div>
                    </Dialog.Panel>
                </Dialog>
            )}
            {showCardIdDiscovery && (
                <CardIdDiscoveryDialog
                    onClose={() => setShowCardIdDiscovery(false)}
                    onSelectId={(selectedCardId) => {
                        setCardId(selectedCardId);
                        setShowCardIdDiscovery(false);
                    }}
                />
            )}
        </Dialog>
    );
}
