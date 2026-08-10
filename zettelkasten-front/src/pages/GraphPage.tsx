import React, { useCallback, useMemo, useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  MarkerType,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { getGraphData } from '../api/graph';
import { GraphNode } from '../models/Graph';
import { useIsMobile } from '../hooks/useIsMobile';
import { MobileTopBar } from '../components/layout/MobileTopBar';
import { useUIState } from '../contexts/UIStateContext';
import { setDocumentTitle } from '../utils/title';

const NODE_COLORS: Record<string, string> = {
  card: '#3b82f6',
  entity: '#a855f7',
  tag: '#22c55e',
};

const EDGE_COLORS: Record<string, string> = {
  reference: '#94a3b8',
  parent: '#f59e0b',
  entity: '#c084fc',
  tag: '#4ade80',
};

function nodeTypeLabel(node: GraphNode): string {
  return node.type === 'card'
    ? node.card_id || node.label
    : `${node.type}: ${node.label}`;
}

// Deterministic circle layout: cards on an outer ring, entities/tags on an
// inner ring. Drag + zoom let users rearrange.
function computePosition(index: number, count: number, radius: number) {
  const angle = (2 * Math.PI * index) / Math.max(count, 1);
  return { x: radius * Math.cos(angle), y: radius * Math.sin(angle) };
}

function buildLayout(nodes: GraphNode[]) {
  const cardNodes = nodes.filter((n) => n.type === 'card');
  const otherNodes = nodes.filter((n) => n.type !== 'card');

  const cardRadius = Math.max(250, Math.sqrt(cardNodes.length) * 160);
  const otherRadius = Math.max(120, Math.sqrt(otherNodes.length) * 100);

  const positions = new Map<string, { x: number; y: number }>();
  cardNodes.forEach((n, i) =>
    positions.set(n.id, computePosition(i, cardNodes.length, cardRadius)),
  );
  otherNodes.forEach((n, i) =>
    positions.set(n.id, computePosition(i, otherNodes.length, otherRadius)),
  );
  return positions;
}

export function GraphPage() {
  const navigate = useNavigate();
  const isMobile = useIsMobile();
  const { toggleMobileSidebar } = useUIState();
  const [graph, setGraph] = useState<GraphNode[]>([]);
  const [edgesData, setEdgesData] = useState<Edge[]>([]);
  const [showEntities, setShowEntities] = useState(false);
  const [showTags, setShowTags] = useState(false);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  useEffect(() => {
    setDocumentTitle('Knowledge Graph');
  }, []);

  const loadGraph = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const types = ['card'];
      if (showEntities) types.push('entity');
      if (showTags) types.push('tag');
      const data = await getGraphData(types.join(','));
      setGraph(data.nodes);
      setEdgesData(
        data.edges.map((e) => ({
          id: `${e.source}-${e.target}-${e.type}`,
          source: e.source,
          target: e.target,
          type: 'smoothstep',
          style: { stroke: EDGE_COLORS[e.type] || '#94a3b8', strokeWidth: 1.2 },
          markerEnd: { type: MarkerType.ArrowClosed },
        })),
      );
    } catch (err) {
      console.error('Failed to load graph:', err);
      setError('Failed to load the knowledge graph');
      setGraph([]);
      setEdgesData([]);
    } finally {
      setLoading(false);
    }
  }, [showEntities, showTags]);

  useEffect(() => {
    loadGraph();
  }, [loadGraph]);

  const positions = useMemo(() => buildLayout(graph), [graph]);

  const visibleNodes = useMemo(() => {
    const q = search.trim().toLowerCase();
    return graph.filter(
      (n) =>
        q === '' ||
        nodeTypeLabel(n).toLowerCase().includes(q) ||
        n.label.toLowerCase().includes(q),
    );
  }, [graph, search]);

  const visibleIds = useMemo(
    () => new Set(visibleNodes.map((n) => n.id)),
    [visibleNodes],
  );

  useEffect(() => {
    const rfNodes = visibleNodes.map((n) => {
      const pos = positions.get(n.id) || { x: 0, y: 0 };
      return {
        id: n.id,
        position: pos,
        data: { label: nodeTypeLabel(n) },
        style: {
          background: NODE_COLORS[n.type] || '#94a3b8',
          color: '#fff',
          border: 'none',
          borderRadius: n.type === 'card' ? 8 : 999,
          fontSize: 12,
        },
      } as Node;
    });
    setNodes(rfNodes);

    const visibleEdgeIds = new Set(visibleIds);
    const rfEdges = edgesData.filter(
      (e) => visibleEdgeIds.has(e.source) && visibleEdgeIds.has(e.target),
    );
    setEdges(rfEdges);
  }, [visibleNodes, positions, edgesData, visibleIds, setNodes, setEdges]);

  const onNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      if (node.id.startsWith('card:')) {
        const cardId = node.id.split(':')[1];
        navigate(`/app/card/${cardId}`);
      }
    },
    [navigate],
  );

  // Mobile fallback: read-only card list instead of the canvas.
  if (isMobile) {
    return (
      <div>
        <MobileTopBar
          title="Knowledge Graph"
          onMenuClick={toggleMobileSidebar}
        />
        <div className="p-4">
          <h1 className="text-2xl font-bold text-gray-900 mb-4">
            Knowledge Graph
          </h1>
          {error && <div className="text-red-600 mb-4">{error}</div>}
          {graph.length === 0 && !loading && !error && (
            <div className="text-gray-500">No cards in your vault yet.</div>
          )}
          <ul className="space-y-1">
            {graph
              .filter((n) => n.type === 'card')
              .map((n) => (
                <li key={n.id}>
                  <button
                    onClick={() => {
                      const id = n.id.split(':')[1];
                      navigate(`/app/card/${id}`);
                    }}
                    className="w-full text-left px-3 py-2 rounded hover:bg-gray-100 text-sm text-blue-600"
                  >
                    [{n.card_id || n.label}] - {n.label}
                  </button>
                </li>
              ))}
          </ul>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      <MobileTopBar title="Knowledge Graph" onMenuClick={toggleMobileSidebar} />
      <div className="flex items-center gap-3 px-4 py-2 border-b bg-white">
        <h1 className="text-lg font-semibold text-gray-900 mr-2 hidden md:block">
          Knowledge Graph
        </h1>
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search cards..."
          className="flex-1 max-w-xs px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <label className="flex items-center gap-1.5 text-sm text-gray-700 cursor-pointer">
          <input
            type="checkbox"
            checked={showEntities}
            onChange={(e) => setShowEntities(e.target.checked)}
            className="accent-purple-500"
          />
          Entities
        </label>
        <label className="flex items-center gap-1.5 text-sm text-gray-700 cursor-pointer">
          <input
            type="checkbox"
            checked={showTags}
            onChange={(e) => setShowTags(e.target.checked)}
            className="accent-green-500"
          />
          Tags
        </label>
        <div className="text-xs text-gray-400 ml-auto hidden md:block">
          {visibleNodes.length} nodes · {edges.length} links
        </div>
      </div>

      {error && (
        <div className="p-4 text-red-600 bg-red-50 border-b">{error}</div>
      )}

      <div className="flex-1">
        {loading && graph.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-500">
            Loading graph...
          </div>
        ) : graph.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-500">
            No cards in your vault yet.
          </div>
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            fitView
            minZoom={0.1}
            maxZoom={2}
            nodesDraggable
            nodesConnectable={false}
          >
            <Background variant={BackgroundVariant.Dots} gap={18} size={1} />
            <Controls />
            <MiniMap
              nodeColor={(n) =>
                (n.style as { background?: string })?.background || '#3b82f6'
              }
            />
          </ReactFlow>
        )}
      </div>
    </div>
  );
}
