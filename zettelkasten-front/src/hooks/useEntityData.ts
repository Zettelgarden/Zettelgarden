import { useState, useEffect } from 'react';
import { Entity, EntityCard } from '../models/Card';
import { FactWithCard } from '../models/Fact';
import {
  getEntityCards,
  getEntityFacts,
  getSimilarEntities,
} from '../api/entities';

export interface EntityData {
  associatedCards: EntityCard[];
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

export function useEntityData(
  showDialog: boolean,
  entity: Entity | null,
): EntityData {
  const [associatedCards, setAssociatedCards] = useState<EntityCard[]>([]);
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
      // Fetch associated cards via the junction (search-independent).
      setIsLoading(true);
      setError(null);
      setAssociatedCards([]);

      getEntityCards(entity.id)
        .then((cards) => {
          setAssociatedCards(cards ?? []);
        })
        .catch((err) => {
          console.error('Error fetching cards for entity:', err);
          setError('Failed to load associated cards.');
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
          console.error('Error fetching facts:', err);
          setFactsError('Failed to load facts.');
          setFacts([]);
        })
        .finally(() => setFactsLoading(false));

      // Fetch similar entities
      setLoadingSimilar(true);
      setSimilarError(null);
      getSimilarEntities(entity.id)
        .then(setSimilarEntities)
        .catch(() => setSimilarError('Failed to load similar entities'))
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
