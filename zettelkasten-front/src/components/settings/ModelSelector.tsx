import React, { useState, useEffect } from "react";
import { getChatModels, ChatModel } from "../../api/chat";

interface ModelSelectorProps {
  currentModel: string;
  onModelChange: (model: string) => void;
}

export function ModelSelector({ currentModel, onModelChange }: ModelSelectorProps) {
  const [models, setModels] = useState<ChatModel[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadModels = async () => {
      try {
        const fetched = await getChatModels();
        setModels(fetched);
      } catch (error) {
        console.error("Failed to load chat models:", error);
      } finally {
        setLoading(false);
      }
    };
    loadModels();
  }, []);

  if (loading) {
    return <div className="text-sm text-gray-500">Loading models...</div>;
  }

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-gray-700">
        Default Chat Model
      </label>
      <select
        value={currentModel}
        onChange={(e) => onModelChange(e.target.value)}
        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      >
        {models.map((model) => (
          <option key={model.value} value={model.value}>
            {model.label}
          </option>
        ))}
      </select>
      <p className="text-xs text-gray-500">
        This model will be used for all new chat conversations.
      </p>
    </div>
  );
}
