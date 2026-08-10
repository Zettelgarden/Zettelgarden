// zettelkasten-front/src/pages/GraphPage.test.tsx
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { GraphPage } from './GraphPage';
import { getGraphData } from '../api/graph';
import { getCardPath } from '../api/cards';
import { GraphData } from '../models/Graph';
import { UIStateProvider } from '../contexts/UIStateContext';

vi.mock('../api/graph', () => ({
  getGraphData: vi.fn(),
}));

vi.mock('../api/cards', () => ({
  getCardPath: vi.fn(),
}));

// React Flow is heavy for jsdom; render a probe that captures its props.
vi.mock('@xyflow/react', () => ({
  ReactFlow: ({ nodes, edges, onNodeClick }: any) => (
    <div
      data-testid="react-flow"
      data-nodes={JSON.stringify(nodes.map((n: any) => n.id))}
      data-edges={JSON.stringify(edges.map((e: any) => e.id))}
    >
      {nodes.map((n: any) => (
        <div
          key={n.id}
          data-testid={`rf-node-${n.id}`}
          onClick={(e) => onNodeClick(e, n)}
        />
      ))}
    </div>
  ),
  Background: () => null,
  BackgroundVariant: { Dots: 'dots' },
  Controls: () => null,
  MiniMap: () => null,
  MarkerType: { ArrowClosed: 'arrowclosed' },
  useNodesState: (initial: any) => {
    const [nodes, setNodes] = React.useState(initial);
    return [nodes, setNodes, () => {}];
  },
  useEdgesState: (initial: any) => {
    const [edges, setEdges] = React.useState(initial);
    return [edges, setEdges, () => {}];
  },
}));

function makeGraph(): GraphData {
  return {
    nodes: [
      { id: 'card:1', type: 'card', label: 'Root Card', card_id: '1' },
      { id: 'card:2', type: 'card', label: 'Child Card', card_id: '2' },
      { id: 'entity:10', type: 'entity', label: 'Python' },
      { id: 'tag:20', type: 'tag', label: 'research' },
    ],
    edges: [
      { source: 'card:1', target: 'card:2', type: 'reference' },
      { source: 'card:1', target: 'entity:10', type: 'entity' },
      { source: 'card:2', target: 'tag:20', type: 'tag' },
    ],
  };
}

// Simulate the backend's ?types= filter so the default fetch only returns
// card nodes and card-to-card edges.
function mockBackend() {
  vi.mocked(getGraphData).mockImplementation((types?: string) => {
    const g = makeGraph();
    const wanted = new Set((types || 'card,entity,tag').split(','));
    return Promise.resolve({
      nodes: g.nodes.filter((n) => wanted.has(n.type)),
      edges: g.edges.filter((e) => {
        const src = g.nodes.find((n) => n.id === e.source);
        const tgt = g.nodes.find((n) => n.id === e.target);
        return !!src && !!tgt && wanted.has(src.type) && wanted.has(tgt.type);
      }),
    });
  });
}

function renderPage(innerWidth: number) {
  Object.defineProperty(window, 'innerWidth', {
    writable: true,
    configurable: true,
    value: innerWidth,
  });
  return render(
    <MemoryRouter>
      <UIStateProvider>
        <GraphPage />
      </UIStateProvider>
    </MemoryRouter>,
  );
}

describe('GraphPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockBackend();
  });

  it('renders a read-only card list on mobile', async () => {
    renderPage(500);

    await waitFor(() => {
      expect(screen.getByText('[1] - Root Card')).toBeInTheDocument();
    });
    // Mobile list shows card rows (label includes card_id and title).
    expect(screen.getByText('[2] - Child Card')).toBeInTheDocument();
    expect(screen.queryByTestId('react-flow')).not.toBeInTheDocument();
  });

  it('renders the React Flow canvas with card nodes by default on desktop', async () => {
    renderPage(1280);

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument();
    });

    // Cards only by default (entities/tags off).
    expect(getGraphData).toHaveBeenCalledWith('card');
    const flow = screen.getByTestId('react-flow');
    const nodeIds = JSON.parse(flow.getAttribute('data-nodes')!);
    expect(nodeIds).toEqual(['card:1', 'card:2']);
    const edgeIds = JSON.parse(flow.getAttribute('data-edges')!);
    // Only the reference edge (entity/tag edges dropped with nodes).
    expect(edgeIds).toEqual(['card:1-card:2-reference']);
  });

  it('refetches with entity and tag types when toggles are enabled', async () => {
    renderPage(1280);

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Entities'));
    fireEvent.click(screen.getByLabelText('Tags'));

    await waitFor(() => {
      expect(getGraphData).toHaveBeenLastCalledWith('card,entity,tag');
    });
    await waitFor(() => {
      const flow = screen.getByTestId('react-flow');
      const nodeIds = JSON.parse(flow.getAttribute('data-nodes')!);
      expect(nodeIds).toContain('entity:10');
      expect(nodeIds).toContain('tag:20');
    });
  });

  it('filters nodes by search term', async () => {
    renderPage(1280);

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByPlaceholderText('Search cards...'), {
      target: { value: 'Child' },
    });

    const flow = screen.getByTestId('react-flow');
    const nodeIds = JSON.parse(flow.getAttribute('data-nodes')!);
    expect(nodeIds).toEqual(['card:2']);
  });

  it('traces and highlights a connection path between two clicked cards', async () => {
    vi.mocked(getCardPath).mockResolvedValue([
      { id: 1, card_id: '1', title: 'Root Card', parent_id: 0 } as any,
      { id: 2, card_id: '2', title: 'Child Card', parent_id: 0 } as any,
    ]);
    renderPage(1280);

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument();
    });

    // First click arms the path finder.
    fireEvent.click(screen.getByTestId('rf-node-card:1'));
    expect(
      screen.getByText(/click another card to trace/i),
    ).toBeInTheDocument();

    // Second click computes the path.
    fireEvent.click(screen.getByTestId('rf-node-card:2'));

    await waitFor(() => {
      expect(getCardPath).toHaveBeenCalledWith(1, 2);
    });
    await waitFor(() => {
      expect(screen.getByText('Path:')).toBeInTheDocument();
      expect(screen.getByText('Root Card')).toBeInTheDocument();
      expect(screen.getByText('Child Card')).toBeInTheDocument();
    });

    // Clearing the path removes the banner.
    fireEvent.click(screen.getByText('Clear path'));
    expect(screen.queryByText('Path:')).not.toBeInTheDocument();
  });
});
