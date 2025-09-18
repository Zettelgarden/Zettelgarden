import { Card } from './Card';

export interface CardReference {
  cardId: string;
  title: string;
  fullMatch: string;
  startIndex: number;
  endIndex: number;
}

export interface ParsedMessageContent {
  text: string;
  cards: Card[];
}