import React, { useState, useEffect } from "react";
import { setDocumentTitle } from "../../utils/title";
import { Card, PartialCard, Entity } from "../../models/Card";
import { isErrorResponse } from "../../models/common";
import { TaskListItem } from "../../components/tasks/TaskListItem";
import { useTaskContext } from "../../contexts/TaskContext";
import { useFileContext } from "../../contexts/FileContext";
import { useParams, useNavigate } from "react-router-dom";

import { CardItem } from "../../components/cards/CardItem";
import { BacklinkInput } from "../../components/cards/BacklinkInput";
import { getCard, saveExistingCard, starCard, unstarCard, getCardReferences, getCardChildren, getCardFiles, getCardTags, getCardTasks, getCardEntities, getLinkedEntitiesByCardPK, CategorizedReferences } from "../../api/cards";
import { Menu } from "@headlessui/react";

import { convertCardToPartialCard } from "../../utils/cards";
import { findNextChildId } from "../../utils/cards";

import { usePartialCardContext } from "../../contexts/CardContext";
import { useCardRefresh } from "../../contexts/CardRefreshContext";
import { useShortcutContext } from "../../contexts/ShortcutContext";
import { useTagContext } from "../../contexts/TagContext";
import { usePinContext } from "../../contexts/PinContext";
import { useChatSidebarContext } from "../../contexts/ChatSidebarContext";

import { PinButton } from "../../components/cards/PinButton";
import { compareCardIds } from "../../utils/cards";
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
  const [viewingCard, setViewCard] = useState<Card | null>(null);
  const [parentCard, setParentCard] = useState<Card | null>(null);
  const { refreshTasks, setRefreshTasks } = useTaskContext();
  const { refreshFiles } = useFileContext();
  const { id: urlId } = useParams<{ id: string }>();
  const id = cardId || urlId; // Use prop cardId if provided, otherwise use URL param
  const { refreshTrigger } = useCardRefresh();

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

  const [summaries, setSummaries] = useState<SummarizeJobResponse[] | null>(null);
  const [latestSummary, setLatestSummary] = useState<SummarizeJobResponse | null>(null);
  const [showingSummary, setShowingSummary] = useState(false);
  const [analysis, setAnalysis] = useState<SectionAnalysis[] | null>(null);
  const [showingAnalysis, setShowingAnalysis] = useState(false);
  const [showIdDiscovery, setShowIdDiscovery] = useState(false);

  const [prevSibling, setPrevSibling] = useState<PartialCard | null>(null);
  const [nextSibling, setNextSibling] = useState<PartialCard | null>(null);

  const [linkedEntities, setLinkedEntities] = useState<Entity[]>([]);
  const [categorizedReferences, setCategorizedReferences] = useState<CategorizedReferences>({
    bidirectional: [],
    outgoing: [],
    incoming: [],
  });

  const navigate = useNavigate();

  const { setLastCard, setNextCardId } = usePartialCardContext();

  // Handler functions
  function handleOpenEntity(entity: Entity) {
    setSelectedEntity(entity);
    setShowEntityDialog(true);
  }

  async function handleTagClick(tagName: string) {
    if (viewingCard === null) {
      return;
    }

    let editedCard = {
      ...viewingCard,
      body: viewingCard.body + "\n\n#" + tagName,
    };
    let response = await saveExistingCard(editedCard);
    setViewCard(editedCard);
  }

  async function handleRemoveTag(tagName: string) {
    if (viewingCard === null) {
      return;
    }

    const tagRegex = new RegExp(`\\n*#${tagName}\\b`, 'g');
    let editedCard = {
      ...viewingCard,
      body: viewingCard.body.replace(tagRegex, ''),
    };
    let response = await saveExistingCard(editedCard);
    setViewCard(editedCard);
    fetchCard(id!);
  }

  async function handleAddBacklink(selectedCard: PartialCard) {
    if (viewingCard === null || selectedCard === null) {
      return;
    }
    let text = "";
    if (selectedCard) {
      text = "\n\n[" + selectedCard.card_id + "] - " + selectedCard.title;
    } else {
      text = "";
    }
    let editedCard = {
      ...viewingCard,
      body: viewingCard.body + text,
    };
    let response = await saveExistingCard(editedCard);
    setViewCard(editedCard);
    fetchCard(id!);
  }

  function onEditCard() {
    if (viewingCard === null) {
      return;
    }
    navigate(`/app/card/${viewingCard.id}/edit`);
  }

  function handleCreateChildCard() {
    if (viewingCard === null) return;
    const nextId = findNextChildId(viewingCard.card_id, viewingCard.children);
    setNextCardId(nextId);
    navigate('/app/card/new');
  }

  async function loadSummaries(id: number) {
    try {
      const jobs = await fetchSummariesForCard(id);
      setSummaries(jobs);
    } catch (err: any) {
      console.error("Failed to fetch summaries", err);
    }
  }

  async function loadAnalysis(id: number) {
    try {
      const analysisData = await fetchAnalysisForCard(id);
      setAnalysis(analysisData);
    } catch (err: any) {
      console.error("Failed to fetch analysis", err);
    }
  }

  async function fetchCard(id: string) {
    try {
      let refreshed = await getCard(id);

      if (isErrorResponse(refreshed)) {
        setError(refreshed["error"]);
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
          const parentCard = await getCard(parentCardId.toString());
          let parentChildren = await getCardChildren(parentCardId.toString());

          parentCard.children = parentChildren;
          console.log("set parent", parentCard, "children", parentChildren);
          setParentCard(parentCard);
        } else {
          setParentCard(null);
        }
      }
    } catch (error: any) {
      setError(error.message);
    }
  }

  const handleToggleStar = async () => {
    if (viewingCard === null) {
      return;
    }
    console.log("?", viewingCard);
    const card = viewingCard;
    try {
      console.log(viewingCard, viewingCard.is_starred);
      if (viewingCard.is_starred) {
        await unstarCard(viewingCard.id);
        setViewCard({
          ...card,
          is_starred: false
        });
      } else {
        await starCard(viewingCard.id);
        setViewCard({
          ...card,
          is_starred: true
        });
      }
    } catch (error) {
      console.log(error);
    }
  };

  function toggleCreateTaskWindow() {
    setShowCreateTaskWindow(!showCreateTaskWindow);
  }

  const handleTogglePin = () => {
    if (!viewingCard) return;

    if (pinnedCard && pinnedCard.id === viewingCard.id) {
      // Unpin if this card is currently pinned
      setPinnedCard(null);
    } else {
      // Pin this card
      setPinnedCard(viewingCard);
    }
  };

  const handleOpenChatSidebar = () => {
    if (!viewingCard) return;
    setChatSidebarCard(viewingCard);
  };

  const isPinned = pinnedCard && viewingCard && pinnedCard.id === viewingCard.id;

  const onResummarize = async () => {
    if (viewingCard) {
      const updatedCard = {
        ...viewingCard,
        process_entities_and_facts: true
      };
      await saveExistingCard(updatedCard);
      fetchCard(id!);
    }
  };

  const onRecategorize = () => setShowIdDiscovery(true);
  const onCloseIdDiscovery = () => setShowIdDiscovery(false);
  const refreshCard = () => fetchCard(id!);

  // useEffects
  useEffect(() => {
    setError("");
    // Reset view states when card changes
    setShowingSummary(false);
    setShowingAnalysis(false);
    fetchCard(id!);
    if (id) {
      loadSummaries(parseInt(id));
      loadAnalysis(parseInt(id));
    }
  }, [id, refreshTasks, refreshFiles, refreshTrigger]);

  useEffect(() => {
    if (summaries && summaries.length > 0) {
      // Filter to only "complete" summaries
      const completeSummaries = summaries.filter(s => s.status === "complete");

      if (completeSummaries.length > 0) {
        // Find the one with the highest ID
        const latest = completeSummaries.reduce((max, s) =>
          s.id > max.id ? s : max
        );

        setLatestSummary(latest);
      } else {
        // Optional: fallback if none are "complete"
        console.log("No complete summaries yet");
        setLatestSummary(null);
      }
    }
  }, [summaries]);

  // Calculate previous and next siblings
  useEffect(() => {
    if (parentCard && viewingCard) {
      const siblings = parentCard.children.sort((a, b) =>
        compareCardIds(a.card_id, b.card_id)
      );
      const currentIndex = siblings.findIndex(s => s.id === viewingCard.id);

      if (currentIndex !== -1) {
        setPrevSibling(currentIndex > 0 ? siblings[currentIndex - 1] : null);
        setNextSibling(currentIndex < siblings.length - 1 ? siblings[currentIndex + 1] : null);
      } else {
        setPrevSibling(null);
        setNextSibling(null);
      }
    } else {
      setPrevSibling(null);
      setNextSibling(null);
    }
  }, [parentCard, viewingCard]);

  // Return data, setters, and actions
  return {
    data: {
      viewingCard,
      parentCard,
      prevSibling,
      nextSibling,
      linkedEntities,
      categorizedReferences,
      summaries,
      latestSummary,
      analysis,
      showingSummary,
      showingAnalysis,
      showIdDiscovery,
      error,
    },
    setters: {
      setViewCard,
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