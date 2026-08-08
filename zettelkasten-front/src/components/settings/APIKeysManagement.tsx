import React, { useState, useEffect } from 'react';
import { APIKeyResponse, CreateAPIKeyRequest } from '../../models/APIKey';
import { createAPIKey, listAPIKeys, revokeAPIKey } from '../../api/apiKeys';

interface APIKeysManagementProps {}

const APIKeysManagement: React.FC<APIKeysManagementProps> = () => {
  const [apiKeys, setApiKeys] = useState<APIKeyResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [createLoading, setCreateLoading] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyDescription, setNewKeyDescription] = useState('');
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [justCreatedKey, setJustCreatedKey] = useState<{
    id: number;
    key: string;
    name: string;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadApiKeys();
  }, []);

  const loadApiKeys = async () => {
    try {
      setLoading(true);
      const keys = await listAPIKeys();
      setApiKeys(keys);
    } catch (err) {
      setError('Failed to load API keys');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newKeyName.trim()) return;

    try {
      setCreateLoading(true);
      setError(null);

      const request: CreateAPIKeyRequest = {
        name: newKeyName.trim(),
        description: newKeyDescription.trim() || undefined,
      };

      const response = await createAPIKey(request);

      // Show the key temporarily
      setJustCreatedKey({
        id: response.id,
        key: response.key,
        name: response.name,
      });

      // Reset form and reload list
      setNewKeyName('');
      setNewKeyDescription('');
      setShowCreateForm(false);
      await loadApiKeys();
    } catch (err) {
      setError('Failed to create API key');
    } finally {
      setCreateLoading(false);
    }
  };

  const handleRevokeKey = async (keyId: number, keyName: string) => {
    if (
      !confirm(
        `Are you sure you want to revoke the API key "${keyName}"? This action cannot be undone.`,
      )
    ) {
      return;
    }

    try {
      await revokeAPIKey(keyId);
      await loadApiKeys();
    } catch (err) {
      setError('Failed to revoke API key');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatDate = (dateString: string | null) => {
    if (!dateString) return 'Never';
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <div className="text-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto"></div>
        <p className="mt-2 text-gray-600">Loading API keys...</p>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-900 mb-2">API Keys</h2>
        <p className="text-gray-600">
          Create API keys for programmatic access to Zettelgarden. These keys
          provide long-lived access beyond JWT token expiration limits.
        </p>
      </div>

      {error && (
        <div className="mb-6 bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-red-700">{error}</p>
          <button
            onClick={() => setError(null)}
            className="mt-2 text-red-600 hover:text-red-800 text-sm underline"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Just created key notification */}
      {justCreatedKey && (
        <div className="mb-6 bg-green-50 border border-green-200 rounded-md p-4">
          <h3 className="font-medium text-green-800 mb-2">
            API Key Created Successfully!
          </h3>
          <p className="text-green-700 text-sm mb-3">
            <strong>{justCreatedKey.name}</strong> - Copy this key now. It will
            only be shown once and cannot be retrieved later.
          </p>
          <div className="bg-gray-100 border rounded p-3 font-mono text-sm break-all mb-3">
            {justCreatedKey.key}
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => copyToClipboard(justCreatedKey.key)}
              className="bg-green-600 text-white px-3 py-1 rounded text-sm hover:bg-green-700"
            >
              Copy Key
            </button>
            <button
              onClick={() => setJustCreatedKey(null)}
              className="bg-gray-600 text-white px-3 py-1 rounded text-sm hover:bg-gray-700"
            >
              I've Copied It
            </button>
          </div>
        </div>
      )}

      {/* Create new key button */}
      <div className="mb-6">
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 flex items-center gap-2"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 4v16m8-8H4"
            />
          </svg>
          Create New API Key
        </button>
      </div>

      {/* Create form */}
      {showCreateForm && (
        <div className="mb-6 bg-gray-50 border border-gray-200 rounded-md p-4">
          <h3 className="font-medium text-gray-900 mb-4">Create New API Key</h3>
          <form onSubmit={handleCreateKey}>
            <div className="mb-4">
              <label
                htmlFor="keyName"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Key Name *
              </label>
              <input
                id="keyName"
                type="text"
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                placeholder="e.g., My Script, CI Pipeline"
                className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
              <p className="text-xs text-gray-500 mt-1">
                A descriptive name to identify this key
              </p>
            </div>

            <div className="mb-4">
              <label
                htmlFor="keyDescription"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Description (Optional)
              </label>
              <textarea
                id="keyDescription"
                value={newKeyDescription}
                onChange={(e) => setNewKeyDescription(e.target.value)}
                placeholder="Brief description of what this key is used for"
                rows={2}
                className="w-full min-h-[100px] max-h-[30vh] sm:max-h-none border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-y"
              />
            </div>

            <div className="flex gap-2">
              <button
                type="submit"
                disabled={createLoading || !newKeyName.trim()}
                className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {createLoading ? 'Creating...' : 'Create Key'}
              </button>
              <button
                type="button"
                onClick={() => setShowCreateForm(false)}
                className="bg-gray-600 text-white px-4 py-2 rounded-md hover:bg-gray-700"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* API Keys list */}
      <div className="space-y-4">
        {apiKeys && apiKeys.length === 0 ? (
          <div className="text-center py-12">
            <div className="text-gray-400 mb-4">
              <svg
                className="w-16 h-16 mx-auto"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1}
                  d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
                />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No API Keys Yet
            </h3>
            <p className="text-gray-600 mb-4">
              Create your first API key to get started with programmatic access.
            </p>
          </div>
        ) : (
          apiKeys &&
          apiKeys.map((key) => (
            <div
              key={key.id}
              className="border border-gray-200 rounded-md p-4 bg-white"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <h4 className="font-medium text-gray-900">{key.name}</h4>
                    <span
                      className={`px-2 py-1 rounded-full text-xs font-medium ${
                        key.is_active
                          ? 'bg-green-100 text-green-800'
                          : 'bg-red-100 text-red-800'
                      }`}
                    >
                      {key.is_active ? 'Active' : 'Revoked'}
                    </span>
                  </div>

                  {key.description && (
                    <p className="text-gray-600 text-sm mb-2">
                      {key.description}
                    </p>
                  )}

                  <div className="text-sm text-gray-500 space-y-1">
                    <p>Created: {formatDate(key.created_at)}</p>
                    <p>Last used: {formatDate(key.last_used_at)}</p>
                    {!key.is_active && key.revoked_at && (
                      <p>Revoked: {formatDate(key.revoked_at)}</p>
                    )}
                  </div>
                </div>

                {key.is_active && (
                  <div className="ml-4">
                    <button
                      onClick={() => handleRevokeKey(key.id, key.name)}
                      className="bg-red-600 text-white px-3 py-1 rounded-md hover:bg-red-700 text-sm"
                    >
                      Revoke
                    </button>
                  </div>
                )}
              </div>
            </div>
          ))
        )}
      </div>

      {/* Security notice */}
      <div className="mt-8 bg-yellow-50 border border-yellow-200 rounded-md p-4">
        <div className="flex">
          <svg
            className="w-5 h-5 text-yellow-400 mt-0.5 mr-3 flex-shrink-0"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path
              fillRule="evenodd"
              d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <div>
            <h4 className="font-medium text-yellow-800">Security Notice</h4>
            <p className="mt-1 text-yellow-700 text-sm">
              API keys provide persistent access to your account. Store them
              securely and treat them like passwords. Revoked keys cannot be
              recovered. If you suspect a key has been compromised, revoke it
              immediately.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default APIKeysManagement;
