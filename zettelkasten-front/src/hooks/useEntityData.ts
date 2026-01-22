import { useState, useEffect } from "react";
import { Entity } from "../models/Card";
import { PartialCard, SearchResult, defaultPartialCard } from "../models/Card";
import { FactWithCard } from "../models/Fact";
import {
  semanticSearchCards,
  escapeEntityNameForSearch,
} from "../api/cards";
import { getEntityFacts, getSimilarEntities } from "../api/entities";

export interface EntityData {
  associatedCards: PartialCard[];
  isLoading: boolean;
  error: string | null;
  facts: FactWithCard[];
  factsError: string | null;
  factsLoading: boolean;
  similarEntities: EntityWithScore[];
  loadingSimilar: boolean;
  similarError: string | null;
}

export interface EntityWithScore extends Entity {
  score: number;
}

export function useEntityData(showDialog: boolean, entity: Entity | null): EntityData {
  const [associatedCards, setAssociatedCards] = useState<PartialCard[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [facts, setFacts] = useState<FactWithCard[]>([]);
  const [factsError, setFactsError] = useState<string | null>(null);
  const [factsLoading, setFactsLoading] = useState(false);
  const [similarEntities, setSimilarEntities] = useState<EntityWithScore[]>([]);
  const [loadingSimilar, setLoadingSimilar] = useState(false);
  const [similarError, setSimilarError] = useState<string | null>(null);

  useEffect(() => {
    if (showDialog && entity) {
      // Fetch associated cards
      setIsLoading(true);
      setError(null);
      setAssociatedCards([]);

      semanticSearchCards(`@[${escapeEntityNameForSearch(entity.name)}]`, false, false, false, true)
        .then((results: SearchResult[]) => {
          if (results === null) {
            setAssociatedCards([]);
            return;
          }
          const cards: PartialCard[] = results.map(result => ({
            id: Number(result.metadata?.id) || 0,
            card_id: result.metadata?.card_id || "",
            title: result.title,
            body: result.preview || "",
            link: "",
            is_deleted: false,
            created_at: new Date(result.created_at),
            updated_at: new Date(result.updated_at),
            parent_id: result.metadata?.parent_id || 0,
            user_id: 0,
            parent: defaultPartialCard,
            files: [],
            children_count: 0,
            references_count: 0,
            tags: result.tags || [],
            tasks_count: 0,
            is_public: false,
            is_template: false,
            is_pinned: false,
            rating: 0,
            card_type: result.metadata?.card_type || "note",
          }));
          setAssociatedCards(cards);
        })
        .catch((err) => {
          console.error("Error fetching cards for entity:", err);
          setError("Failed to load associated cards.");
        })
        .finally(() => {
          setIsLoading(false);
        });

      // Fetch facts
      setFacts([]);
      setFactsError(null);
      setFactsLoading(true);

      getEntityFacts(entity.id)
        .then((res) => setFacts(res ?? []))
        .catch((err) => {
          console.error("Error fetching facts:", err);
          setFactsError("Failed to load facts.");
          setFacts([]);
        })
        .finally(() => setFactsLoading(false));

      // Fetch similar entities
      setLoadingSimilar(true);
      setSimilarError(null);
      getSimilarEntities(entity.id)
        .then(setSimilarEntities)
        .catch(() => setSimilarError("Failed to load similar entities"))
        .finally(() => setLoadingSimilar(false));
    }
  }, [showDialog, entity]);

  return {
    associatedCards,
    isLoading,
    error,
    facts,
    factsError,
    factsLoading,
    similarEntities,
    loadingSimilar,
    similarError,
  };
}
