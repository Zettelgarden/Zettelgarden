import React, { useState, useEffect, useRef } from 'react';
import {
  Card,
  PartialCard,
  Entity,
  RelatedCard,
  UnlinkedMention,
} from '../../models/Card';
import { isErrorResponse } from '../../models/common';
import { TaskListItem } from '../../components/tasks/TaskListItem';
import { useTaskContext } from '../../contexts/TaskContext';
import { useUIState } from '../../contexts/UIStateContext';
import { useDialogState } from '../../contexts/DialogStateContext';
import { useParams, useNavigate } from 'react-router-dom';

import { CardItem } from '../../components/cards/CardItem';
import { BacklinkInput } from '../../components/cards/BacklinkInput';
import {
  getCard,
  CategorizedReferences,
  getRelatedCards,
  getUnlinkedMentions,
  getCardSuggestions,
} from '../../api/cards';
import { Menu } from '@headlessui/react';

import { convertCardToPartialCard } from '../../utils/cards';
import {
  calculateNextChildId,
  addTagToCard,
  removeTagFromCard,
  addBacklinkToCard,
  toggleCardStar,
  resummarizeCard,
} from '../../utils/cardActions';

import { useTagContext } from '../../contexts/TagContext';
import { useCardData } from '../../hooks/useCardData';
import { useCardNavigation } from '../../hooks/useCardNavigation';

import {
  fetchSummariesForCard,
  SummarizeJobResponse,
} from '../../api/summarizer';
import { FactWithCard } from '../../models/Fact';

interface ViewPageProps {
  cardId?: string; // Optional card ID prop
}

/** Active rendering mode for the card view. */
export type ViewMode = 'normal' | 'summary';

interface ViewPageContainerData {
  viewingCard: Card | null;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  linkedEntities: Entity[];
  categorizedReferences: CategorizedReferences;
  summaries: SummarizeJobResponse[] | null;
  latestSummary: SummarizeJobResponse | null;
  relatedCards: RelatedCard[] | null;
  unlinkedMentions: UnlinkedMention[] | null;
  suggestions: RelatedCard[] | null;
  showingSummary: boolean;
  showIdDiscovery: boolean;
  error: string;
  viewMode: ViewMode;
}

interface ViewPageContainerStateSetters {
  setViewCard: (card: Card | null) => void;
  setError: (error: string) => void;
  setShowingSummary: (showing: boolean) => void;
  setViewMode: (mode: ViewMode) => void;
}

interface ViewPageContainerActions {
  onEditCard: () => void;
  onCreateChildCard: () => void;
  onToggleStar: () => void;
  toggleCreateTaskWindow: () => void;
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  onAddBacklink: (selectedCard: PartialCard) => void;
  handleOpenEntity: (entity: Entity) => void;
  onResummarize: () => Promise<void>;
  onRecategorize: () => void;
  onCloseIdDiscovery: () => void;
  refreshCard: () => void;
  refreshRelatedCards: () => void;
  refreshUnlinkedMentions: () => void;
  addUnlinkedMentionLink: (mention: UnlinkedMention) => Promise<void>;
}

export function useViewPageContainer({ cardId }: ViewPageProps): {
  data: ViewPageContainerData;
  setters: ViewPageContainerStateSetters;
  actions: ViewPageContainerActions;
} {
  const [error, setError] = useState('');
  const { refreshTasks, setRefreshTasks } = useTaskContext();
  const { refreshFiles, refreshTrigger } = useUIState();
  const { id: urlId } = useParams<{ id: string }>();
  const id = cardId || urlId; // Use prop cardId if provided, otherwise use URL param

  // Track last processed refreshTrigger to prevent infinite loops
  const lastProcessedTriggerRef = useRef<string | null>(null);

  // Use the card data hook for data fetching and state management
  const cardData = useCardData(id);

  // Use the card navigation hook for sibling navigation logic
  const { prevSibling, nextSibling } = useCardNavigation(
    cardData.parentCard,
    cardData.viewingCard,
  );

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

  const [showingSummary, setShowingSummary] = useState(false);
  const [showIdDiscovery, setShowIdDiscovery] = useState(false);
  const [relatedCards, setRelatedCards] = useState<RelatedCard[] | null>(null);
  const [suggestions, setSuggestions] = useState<RelatedCard[] | null>(null);
  const [unlinkedMentions, setUnlinkedMentions] = useState<
    UnlinkedMention[] | null
  >(null);
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
    const nextId = calculateNextChildId(
      cardData.viewingCard.card_id,
      cardData.viewingCard.children,
    );
    console.log('next id', nextId);
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
      console.error('Error toggling star status:', error);
      setError('Failed to toggle star status');
    }
  };

  function toggleCreateTaskWindow() {
    setShowCreateTaskWindow(!showCreateTaskWindow);
  }

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

  /** Invalidate cached related cards so the fetch effect repopulates them. */
  const refreshRelatedCards = () => {
    setRelatedCards(null);
    // Suggestions are driven by references, which change on body save, so
    // refresh them together.
    setSuggestions(null);
  };

  /** Invalidate cached unlinked mentions so the fetch effect repopulates them. */
  const refreshUnlinkedMentions = () => {
    setUnlinkedMentions(null);
  };

  /**
   * Insert a wiki-link from the mentioning card to the viewing card, then
   * drop it from the list (the backend will also exclude it on refetch).
   */
  async function handleAddUnlinkedMentionLink(mention: UnlinkedMention) {
    if (cardData.viewingCard === null) {
      return;
    }
    try {
      const fullMentionCard = await getCard(mention.card.id.toString());
      await addBacklinkToCard(fullMentionCard, cardData.viewingCard, () => {
        if (id) {
          cardData.fetchCard(id);
        }
      });
      setUnlinkedMentions(
        (prev) => prev?.filter((m) => m.card.id !== mention.card.id) ?? null,
      );
    } catch (err) {
      console.error('Failed to add unlinked mention link:', err);
      setError('Failed to add link');
    }
  }

  // useEffects
  useEffect(() => {
    // Reset view states when card changes
    setShowingSummary(false);
    setRelatedCards(null);
    setSuggestions(null);
    setUnlinkedMentions(null);
  }, [id]);

  useEffect(() => {
    // Listen for refreshTrigger changes and fetch card when triggered
    // Use ref to prevent infinite loops from cardData changing on every render
    if (
      refreshTrigger &&
      id === refreshTrigger &&
      lastProcessedTriggerRef.current !== refreshTrigger
    ) {
      lastProcessedTriggerRef.current = refreshTrigger;
      cardData.fetchCard(id);
    }
  }, [refreshTrigger, id]);

  useEffect(() => {
    // Fetch related cards when viewingCard loads and relatedCards is null.
    // relatedCards is a dependency so a refreshRelatedCards() invalidation
    // (reset to null) triggers a fresh fetch, not just the initial load.
    if (cardData.viewingCard && relatedCards === null) {
      getRelatedCards(cardData.viewingCard.id.toString())
        .then(setRelatedCards)
        .catch((err) => console.error('Failed to fetch related cards:', err));
    }
  }, [cardData.viewingCard, relatedCards]);

  useEffect(() => {
    // Fetch unlinked mentions when viewingCard loads and the cache is null.
    if (cardData.viewingCard && unlinkedMentions === null) {
      getUnlinkedMentions(cardData.viewingCard.id.toString())
        .then(setUnlinkedMentions)
        .catch((err) =>
          console.error('Failed to fetch unlinked mentions:', err),
        );
    }
  }, [cardData.viewingCard, unlinkedMentions]);

  useEffect(() => {
    // Fetch second-degree suggestions when viewingCard loads and cache is null.
    if (cardData.viewingCard && suggestions === null) {
      getCardSuggestions(cardData.viewingCard.id.toString())
        .then(setSuggestions)
        .catch((err) => console.error('Failed to fetch suggestions:', err));
    }
  }, [cardData.viewingCard, suggestions]);

  // Filter out related cards that already appear in the card's references so
  // the Related Cards list doesn't duplicate the Linked references section.
  // References can change after the related-cards fetch (e.g. adding a
  // backlink from the +Ref button), so filter on every render rather than
  // relying on the backend's initial exclusion alone.
  const referenceCardIds = new Set(
    [
      ...cardData.categorizedReferences.bidirectional,
      ...cardData.categorizedReferences.incoming,
      ...cardData.categorizedReferences.outgoing,
    ].map((ref) => ref.id),
  );
  const filteredRelatedCards =
    relatedCards?.filter((rc) => !referenceCardIds.has(rc.card.id)) ?? null;

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
      relatedCards: filteredRelatedCards,
      unlinkedMentions,
      suggestions,
      showingSummary,
      showIdDiscovery,
      error,
      viewMode,
    },
    setters: {
      setViewCard: cardData.setViewingCard,
      setError,
      setShowingSummary,
      setViewMode,
    },
    actions: {
      onEditCard,
      onCreateChildCard: handleCreateChildCard,
      onToggleStar: handleToggleStar,
      toggleCreateTaskWindow,
      onTagClick: handleTagClick,
      onRemoveTag: handleRemoveTag,
      onAddBacklink: handleAddBacklink,
      handleOpenEntity,
      onResummarize,
      onRecategorize,
      onCloseIdDiscovery,
      refreshCard,
      refreshRelatedCards,
      refreshUnlinkedMentions,
      addUnlinkedMentionLink: handleAddUnlinkedMentionLink,
    },
  };
}
