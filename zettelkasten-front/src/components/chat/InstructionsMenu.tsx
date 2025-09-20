import React, { useState, useEffect } from "react";
import { getChatInstructions, updateChatInstructions, ChatInstructions } from "../../api/chat";

interface InstructionsMenuProps {
  isOpen: boolean;
  onClose: () => void;
}

export function InstructionsMenu({ isOpen, onClose }: InstructionsMenuProps) {
  const [instructions, setInstructions] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (isOpen) {
      loadInstructions();
    }
  }, [isOpen]);

  const loadInstructions = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await getChatInstructions();
      setInstructions(response.instructions || "");
      setHasChanges(false);
    } catch (err) {
      console.error("Failed to load instructions:", err);
      setError("Failed to load instructions");
      setInstructions("");
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setIsSaving(true);
      setError(null);
      await updateChatInstructions(instructions);
      setHasChanges(false);
    } catch (err) {
      console.error("Failed to save instructions:", err);
      setError("Failed to save instructions");
    } finally {
      setIsSaving(false);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInstructions(e.target.value);
    setHasChanges(true);
  };

  const handleClose = () => {
    if (hasChanges) {
      const confirmClose = window.confirm("You have unsaved changes. Are you sure you want to close?");
      if (!confirmClose) return;
    }
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div
        className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold text-gray-900">Chat Instructions</h2>
              <p className="text-sm text-gray-600 mt-1">
                Set special instructions that will be included with every chat message to customize AI behavior.
              </p>
            </div>
            <button
              onClick={handleClose}
              className="text-gray-400 hover:text-gray-600 p-2 rounded-lg hover:bg-gray-100 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 p-6 overflow-y-auto">
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded-lg text-sm">
              {error}
            </div>
          )}

          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="flex items-center gap-3 text-gray-600">
                <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
                <span>Loading instructions...</span>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div>
                <label htmlFor="instructions" className="block text-sm font-medium text-gray-700 mb-2">
                  Instructions (Markdown supported)
                </label>
                <textarea
                  id="instructions"
                  value={instructions}
                  onChange={handleInputChange}
                  placeholder="Enter your custom instructions here. These will be included with every chat message to guide the AI's behavior.

Example:
- Always respond in a professional tone
- Focus on actionable advice
- Include relevant examples when possible"
                  className="w-full h-64 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none text-sm"
                  style={{ fontFamily: 'monospace' }}
                />
                <div className="mt-2 flex justify-between items-center">
                  <span className="text-xs text-gray-500">
                    {instructions.length}/10000 characters
                  </span>
                  {hasChanges && (
                    <span className="text-xs text-amber-600 flex items-center gap-1">
                      <div className="w-2 h-2 bg-amber-500 rounded-full"></div>
                      Unsaved changes
                    </span>
                  )}
                </div>
              </div>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <div className="flex items-start gap-3">
                  <svg className="w-5 h-5 text-blue-500 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <div className="text-sm text-blue-800">
                    <p className="font-medium mb-1">Tips for effective instructions:</p>
                    <ul className="space-y-1 text-blue-700">
                      <li>• Be specific about the tone and style you want</li>
                      <li>• Include examples of desired behavior</li>
                      <li>• Mention any constraints or preferences</li>
                      <li>• Keep instructions concise but clear</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-gray-200">
          <div className="flex justify-between items-center">
            <button
              onClick={handleClose}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 transition-colors"
            >
              Cancel
            </button>
            <div className="flex gap-3">
              {hasChanges && (
                <button
                  onClick={loadInstructions}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 border border-gray-300 rounded-lg transition-colors"
                >
                  Reset
                </button>
              )}
              <button
                onClick={handleSave}
                disabled={isSaving || !hasChanges}
                className="px-6 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isSaving ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Save Instructions</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}