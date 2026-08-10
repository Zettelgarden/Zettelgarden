import React from 'react';
import { PartialCard, RelatedCard } from '../../models/Card';

interface EgoNetworkProps {
  center: PartialCard;
  parent: PartialCard | null;
  children: PartialCard[];
  references: PartialCard[];
  relatedCards: RelatedCard[];
  suggestions: RelatedCard[];
  onCardClick: (cardId: number) => void;
}

type EdgeType = 'parent' | 'child' | 'reference' | 'related';

interface EgoNode {
  id: number;
  label: string;
  edgeType: EdgeType;
}

const EDGE_COLORS: Record<EdgeType, string> = {
  parent: '#f59e0b',
  child: '#f59e0b',
  reference: '#94a3b8',
  related: '#3b82f6',
};

const NODE_COLOR = '#3b82f6';
const SIZE = 280;
const CENTER = SIZE / 2;
const RING_RADIUS = 92;
const NODE_RADIUS = 15;
const MAX_RING1 = 9;
const MAX_RING2 = 8;

function truncate(label: string, max: number): string {
  return label.length > max ? label.slice(0, max - 1) + '…' : label;
}

export function EgoNetwork({
  center,
  parent,
  children,
  references,
  relatedCards,
  suggestions,
  onCardClick,
}: EgoNetworkProps) {
  const ring1: EgoNode[] = [];
  const seen = new Set<number>();

  const push = (node: EgoNode) => {
    if (seen.has(node.id) || ring1.length >= MAX_RING1) return;
    seen.add(node.id);
    ring1.push(node);
  };

  if (parent)
    push({
      id: parent.id,
      label: parent.title || parent.card_id,
      edgeType: 'parent',
    });
  children
    .slice(0, 3)
    .forEach((c) =>
      push({ id: c.id, label: c.title || c.card_id, edgeType: 'child' }),
    );
  references.forEach((ref) =>
    push({
      id: ref.id,
      label: ref.title || ref.card_id,
      edgeType: 'reference',
    }),
  );
  relatedCards.forEach((rc) =>
    push({
      id: rc.card.id,
      label: rc.card.title || rc.card.card_id,
      edgeType: 'related',
    }),
  );

  const ring2 = suggestions
    .filter((s) => !seen.has(s.card.id))
    .slice(0, MAX_RING2);

  if (ring1.length === 0 && ring2.length === 0) {
    return (
      <div className="text-gray-400 text-sm py-2">
        No connections yet — add references or link related cards to grow your
        network.
      </div>
    );
  }

  const nodePositions = ring1.map((node, i) => {
    const angle = (2 * Math.PI * i) / ring1.length - Math.PI / 2;
    return {
      node,
      x: CENTER + RING_RADIUS * Math.cos(angle),
      y: CENTER + RING_RADIUS * Math.sin(angle),
    };
  });

  const renderNode = (
    node: EgoNode,
    x: number,
    y: number,
    keyPrefix: string,
  ) => (
    <g
      key={`${keyPrefix}-${node.id}`}
      onClick={() => onCardClick(node.id)}
      className="cursor-pointer"
      role="button"
      aria-label={node.label}
      data-testid={`ego-node-${node.id}`}
    >
      <circle cx={x} cy={y} r={NODE_RADIUS} fill={NODE_COLOR} />
      <text
        x={x}
        y={y + 4}
        textAnchor="middle"
        fill="#fff"
        fontSize="9"
        fontWeight="600"
      >
        {truncate(node.label, 12)}
      </text>
      <text
        x={x}
        y={y + NODE_RADIUS + 11}
        textAnchor="middle"
        fill="#64748b"
        fontSize="8"
      >
        {node.edgeType}
      </text>
    </g>
  );

  return (
    <div>
      <svg
        width={SIZE}
        height={SIZE}
        viewBox={`0 0 ${SIZE} ${SIZE}`}
        className="max-w-full"
        data-testid="ego-network"
      >
        {/* Edges from center to ring 1, colored by link type */}
        {nodePositions.map(({ node, x, y }) => (
          <line
            key={`line-${node.id}`}
            x1={CENTER}
            y1={CENTER}
            x2={x}
            y2={y}
            stroke={EDGE_COLORS[node.edgeType]}
            strokeWidth={1.5}
            opacity={0.7}
          />
        ))}

        {/* Center node */}
        <g
          onClick={() => onCardClick(center.id)}
          className="cursor-pointer"
          role="button"
          aria-label={center.title || center.card_id}
          data-testid={`ego-node-${center.id}`}
        >
          <circle cx={CENTER} cy={CENTER} r={NODE_RADIUS + 4} fill="#1e40af" />
          <text
            x={CENTER}
            y={CENTER + 4}
            textAnchor="middle"
            fill="#fff"
            fontSize="9"
            fontWeight="700"
          >
            {truncate(center.title || center.card_id, 12)}
          </text>
        </g>

        {nodePositions.map(({ node, x, y }) => renderNode(node, x, y, 'r1'))}
      </svg>

      {/* Ring 2: second-degree suggestions */}
      {ring2.length > 0 && (
        <div className="mt-2">
          <div className="text-xs font-medium text-gray-500 mb-1">
            Two hops out
          </div>
          <div className="flex flex-wrap gap-1">
            {ring2.map((s) => (
              <button
                key={s.card.id}
                type="button"
                onClick={() => onCardClick(s.card.id)}
                className="px-2 py-0.5 text-[11px] rounded-full bg-slate-100 text-slate-700 hover:bg-blue-50 hover:text-blue-700 border border-slate-200 transition-colors"
                title={s.card.title || s.card.card_id}
              >
                {truncate(s.card.title || s.card.card_id, 18)}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
