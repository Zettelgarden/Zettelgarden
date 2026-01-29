import React, { useState, useEffect } from "react";
import { Card, PartialCard, Entity } from "../../models/Card";
import { isErrorResponse } from "../../models/common";
import { TaskListItem } from "../../components/tasks/TaskListItem";
import { useTaskContext } from "../../contexts/TaskContext";
import { useFileContext } from "../../contexts/FileContext";
import { useParams, useNavigate } from "react-router-dom";

import { CardItem } from "../../components/cards/CardItem";
import { BacklinkInput } from "../../components/cards/BacklinkInput";
import { getCard, CategorizedReferences } from "../../api/cards";
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

import { usePartialCardContext } from "../../contexts/CardContext";
import { useCardRefresh } from "../../contexts/CardRefreshContext";
import { useShortcutContext } from "../../contexts/ShortcutContext";
import { useTagContext } from "../../contexts/TagContext";
import { usePinContext } from "../../contexts/PinContext";
import { useChatSidebarContext } from "../../contexts/ChatSidebarContext";
import { useCardData } from "../../hooks/useCardData";
import { useCardNavigation } from "../../hooks/useCardNavigation";

import { PinButton } from "../../components/cards/PinButton";
import { fetchSummariesForCard, fetchAnalysisForCard, SectionAnalysis, SummarizeJobResponse } from "../../api/summarizer";
import { FactWithCard } from "../../models/Fact";

interface ViewPageProps {
  cardId?: string; // Optional card ID prop for pinned cards
}

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
  showingSummary: boolean;
  showingAnalysis: boolean;
  showIdDiscovery: boolean;
  error: string;
}

interface ViewPageContainerStateSetters {
  setViewCard: (card: Card | null) => void;
  setError: (error: string) => void;
  setShowingSummary: (showing: boolean) => void;
  setShowingAnalysis: (showing: boolean) => void;
}

interface ViewPageContainerActions {
  onEditCard: () => void;
  onCreateChildCard: () => void;
  onToggleStar: () => void;
  onTogglePin: () => void;
  onOpenChatSidebar: () => void;
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
  const { refreshFiles } = useFileContext();
  const { id: urlId } = useParams<{ id: string }>();
  const id = cardId || urlId; // Use prop cardId if provided, otherwise use URL param
  const { refreshTrigger } = useCardRefresh();

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
  } = useShortcutContext();

  const { tags } = useTagContext();
  const { pinnedCard, setPinnedCard } = usePinContext();
  const { setChatSidebarCard } = useChatSidebarContext();

  const [showingSummary, setShowingSummary] = useState(false);
  const [showingAnalysis, setShowingAnalysis] = useState(false);
  const [showIdDiscovery, setShowIdDiscovery] = useState(false);


  const navigate = useNavigate();

  const { setNextCardId } = usePartialCardContext();

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

  const handleOpenChatSidebar = () => {
    if (!cardData.viewingCard) return;
    setChatSidebarCard(cardData.viewingCard);
  };

  const isPinned = pinnedCard && cardData.viewingCard && pinnedCard.id === cardData.viewingCard.id;

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
  }, [id]);


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
      showingSummary,
      showingAnalysis,
      showIdDiscovery,
      error,
    },
    setters: {
      setViewCard: cardData.setViewingCard,
      setError,
      setShowingSummary,
      setShowingAnalysis,
    },
    actions: {
      onEditCard,
      onCreateChildCard: handleCreateChildCard,
      onToggleStar: handleToggleStar,
      onTogglePin: handleTogglePin,
      onOpenChatSidebar: handleOpenChatSidebar,
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