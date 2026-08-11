import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getNetworkStats } from '../../api/graph';
import { NetworkStats } from '../../models/Graph';
import { CardItem } from '../cards/CardItem';

const MAX_MONTH_BARS = 6;

export function NetworkHealthSection() {
  const navigate = useNavigate();
  const [stats, setStats] = useState<NetworkStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = () => {
    setError(null);
    getNetworkStats()
      .then(setStats)
      .catch((err) => {
        console.error('Failed to load network stats:', err);
        setError('Failed to load network stats');
      });
  };

  useEffect(() => {
    load();
  }, []);

  const maxMonthCount = Math.max(
    1,
    ...(stats?.links_by_month.map((m) => m.count) || [1]),
  );

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-xl font-semibold text-gray-800">Network Health</h2>
        <button
          onClick={load}
          className="text-sm text-blue-500 hover:text-blue-700"
        >
          Refresh
        </button>
      </div>

      {error && <div className="text-red-600 text-sm mb-4">{error}</div>}

      {!stats && !error && (
        <div className="text-gray-500 text-sm">Loading network stats...</div>
      )}

      {stats && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <div>
              <div className="text-sm text-gray-600">Cards</div>
              <div className="text-2xl font-bold text-gray-900">
                {stats.total_cards}
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-600">Links</div>
              <div className="text-2xl font-bold text-gray-900">
                {stats.total_links}
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-600">Avg links / card</div>
              <div className="text-2xl font-bold text-gray-900">
                {stats.avg_links_per_card.toFixed(2)}
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-600">Orphan cards</div>
              <div className="text-2xl font-bold text-gray-900">
                {stats.orphan_count}
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Top connectors */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                Top connectors
              </h3>
              {stats.top_connectors.length === 0 ? (
                <div className="text-gray-500 text-sm">No links yet.</div>
              ) : (
                <ul className="space-y-1">
                  {stats.top_connectors.slice(0, 5).map((c) => (
                    <li key={c.card.id} className="flex items-center gap-2">
                      <div
                        className="flex-grow min-w-0 cursor-pointer"
                        onClick={() => navigate(`/app/card/${c.card.id}`)}
                      >
                        <CardItem card={c.card} />
                      </div>
                      <span className="shrink-0 text-xs text-gray-500 bg-gray-100 rounded-full px-2 py-0.5">
                        {c.count} links
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            {/* Link growth by month */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                Links added (last {MAX_MONTH_BARS} months)
              </h3>
              {stats.links_by_month.every((m) => m.count === 0) ? (
                <div className="text-gray-500 text-sm">No links added yet.</div>
              ) : (
                <div className="space-y-1.5">
                  {stats.links_by_month.map((m) => (
                    <div key={m.month} className="flex items-center gap-2">
                      <span className="w-16 shrink-0 text-xs text-gray-500">
                        {m.month}
                      </span>
                      <div className="flex-grow h-4 bg-gray-100 rounded">
                        <div
                          className="h-4 bg-blue-500 rounded"
                          style={{
                            width: `${Math.round(
                              (m.count / maxMonthCount) * 100,
                            )}%`,
                          }}
                        />
                      </div>
                      <span className="w-6 shrink-0 text-xs text-gray-600 text-right">
                        {m.count}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
