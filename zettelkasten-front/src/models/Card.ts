import { File } from "./File";
import { Tag } from "./Tags";
import { Task } from "./Task";
import { ExternalEvent } from "./ExternalEvent";
import { RSSArticle } from "../api/rss";

export interface PartialCard {
  id: number;
  card_id: string;
  user_id: number;
  title: string;
  parent_id: number;
  created_at: Date;
  updated_at: Date;
  tags: Tag[];
  is_starred?: boolean;
}

export interface Entity {
  id: number;
  user_id: number;
  name: string;
  description: string;
  type: string;
  created_at: Date;
  updated_at: Date;
  card_count: number;
  card_pk: number | null;
  card?: PartialCard;
}

export interface EntityWithScore extends Entity {
  score: number;
}

export interface RelatedCard {
  card: PartialCard;
  score: number;
  reasons: string[];
}

export interface Card {
  id: number;
  card_id: string;
  user_id: number;
  title: string;
  body: string;
  link: string;
  is_deleted: boolean;
  created_at: Date;
  updated_at: Date;
  parent_id: number;
  parent: PartialCard;
  files: File[];
  children: PartialCard[];
  references: PartialCard[];
  tags: Tag[];
  tasks: Task[];
  external_events: ExternalEvent[];
  entities: Entity[];
  is_starred?: boolean; // Whether the current user has starred this card
  process_entities_and_facts?: boolean; // Whether to process entities and facts on save
  schema_id?: number | null; // ID of the schema this card uses
  structured_data?: Record<string, any> | null; // The structured data for this card's schema
  source_article?: RSSArticle; // The RSS article this card was created from, if any
}

export interface CardChunk {
  id: number;
  card_id: string;
  user_id: number;
  title: string;
  body: string;
  parent_id: number;
  created_at: Date;
  updated_at: Date;
  tags: Tag[];
  ranking: number;
  combined_score: number;
  shared_entities: number;
  entity_similarity: number;
}

export interface SearchResult {
  id: string;
  pk?: number;  // Internal database ID for linking
  type: string;
  title: string;
  preview: string;
  score: number;
  created_at: Date;
  updated_at: Date;
  tags: Tag[] | undefined;
  metadata: {
    parent_id?: number;
    shared_entities?: number;
    entity_similarity?: number;
    semantic_ranking?: number;
    [key: string]: any;
  };
}

export const defaultPartialCard: PartialCard = {
  id: -1,
  card_id: "",
  user_id: -1,
  title: "",
  parent_id: -1,
  created_at: new Date(0),
  updated_at: new Date(0),
  tags: [],
  is_starred: false,
};

export const defaultCard: Card = {
  id: -1,
  card_id: "",
  user_id: -1,
  title: "",
  body: "",
  link: "",
  is_deleted: false,
  created_at: new Date(0),
  updated_at: new Date(0),
  parent_id: -1,
  parent: defaultPartialCard,
  files: [],
  children: [],
  references: [],
  tags: [],
  tasks: [],
  external_events: [],
  entities: [],
  is_starred: false,
  process_entities_and_facts: false,
};

export interface NextIdResponse {
  error: boolean;
  message: string;
  new_id: string;  // Matches the actual backend response
}

enum Rating {
  Again = 0,
  Hard = 1,
  Good = 2,
  Easy = 3
}

export function getRatingValue(rating: string): number {
  switch (rating.toLowerCase()) {
    case 'again':
      return Rating.Again;
    case 'hard':
      return Rating.Hard;
    case 'good':
      return Rating.Good;
    case 'easy':
      return Rating.Easy;
    default:
      throw new Error(`Invalid rating value: ${rating}`);
  }
}

export interface CardTemplate {
  id: number;
  user_id: number;
  name: string;
  title: string;
  body: string;
  created_at: Date;
  updated_at: Date;
}

export const defaultCardTemplate: CardTemplate = {
  id: -1,
  user_id: -1,
  name: "",
  title: "",
  body: "",
  created_at: new Date(0),
  updated_at: new Date(0),
};

export interface ProcessedCardWithDescendants extends CardWithDescendants {
  // Additional frontend-only fields
  isExpanded?: boolean;
  isLoading?: boolean;
}

export interface CardWithDescendants {
  id: number;
  card_id: string;
  user_id: number;
  title: string;
  body: string;
  link: string;
  parent_id: number;
  created_at: Date;
  updated_at: Date;
  depth: number;
  descendants: ProcessedCardWithDescendants[];
}

const defaultCardWithDescendants: CardWithDescendants = {
  id: -1,
  card_id: "",
  user_id: -1,
  title: "",
  body: "",
  link: "",
  parent_id: -1,
  created_at: new Date(0),
  updated_at: new Date(0),
  depth: 0,
  descendants: [],
};

export const processCardWithDescendants = (card: CardWithDescendants): ProcessedCardWithDescendants => {
  return {
    ...card,
    created_at: new Date(card.created_at),
    updated_at: new Date(card.updated_at),
    descendants: card.descendants && card.descendants.map(child => processCardWithDescendants(child)),
  };
};
