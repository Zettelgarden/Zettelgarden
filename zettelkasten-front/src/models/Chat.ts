export interface CardReference {
  cardId: string;
  title: string;
  fullMatch: string;
  startIndex: number;
  endIndex: number;
}

export interface ChatCardData {
  id: number;
  card_id: string;
  title: string;
  body_preview?: string;
  created_at: string;
  updated_at: string;
  tags?: string[];
}

export interface ParsedMessageContent {
  text: string;
  cards: ChatCardData[];
}