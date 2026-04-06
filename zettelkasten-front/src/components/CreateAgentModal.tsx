import React, { useState } from 'react';
import { createAgent } from '../api/agents';

interface CreateAgentModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const CreateAgentModal: React.FC<CreateAgentModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
}) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await createAgent(name, description || undefined);
      setApiKey(response.api_key);
      setName('');
      setDescription('');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to create agent');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (apiKey) {
      onSuccess();
    }
    setApiKey(null);
    onClose();
  };

  const copyToClipboard = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey);
      // Could show success message
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full">
        {apiKey ? (
          <>
            <h2 className="text-xl font-bold mb-4">Agent Created!</h2>
            <p className="mb-2 text-sm text-gray-600">
              Save this API key now - it won't be shown again:
            </p>
            <code className="block bg-gray-100 p-3 rounded mb-4 text-xs break-all">
              {apiKey}
            </code>
            <button
              onClick={handleClose}
              className="w-full bg-blue-500 text-white py-2 rounded hover:bg-blue-600"
            >
              Close
            </button>
          </>
        ) : (
          <>
            <h2 className="text-xl font-bold mb-4">Create New Agent</h2>
            <form onSubmit={handleSubmit}>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-1">
                  Agent Name *
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full border rounded px-3 py-2"
                  required
                  placeholder="e.g., Claude Research Agent"
                />
              </div>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-1">
                  Description (optional)
                </label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full border rounded px-3 py-2"
                  rows={2}
                  placeholder="What does this agent do?"
                />
              </div>
              {error && (
                <p className="text-red-500 text-sm mb-4">{error}</p>
              )}
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={onClose}
                  className="flex-1 border border-gray-300 py-2 rounded hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={loading || !name}
                  className="flex-1 bg-blue-500 text-white py-2 rounded hover:bg-blue-600 disabled:bg-gray-300"
                >
                  {loading ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          </>
        )}
      </div>
    </div>
  );
};
