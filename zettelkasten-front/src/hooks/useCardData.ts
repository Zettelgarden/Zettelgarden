import { useState, useEffect } from "react";
import { setDocumentTitle } from "../utils/title";
import { Card, PartialCard, Entity } from "../models/Card";
import { isErrorResponse } from "../models/common";
import { convertCardToPartialCard } from "../utils/cards";
import { useUIState } from "../contexts/UIStateContext";
import { fetchSummariesForCard, SummarizeJobResponse } from "../api/summarizer";
import { getCard, getCardReferences, getCardChildren, getCardFiles, getCardTags, getCardTasks, getCardEntities, getLinkedEntitiesByCardPK, CategorizedReferences } from "../api/cards";

export interface UseCardDataResult {
  // Data state
  viewingCard: Card | null;
  parentCard: Card | null;
  linkedEntities: Entity[];
  categorizedReferences: CategorizedReferences;
  summaries: SummarizeJobResponse[] | null;
  latestSummary: SummarizeJobResponse | null;

  // Loading/fetching functions
  fetchCard: (id: string) => Promise<void>;
  loadSummaries: (id: number) => Promise<void>;

  // State setters (for optimistic updates)
  setViewingCard: (card: Card | null) => void;
}

export function useCardData(cardId?: string): UseCardDataResult {
  const [viewingCard, setViewCard] = useState<Card | null>(null);
  const [parentCard, setParentCard] = useState<Card | null>(null);
  const [linkedEntities, setLinkedEntities] = useState<Entity[]>([]);
  const [categorizedReferences, setCategorizedReferences] = useState<CategorizedReferences>({
    bidirectional: [],
    outgoing: [],
    incoming: [],
  });
  const [summaries, setSummaries] = useState<SummarizeJobResponse[] | null>(null);

  const { setLastCard } = useUIState();

  // Derive latestSummary from summaries
  const latestSummary = summaries ? (() => {
    // Filter to only "complete" summaries
    const completeSummaries = summaries.filter(s => s.status === "complete");

    if (completeSummaries.length > 0) {
      // Find the one with the highest ID
      return completeSummaries.reduce((max, s) =>
        s.id > max.id ? s : max
      );
    }

    return null;
  })() : null;

  async function loadSummaries(id: number) {
    try {
      const jobs = await fetchSummariesForCard(id);
      setSummaries(jobs);
    } catch (err: any) {
      // Silently handle case where card has no summaries - this is expected
      const errorMessage = err?.message || String(err);
      if (errorMessage.includes("no rows in result set") || errorMessage.includes("failed to find summarization")) {
        // No summaries exist for this card, which is expected
        setSummaries(null);
      } else {
        console.error("Failed to fetch summaries", err);
      }
    }
  }

  async function fetchCard(id: string) {
    try {
      let refreshed = await getCard(id);

      if (isErrorResponse(refreshed)) {
        // Handle error at the hook level - for now, just log and continue
        // Error handling will be managed by the consuming component
        console.error("Error fetching card:", refreshed.error);
        return;
      } else {
        // Also fetch categorized references via new endpoint
        const refs = await getCardReferences(id);
        setCategorizedReferences(refs);
        // Combine all references for backward compatibility with Card.references
        refreshed.references = [...refs.bidirectional, ...refs.outgoing, ...refs.incoming];
        // Also fetch children via new endpoint
        const kids = await getCardChildren(id);
        refreshed.children = kids;
        // Also fetch files via new endpoint
        const files = await getCardFiles(id);
        refreshed.files = files;
        // Also fetch tags via new endpoint
        const tags = await getCardTags(id);
        refreshed.tags = tags;
        // Also fetch tasks via new endpoint
        const tasks = await getCardTasks(id);
        refreshed.tasks = tasks;
        // Also fetch entities via new endpoint
        const entities = await getCardEntities(id);
        refreshed.entities = entities;

        // Also fetch linked entities via new endpoint
        const linked = await getLinkedEntitiesByCardPK(id);
        setLinkedEntities(linked);

        setViewCard(refreshed);
        setDocumentTitle(refreshed.card_id + " - " + refreshed.title);
        setLastCard(convertCardToPartialCard(refreshed));

        if (refreshed.parent && "id" in refreshed.parent) {
          let parentCardId = refreshed.parent.id;
          const parentCardData = await getCard(parentCardId.toString());
          let parentChildren = await getCardChildren(parentCardId.toString());

          parentCardData.children = parentChildren;
          setParentCard(parentCardData);
        } else {
          setParentCard(null);
        }
      }
    } catch (error: any) {
      console.error("Error fetching card:", error);
      // Error handling will be managed by the consuming component
    }
  }

  // Auto-fetch card when cardId changes
  useEffect(() => {
    if (cardId) {
      fetchCard(cardId);
      loadSummaries(parseInt(cardId));
    }
  }, [cardId]);

  return {
    viewingCard,
    parentCard,
    linkedEntities,
    categorizedReferences,
    summaries,
    latestSummary,
    fetchCard,
    loadSummaries,
    setViewingCard: setViewCard,
  };
}