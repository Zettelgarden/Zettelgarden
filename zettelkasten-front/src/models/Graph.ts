// Knowledge graph types shared with the backend /api/graph endpoint.

export interface GraphNode {
  id: string; // "card:123", "entity:456", "tag:789"
  type: 'card' | 'entity' | 'tag';
  label: string;
  card_id?: string; // cards only
  sub?: string; // entity type or tag color
}

export interface GraphEdge {
  source: string;
  target: string;
  type: 'reference' | 'parent' | 'entity' | 'tag';
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}
