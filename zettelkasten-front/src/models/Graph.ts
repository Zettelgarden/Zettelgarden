// Knowledge graph types shared with the backend /api/graph endpoint.
import { PartialCard } from './Card';

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

export interface Connector {
  card: PartialCard;
  count: number;
}

export interface MonthCount {
  month: string;
  count: number;
}

export interface NetworkStats {
  total_cards: number;
  total_links: number;
  avg_links_per_card: number;
  orphan_count: number;
  top_connectors: Connector[];
  links_by_month: MonthCount[];
}
