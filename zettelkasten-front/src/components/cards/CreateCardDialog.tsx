import React, { useState } from "react";
import { Dialog } from "@headlessui/react";
import { saveNewCard, suggestCardTitle, getNextRootId } from "../../api/cards";
import { defaultCard } from "../../models/Card";
import { CardIdDiscoveryDialog } from "./CardIdDiscoveryDialog";
import { CardSchemaSection } from "../schemas/CardSchemaSection";

interface CreateCardDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onCardCreated: (cardId: number) => void;
    title: string;
    initialCardId?: string;
    initialTitle?: string;
    initialBody?: string;
    processEntitiesAndFacts?: boolean;
}

export function CreateCardDialog({
    isOpen,
    onClose,
    onCardCreated,
    title,
    initialCardId = "",
    initialTitle = "",
    initialBody = "",
    processEntitiesAndFacts = false,
}: CreateCardDialogProps) {
    const [isConverting, setIsConverting] = useState(false);
    const [convertError, setConvertError] = useState<string | null>(null);
    const [cardTitle, setCardTitle] = useState(initialTitle);
    const [cardBody, setCardBody] = useState(initialBody);
    const [cardId, setCardId] = useState(initialCardId);
    const [showCardIdDiscovery, setShowCardIdDiscovery] = useState(false);
    const [suggestingTitle, setSuggestingTitle] = useState(false);
    const [schemaId, setSchemaId] = useState<number | null>(null);
    const [structuredData, setStructuredData] = useState<Record<string, any>>({});

    const handleSuggestTitle = async () => {
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
    };

    const handleCreateCard = async () => {
        setIsConverting(true);
        setConvertError(null);

        try {
            const newCard = await saveNewCard({
                ...defaultCard,
                card_id: cardId,
                title: cardTitle,
                body: cardBody,
                schema_id: schemaId,
                structured_data: Object.keys(structuredData).length > 0 ? structuredData : undefined,
                process_entities_and_facts: processEntitiesAndFacts,
            });

            if ("error" in newCard) {
                throw new Error("Failed to create card");
            }

            onCardCreated(newCard.id);
            onClose();
        } catch (err) {
            console.error(err);
            setConvertError("Failed to create card");
        } finally {
            setIsConverting(false);
        }
    };

    const handleClose = () => {
        setCardTitle(initialTitle);
        setCardBody(initialBody);
        setCardId(initialCardId);
        setSchemaId(null);
        setStructuredData({});
        setConvertError(null);
        onClose();
    };

    return (
        <>
            <Dialog
                open={isOpen}
                onClose={handleClose}
                className="fixed inset-0 z-50 flex items-center justify-center"
            >
                <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />
                <Dialog.Panel className="bg-white p-4 rounded-lg max-w-2xl mx-auto relative w-full">
                    <Dialog.Title className="text-base font-semibold mb-3">
                        {title}
                    </Dialog.Title>
                    <div className="space-y-3 mb-4">
                        <div>
                            <label className="block text-xs font-medium text-gray-700 mb-1">
                                Card ID
                            </label>
                            <div className="relative">
                                <input
                                    type="text"
                                    value={cardId}
                                    onChange={(e) => setCardId(e.target.value)}
                                    className="block w-full rounded-md border border-gray-300 px-2.5 py-1.5 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm pr-16 text-sm"
                                    placeholder="Enter card ID..."
                                />
                                <div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
                                    <button
                                        onClick={async () => {
                                            const response = await getNextRootId();
                                            setCardId(response.new_id);
                                        }}
                                        className="p-1 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded"
                                        type="button"
                                        title="Use next available root ID"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                            <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                                        </svg>
                                    </button>
                                    <button
                                        onClick={() => setShowCardIdDiscovery(true)}
                                        className="p-1 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded"
                                        type="button"
                                        title="Discover card ID from hierarchy"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                            <path fillRule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clipRule="evenodd" />
                                        </svg>
                                    </button>
                                </div>
                            </div>
                        </div>
                        <div>
                            <label className="block text-xs font-medium text-gray-700 mb-1">
                                Card Title
                            </label>
                            <div className="relative">
                                <input
                                    type="text"
                                    value={cardTitle}
                                    onChange={(e) => setCardTitle(e.target.value)}
                                    className="w-full p-2 border rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500 pr-20 text-sm"
                                    placeholder="Enter card title..."
                                />
                                <button
                                    onClick={handleSuggestTitle}
                                    disabled={suggestingTitle || !cardBody.trim()}
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
                        <div>
                            <label className="block text-xs font-medium text-gray-700 mb-1">
                                Card Content
                            </label>
                            <textarea
                                value={cardBody}
                                onChange={(e) => setCardBody(e.target.value)}
                                className="w-full min-h-[120px] max-h-[30vh] sm:h-32 sm:max-h-none p-2 border rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm resize-y"
                                placeholder="Enter card content..."
                            />
                        </div>
                        <div>
                            <CardSchemaSection
                                schemaId={schemaId}
                                structuredData={structuredData}
                                onSchemaChange={setSchemaId}
                                onDataChange={setStructuredData}
                                disabled={isConverting}
                            />
                        </div>
                    </div>
                    {convertError && <p className="text-red-600 mb-3 text-sm">{convertError}</p>}
                    <div className="flex justify-end gap-3">
                        <button
                            onClick={handleClose}
                            className="px-3 py-1.5 text-gray-600 hover:text-gray-800 text-sm"
                            disabled={isConverting}
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleCreateCard}
                            disabled={isConverting || !cardTitle.trim() || !cardBody.trim()}
                            className="px-3 py-1.5 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50 text-sm"
                        >
                            {isConverting ? "Creating..." : "Create Card"}
                        </button>
                    </div>
                </Dialog.Panel>
            </Dialog>
            {showCardIdDiscovery && (
                <CardIdDiscoveryDialog
                    onClose={() => setShowCardIdDiscovery(false)}
                    onSelectId={(selectedCardId) => {
                        setCardId(selectedCardId);
                        setShowCardIdDiscovery(false);
                    }}
                />
            )}
        </>
    );
}