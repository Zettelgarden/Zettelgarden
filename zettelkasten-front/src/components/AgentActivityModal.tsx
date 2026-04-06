import React, { useEffect, useState } from 'react';
import { getAgentActivity } from '../api/agents';
import { AgentActivityLog } from '../models/Agent';

interface AgentActivityModalProps {
  isOpen: boolean;
  onClose: () => void;
  agentId: number;
  agentName: string;
}

export const AgentActivityModal: React.FC<AgentActivityModalProps> = ({
  isOpen,
  onClose,
  agentId,
  agentName,
}) => {
  const [logs, setLogs] = useState<AgentActivityLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    if (isOpen) {
      fetchActivity();
    }
  }, [isOpen, agentId, page]);

  const fetchActivity = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getAgentActivity(agentId, page, 20);
      setLogs(response.logs);
      setTotal(response.pagination.total);
      setTotalPages(response.pagination.total_pages);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch activity');
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-3xl w-full max-h-[80vh] overflow-hidden flex flex-col">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-bold">
            Activity: {agentName}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700"
          >
            ✕
          </button>
        </div>

        <div className="overflow-auto flex-1">
          {loading ? (
            <p className="text-center py-4">Loading...</p>
          ) : logs.length === 0 ? (
            <p className="text-center py-4 text-gray-500">
              No activity yet
            </p>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-3 py-2 text-left">Action</th>
                  <th className="px-3 py-2 text-left">Target</th>
                  <th className="px-3 py-2 text-left">Details</th>
                  <th className="px-3 py-2 text-left">Time</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id} className="border-t">
                    <td className="px-3 py-2">{log.action}</td>
                    <td className="px-3 py-2">
                      {log.target_type} #{log.target_id}
                    </td>
                    <td className="px-3 py-2">
                      {log.details && JSON.stringify(log.details).slice(0, 50)}
                      {log.details && '...'}
                    </td>
                    <td className="px-3 py-2 text-gray-500">
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {totalPages > 1 && (
          <div className="flex justify-between items-center mt-4 pt-4 border-t">
            <button
              onClick={() => setPage(page - 1)}
              disabled={page === 1}
              className="px-3 py-1 border rounded disabled:opacity-50"
            >
              Previous
            </button>
            <span className="text-sm text-gray-600">
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage(page + 1)}
              disabled={page === totalPages}
              className="px-3 py-1 border rounded disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
