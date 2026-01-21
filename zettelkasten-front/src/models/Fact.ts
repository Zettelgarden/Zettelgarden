import { PartialCard } from "./Card";

export interface Fact {
  id: number;
  fact: string;
  created_at: string;
  updated_at: string;
}

export interface FactWithCard {
  id: number;
  fact: string;
  created_at: string;
  updated_at: string;
  card: PartialCard;
}

export interface FactWithCardAndScore extends FactWithCard {
  score: number;
}
