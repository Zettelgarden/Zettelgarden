import React, { useState, useEffect } from "react";
import { getUserMemory, updateUserMemory } from "../api/users";
import { setDocumentTitle } from "../utils/title";
import { H6 } from "../components/Header";
import ReactMarkdown from 'react-markdown';

export function MemoryPage() {
    const [userMemory, setUserMemory] = useState<string | null>(null);
    const [isEditing, setIsEditing] = useState(false);
    const [editedMemory, setEditedMemory] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        async function fetchMemory() {
            try {
                const memory = await getUserMemory();
                setUserMemory(memory.memory);
                setEditedMemory(memory.memory);
            } catch (error) {
                console.error("Failed to fetch user memory:", error);
            }
        }

        setDocumentTitle("Memory");
        fetchMemory();
    }, []);

    const handleEdit = () => {
        setIsEditing(true);
        setError(null);
    };

    const handleCancel = () => {
        if (userMemory) {
            setEditedMemory(userMemory);
        }
        setIsEditing(false);
        setError(null);
    };

    const handleSave = async () => {
        setIsLoading(true);
        setError(null);

        try {
            await updateUserMemory(editedMemory);
            setUserMemory(editedMemory);
            setIsEditing(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update memory');
        } finally {
            setIsLoading(false);
        }
    };


    if (isLoading && userMemory === null) {
        return <p>Loading memory...</p>
    }

    if (isEditing) {
        return (
            <div className="p-6">
                <H6>Memory</H6>
                <div className="space-y-4">
                    <div className="flex justify-between items-center">
                        <h2 className="text-xl font-semibold">Your AI Memory</h2>
                        <div className="flex space-x-2">
                            <button
                                onClick={handleSave}
                                disabled={isLoading}
                                className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 disabled:opacity-50"
                            >
                                {isLoading ? "Saving..." : "Save"}
                            </button>
                            <button
                                onClick={handleCancel}
                                disabled={isLoading}
                                className="bg-gray-300 text-gray-700 px-4 py-2 rounded hover:bg-gray-400"
                            >
                                Cancel
                            </button>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                        <div>
                            <h3 className="text-lg font-medium mb-2">Edit Memory</h3>
                            <textarea
                                value={editedMemory}
                                onChange={(e) => setEditedMemory(e.target.value)}
                                className="w-full min-h-[200px] max-h-[40vh] lg:h-96 lg:max-h-none p-3 border border-gray-300 rounded-md font-mono text-sm resize-y"
                                placeholder="Enter your memory content in markdown format..."
                            />
                        </div>

                        <div>
                            <h3 className="text-lg font-medium mb-2">Preview</h3>
                            <div className="min-h-[200px] max-h-[40vh] lg:h-96 lg:max-h-none p-3 border border-gray-200 rounded-md overflow-y-auto bg-gray-50">
                                {editedMemory ? (
                                    <ReactMarkdown>{editedMemory}</ReactMarkdown>
                                ) : (
                                    <p className="text-gray-500 italic">No content to preview</p>
                                )}
                            </div>
                        </div>
                    </div>

                    {error && (
                        <div className="text-red-600 text-sm bg-red-50 p-3 rounded">
                            {error}
                        </div>
                    )}
                </div>
            </div>
        );
    }

    return (
        <div className="p-6">
            <H6>Memory</H6>
            <p className="text-gray-600 mb-4">This memory is used to give the LLM context about you and your preferences when interacting with it.</p>
            <div className="space-y-4">
                <div className="flex justify-between items-center">
                    <button
                        onClick={handleEdit}
                        className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
                    >
                        Edit Memory
                    </button>
                </div>

                {userMemory ? (
                    <div className="prose max-w-none">
                        <ReactMarkdown>{userMemory}</ReactMarkdown>
                    </div>
                ) : (
                    <p className="text-gray-600 italic">
                        No memory content yet. Click "Edit Memory" to add some content.
                    </p>
                )}
            </div>
        </div>
    );
}
