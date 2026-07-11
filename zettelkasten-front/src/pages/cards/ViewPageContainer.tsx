import React, { useState, useEffect, useRef } from "react";
import { Card, PartialCard, Entity, RelatedCard } from "../../models/Card";
import { isErrorResponse } from "../../models/common";
import { TaskListItem } from "../../components/tasks/TaskListItem";
import { useTaskContext } from "../../contexts/TaskContext";
import { useUIState } from "../../contexts/UIStateContext";
import { useDialogState } from "../../contexts/DialogStateContext";
import { useParams, useNavigate } from "react-router-dom";

import { CardItem } from "../../components/cards/CardItem";
import { BacklinkInput } from "../../components/cards/BacklinkInput";
import { getCard, CategorizedReferences, getRelatedCards } from "../../api/cards";
import { Menu } from "@headlessui/react";

import { convertCardToPartialCard } from "../../utils/cards";
import {
  calculateNextChildId,
  addTagToCard,
  removeTagFromCard,
  addBacklinkToCard,
  toggleCardStar,
  resummarizeCard
} from "../../utils/cardActions";

import { useTagContext } from "../../contexts/TagContext";
import { useCardData } from "../../hooks/useCardData";
import { useCardNavigation } from "../../hooks/useCardNavigation";

import { fetchSummariesForCard, fetchAnalysisForCard, SectionAnalysis, SummarizeJobResponse } from "../../api/summarizer";
import { FactWithCard } from "../../models/Fact";

interface ViewPageProps {
  cardId?: string; // Optional card ID prop for pinned cards
}

/** Active rendering mode for the card view. */
export type ViewMode = 'normal' | 'summary' | 'analysis';

interface ViewPageContainerData {
  viewingCard: Card | null;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  linkedEntities: Entity[];
  categorizedReferences: CategorizedReferences;
  summaries: SummarizeJobResponse[] | null;
  latestSummary: SummarizeJobResponse | null;
  analysis: SectionAnalysis[] | null;
  relatedCards: RelatedCard[] | null;
  showingSummary: boolean;
  showingAnalysis: boolean;
  showIdDiscovery: boolean;
  isPinned: boolean;
  error: string;
  viewMode: ViewMode;
}

interface ViewPageContainerStateSetters {
  setViewCard: (card: Card | null) => void;
  setError: (error: string) => void;
  setShowingSummary: (showing: boolean) => void;
  setShowingAnalysis: (showing: boolean) => void;
  setViewMode: (mode: ViewMode) => void;
}

interface ViewPageContainerActions {
  onEditCard: () => void;
  onCreateChildCard: () => void;
  onToggleStar: () => void;
  onTogglePin: () => void;
  toggleCreateTaskWindow: () => void;
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  onAddBacklink: (selectedCard: PartialCard) => void;
  handleOpenEntity: (entity: Entity) => void;
  onResummarize: () => Promise<void>;
  onRecategorize: () => void;
  onCloseIdDiscovery: () => void;
  refreshCard: () => void;
}

export function useViewPageContainer({ cardId }: ViewPageProps): {
  data: ViewPageContainerData;
  setters: ViewPageContainerStateSetters;
  actions: ViewPageContainerActions;
} {
  const [error, setError] = useState("");
  const { refreshTasks, setRefreshTasks } = useTaskContext();
  const { refreshFiles, refreshTrigger } = useUIState();
  const { id: urlId } = useParams<{ id: string }>();
  const id = cardId || urlId; // Use prop cardId if provided, otherwise use URL param

  // Track last processed refreshTrigger to prevent infinite loops
  const lastProcessedTriggerRef = useRef<string | null>(null);

  // Use the card data hook for data fetching and state management
  const cardData = useCardData(id);

  // Use the card navigation hook for sibling navigation logic
  const { prevSibling, nextSibling } = useCardNavigation(cardData.parentCard, cardData.viewingCard);

  const fileUploadRef = React.useRef<HTMLInputElement>(null);

  const {
    showCreateTaskWindow,
    setShowCreateTaskWindow,
    setShowEntityDialog,
    setSelectedEntity,
    setSelectedFact,
    setShowFactDialog,
  } = useDialogState();

  const { tags } = useTagContext();
  const { pinnedCard, setPinnedCard } = useUIState();

  const [showingSummary, setShowingSummary] = useState(false);
  const [showingAnalysis, setShowingAnalysis] = useState(false);
  const [showIdDiscovery, setShowIdDiscovery] = useState(false);
  const [relatedCards, setRelatedCards] = useState<RelatedCard[] | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('normal');

  const navigate = useNavigate();

  const { setNextCardId } = useUIState();

  // Handler functions
  function handleOpenEntity(entity: Entity) {
    setSelectedEntity(entity);
    setShowEntityDialog(true);
  }

  async function handleTagClick(tagName: string) {
    if (cardData.viewingCard === null) {
      return;
    }

    await addTagToCard(cardData.viewingCard, tagName, () => {
      if (id) {
        cardData.fetchCard(id);
      }
    });
  }

  async function handleRemoveTag(tagName: string) {
    if (cardData.viewingCard === null) {
      return;
    }

    await removeTagFromCard(cardData.viewingCard, tagName, () => {
      if (id) {
        cardData.fetchCard(id);
      }
    });
  }

  async function handleAddBacklink(selectedCard: PartialCard) {
    if (cardData.viewingCard === null || selectedCard === null) {
      return;
    }

    await addBacklinkToCard(cardData.viewingCard, selectedCard, () => {
      if (id) {
        cardData.fetchCard(id);
      }
    });
  }

  function onEditCard() {
    if (cardData.viewingCard === null) {
      return;
    }
    navigate(`/app/card/${cardData.viewingCard.id}/edit`);
  }

  function handleCreateChildCard() {
    if (cardData.viewingCard === null) return;
    const nextId = calculateNextChildId(cardData.viewingCard.card_id, cardData.viewingCard.children);
    console.log("next id", nextId)
    setNextCardId(nextId);
    navigate('/app/card/new');
  }


  const handleToggleStar = async () => {
    if (cardData.viewingCard === null) {
      return;
    }

    try {
      await toggleCardStar(cardData.viewingCard, () => {
        if (id) {
          cardData.fetchCard(id);
        }
      });
    } catch (error) {
      console.error("Error toggling star status:", error);
      setError("Failed to toggle star status");
    }
  };

  function toggleCreateTaskWindow() {
    setShowCreateTaskWindow(!showCreateTaskWindow);
  }

  const handleTogglePin = () => {
    if (!cardData.viewingCard) return;

    if (pinnedCard && pinnedCard.id === cardData.viewingCard.id) {
      // Unpin if this card is currently pinned
      setPinnedCard(null);
    } else {
      // Pin this card
      setPinnedCard(cardData.viewingCard);
    }
  };

  const isPinned = !!(pinnedCard && cardData.viewingCard && pinnedCard.id === cardData.viewingCard.id);

  const onResummarize = async () => {
    if (cardData.viewingCard) {
      await resummarizeCard(cardData.viewingCard, () => {
        if (id) {
          cardData.fetchCard(id);
        }
      });
    }
  };

  const onRecategorize = () => setShowIdDiscovery(true);
  const onCloseIdDiscovery = () => setShowIdDiscovery(false);
  const refreshCard = () => {
    if (id) {
      cardData.fetchCard(id);
    }
  };

  // useEffects
  useEffect(() => {
    // Reset view states when card changes
    setShowingSummary(false);
    setShowingAnalysis(false);
    setRelatedCards(null);
  }, [id]);

  useEffect(() => {
    // Listen for refreshTrigger changes and fetch card when triggered
    // Use ref to prevent infinite loops from cardData changing on every render
    if (refreshTrigger && id === refreshTrigger && lastProcessedTriggerRef.current !== refreshTrigger) {
      lastProcessedTriggerRef.current = refreshTrigger;
      cardData.fetchCard(id);
    }
  }, [refreshTrigger, id]);

  useEffect(() => {
    // Fetch related cards when viewingCard loads and relatedCards is null
    if (cardData.viewingCard && relatedCards === null) {
      getRelatedCards(cardData.viewingCard.id.toString())
        .then(setRelatedCards)
        .catch(err => console.error("Failed to fetch related cards:", err));
    }
  }, [cardData.viewingCard]);

  // Return data, setters, and actions
  return {
    data: {
      viewingCard: cardData.viewingCard,
      parentCard: cardData.parentCard,
      prevSibling,
      nextSibling,
      linkedEntities: cardData.linkedEntities,
      categorizedReferences: cardData.categorizedReferences,
      summaries: cardData.summaries,
      latestSummary: cardData.latestSummary,
      analysis: cardData.analysis,
      relatedCards,
      showingSummary,
      showingAnalysis,
      showIdDiscovery,
      isPinned,
      error,
      viewMode,
    },
    setters: {
      setViewCard: cardData.setViewingCard,
      setError,
      setShowingSummary,
      setShowingAnalysis,
      setViewMode,
    },
    actions: {
      onEditCard,
      onCreateChildCard: handleCreateChildCard,
      onToggleStar: handleToggleStar,
      onTogglePin: handleTogglePin,
      toggleCreateTaskWindow,
      onTagClick: handleTagClick,
      onRemoveTag: handleRemoveTag,
      onAddBacklink: handleAddBacklink,
      handleOpenEntity,
      onResummarize,
      onRecategorize,
      onCloseIdDiscovery,
      refreshCard,
    }
  };
}