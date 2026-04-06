import React, { useEffect, useState } from 'react';
import { listAgents, revokeAgent } from '../api/agents';
import { Agent } from '../models/Agent';
import { CreateAgentModal } from './CreateAgentModal';
import { AgentActivityModal } from './AgentActivityModal';

export const AgentManagement: React.FC = () => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [activityAgent, setActivityAgent] = useState<Agent | null>(null);

  const [revoking, setRevoking] = useState<number | null>(null);

  useEffect(() => {
    fetchAgents();
  }, []);

  const fetchAgents = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listAgents();
      setAgents(data);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch agents');
    } finally {
      setLoading(false);
    }
  };

  const handleRevoke = async (agentId: number) => {
    if (!window.confirm('Are you sure? This will revoke the agent\'s access immediately.')) {
      return;
    }

    setRevoking(agentId);
    try {
      await revokeAgent(agentId);
      await fetchAgents();
    } catch (err: any) {
      console.error('Failed to revoke agent:', err);
      alert('Failed to revoke agent');
    } finally {
      setRevoking(null);
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">AI Agents</h1>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
        >
          + Create Agent
        </button>
      </div>

      <p className="text-gray-600 mb-6">
        Manage AI agents that can access your Zettelgarden workspace via API keys.
      </p>

      {loading ? (
        <p className="text-center py-8">Loading...</p>
      ) : error ? (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      ) : agents.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <p className="text-gray-500 mb-4">No agents configured yet</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="text-blue-500 hover:underline"
          >
            Create your first agent
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left">Name</th>
                <th className="px-4 py-3 text-left">Status</th>
                <th className="px-4 py-3 text-left">Created</th>
                <th className="px-4 py-3 text-left">Last Used</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((agent) => (
                <tr key={agent.id} className="border-t">
                  <td className="px-4 py-3 font-medium">
                    🤖 {agent.name}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-2 py-1 rounded text-xs ${
                        agent.is_active
                          ? 'bg-green-100 text-green-700'
                          : 'bg-red-100 text-red-700'
                      }`}
                    >
                      {agent.is_active ? 'Active' : 'Revoked'}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {new Date(agent.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {agent.last_used
                      ? new Date(agent.last_used).toLocaleDateString()
                      : 'Never'}
                  </td>
                  <td className="px-4 py-3 text-right space-x-2">
                    <button
                      onClick={() => setActivityAgent(agent)}
                      className="text-blue-500 hover:underline text-sm"
                    >
                      Activity
                    </button>
                    {agent.is_active && (
                      <button
                        onClick={() => handleRevoke(agent.id)}
                        disabled={revoking === agent.id}
                        className="text-red-500 hover:underline text-sm"
                      >
                        {revoking === agent.id ? 'Revoking...' : 'Revoke'}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <CreateAgentModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={fetchAgents}
      />

      {activityAgent && (
        <AgentActivityModal
          isOpen={true}
          onClose={() => setActivityAgent(null)}
          agentId={activityAgent.id}
          agentName={activityAgent.name}
        />
      )}
    </div>
  );
};
