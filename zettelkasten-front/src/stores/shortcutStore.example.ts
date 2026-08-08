/**
 * PROOF OF CONCEPT: Zustand for UI State
 *
 * This file demonstrates how ShortcutProvider, ChatProvider, and PartialCardProvider
 * can be replaced with Zustand stores.
 *
 * Benefits:
 * - No provider nesting needed
 * - Selective subscriptions prevent unnecessary re-renders
 * - Less boilerplate code
 * - Easy to test
 * - Works outside React components
 */

import { create } from 'zustand';
import { Card, PartialCard, Entity } from '../models/Card';
import { FactWithCard } from '../models/Fact';

// =============================================================================
// SHORTCUT STORE
// =============================================================================

/**
 * Replaces: ShortcutProvider (92 lines → ~50 lines)
 *
 * Before:
 *   const {
 *     showCreateTaskWindow,
 *     setShowCreateTaskWindow,
 *     showQuickSearchWindow,
 *     setShowQuickSearchWindow,
 *     // ... 8 more state variables
 *   } = useShortcutContext();
 *
 * After:
 *   const showCreateTaskWindow = useShortcutStore(s => s.showCreateTaskWindow);
 *   const setShowCreateTaskWindow = useShortcutStore(s => s.setShowCreateTaskWindow);
 *
 * Or (for multiple values):
 *   const { showCreateTaskWindow, setShowCreateTaskWindow } = useShortcutStore();
 */

interface ShortcutState {
  // Dialog visibility states
  showCreateTaskWindow: boolean;
  showQuickSearchWindow: boolean;
  showEntityDialog: boolean;
  showFactDialog: boolean;
  showTaskDialog: boolean;

  // Selection states
  selectedEntity: Entity | null;
  selectedFact: FactWithCard | null;
  selectedTaskId: number | null;
}

interface ShortcutActions {
  // Setters for visibility
  setShowCreateTaskWindow: (show: boolean) => void;
  setShowQuickSearchWindow: (show: boolean) => void;
  setShowEntityDialog: (show: boolean) => void;
  setShowFactDialog: (show: boolean) => void;
  setShowTaskDialog: (show: boolean) => void;

  // Setters for selections
  setSelectedEntity: (entity: Entity | null) => void;
  setSelectedFact: (fact: FactWithCard | null) => void;
  setSelectedTaskId: (taskId: number | null) => void;

  // Combined actions
  openEntityDialog: (entity: Entity) => void;
  openFactDialog: (fact: FactWithCard) => void;
  openTaskDialog: (taskId: number) => void;
  closeAllDialogs: () => void;
}

type ShortcutStore = ShortcutState & ShortcutActions;

export const useShortcutStore = create<ShortcutStore>((set) => ({
  // Initial state
  showCreateTaskWindow: false,
  showQuickSearchWindow: false,
  showEntityDialog: false,
  showFactDialog: false,
  showTaskDialog: false,
  selectedEntity: null,
  selectedFact: null,
  selectedTaskId: null,

  // Setters
  setShowCreateTaskWindow: (show) => set({ showCreateTaskWindow: show }),
  setShowQuickSearchWindow: (show) => set({ showQuickSearchWindow: show }),
  setShowEntityDialog: (show) => set({ showEntityDialog: show }),
  setShowFactDialog: (show) => set({ showFactDialog: show }),
  setShowTaskDialog: (show) => set({ showTaskDialog: show }),
  setSelectedEntity: (entity) => set({ selectedEntity: entity }),
  setSelectedFact: (fact) => set({ selectedFact: fact }),
  setSelectedTaskId: (taskId) => set({ selectedTaskId: taskId }),

  // Combined actions
  openEntityDialog: (entity) =>
    set({ showEntityDialog: true, selectedEntity: entity }),
  openFactDialog: (fact) => set({ showFactDialog: true, selectedFact: fact }),
  openTaskDialog: (taskId) =>
    set({ showTaskDialog: true, selectedTaskId: taskId }),
  closeAllDialogs: () =>
    set({
      showCreateTaskWindow: false,
      showQuickSearchWindow: false,
      showEntityDialog: false,
      showFactDialog: false,
      showTaskDialog: false,
    }),
}));

// Selectors for optimized re-renders
export const useShowCreateTaskWindow = () =>
  useShortcutStore((state) => state.showCreateTaskWindow);

export const useSelectedEntity = () =>
  useShortcutStore((state) => state.selectedEntity);

export const useShortcutActions = () =>
  useShortcutStore((state) => ({
    setShowCreateTaskWindow: state.setShowCreateTaskWindow,
    setShowQuickSearchWindow: state.setShowQuickSearchWindow,
    setShowEntityDialog: state.setShowEntityDialog,
    setShowFactDialog: state.setShowFactDialog,
    setShowTaskDialog: state.setShowTaskDialog,
    openEntityDialog: state.openEntityDialog,
    openFactDialog: state.openFactDialog,
    openTaskDialog: state.openTaskDialog,
    closeAllDialogs: state.closeAllDialogs,
  }));

// =============================================================================
// CARD STORE
// =============================================================================

/**
 * Replaces: PartialCardProvider + PinProvider (44 + 44 lines → ~60 lines)
 *
 * Combines card-related UI state into one store
 */

interface CardState {
  // From PartialCardProvider
  lastCard: PartialCard | null;
  nextCardId: string | null;

  // From PinProvider
  pinnedCard: Card | null;
  isPinMode: boolean;

  // From ChatSidebarProvider
  chatSidebarCard: Card | null;
  isChatSidebarMode: boolean;
}

interface CardActions {
  // Setters from PartialCardProvider
  setLastCard: (card: PartialCard) => void;
  setNextCardId: (id: string | null) => void;

  // Setters from PinProvider
  setPinnedCard: (card: Card | null) => void;
  setIsPinMode: (mode: boolean) => void;

  // Setters from ChatSidebarProvider
  setChatSidebarCard: (card: Card | null) => void;
  setIsChatSidebarMode: (mode: boolean) => void;

  // Combined actions
  clearLastCard: () => void;
  unpinCard: () => void;
  clearChatSidebar: () => void;
}

type CardStore = CardState & CardActions;

export const useCardStore = create<CardStore>((set, get) => ({
  // Initial state
  lastCard: null,
  nextCardId: null,
  pinnedCard: null,
  isPinMode: false,
  chatSidebarCard: null,
  isChatSidebarMode: false,

  // Setters
  setLastCard: (card) => set({ lastCard: card }),
  setNextCardId: (id) => set({ nextCardId: id }),
  setPinnedCard: (card) => {
    set({ pinnedCard: card });
    // Auto-enable pin mode (replaces useEffect in PinProvider)
    set({ isPinMode: card !== null });
  },
  setIsPinMode: (mode) => set({ isPinMode: mode }),
  setChatSidebarCard: (card) => {
    set({ chatSidebarCard: card });
    // Auto-enable chat sidebar mode (replaces useEffect in ChatSidebarProvider)
    set({ isChatSidebarMode: card !== null });
  },
  setIsChatSidebarMode: (mode) => set({ isChatSidebarMode: mode }),

  // Combined actions
  clearLastCard: () => set({ lastCard: null }),
  unpinCard: () => set({ pinnedCard: null, isPinMode: false }),
  clearChatSidebar: () =>
    set({ chatSidebarCard: null, isChatSidebarMode: false }),
}));

// Selectors for optimized re-renders
export const useLastCard = () => useCardStore((state) => state.lastCard);

export const usePinnedCard = () =>
  useCardStore((state) => ({
    pinnedCard: state.pinnedCard,
    isPinMode: state.isPinMode,
  }));

export const useChatSidebarCard = () =>
  useCardStore((state) => ({
    chatSidebarCard: state.chatSidebarCard,
    isChatSidebarMode: state.isChatSidebarMode,
  }));

// =============================================================================
// CHAT STORE
// =============================================================================

/**
 * Replaces: ChatProvider (39 lines → ~20 lines)
 *
 * Before:
 *   const { conversationId, setConversationId, showChat, setShowChat } = useChatContext();
 *
 * After:
 *   const { conversationId, setConversationId, showChat, setShowChat } = useChatStore();
 */

interface ChatState {
  conversationId: string;
  showChat: boolean;
}

interface ChatActions {
  setConversationId: (id: string) => void;
  setShowChat: (show: boolean) => void;
  startNewChat: () => void;
  endChat: () => void;
}

type ChatStore = ChatState & ChatActions;

export const useChatStore = create<ChatStore>((set) => ({
  // Initial state
  conversationId: '',
  showChat: false,

  // Setters
  setConversationId: (id) => set({ conversationId: id }),
  setShowChat: (show) => set({ showChat: show }),

  // Combined actions
  startNewChat: () => set({ conversationId: '', showChat: true }),
  endChat: () => set({ conversationId: '', showChat: false }),
}));

// Selectors
export const useConversationId = () =>
  useChatStore((state) => state.conversationId);

export const useShowChat = () => useChatStore((state) => state.showChat);

// =============================================================================
// USAGE EXAMPLES
// =============================================================================

/**
 * Example 1: Using individual selectors (optimized)
 *
 * Only re-renders when showCreateTaskWindow changes
 *
 * Before:
 *   const { showCreateTaskWindow } = useShortcutContext();
 *   // Re-renders on ANY change in ShortcutContext
 *
 * After:
 *   const showCreateTaskWindow = useShortcutStore(s => s.showCreateTaskWindow);
 *   // Only re-renders when showCreateTaskWindow changes
 */

/**
 * Example 2: Using multiple values
 *
 * Before:
 *   const {
 *     showEntityDialog,
 *     selectedEntity,
 *     setShowEntityDialog,
 *   } = useShortcutContext();
 *
 * After:
 *   const { showEntityDialog, selectedEntity, setShowEntityDialog } = useShortcutStore();
 */

/**
 * Example 3: Using combined actions
 *
 * Before:
 *   const { setShowEntityDialog, setSelectedEntity } = useShortcutContext();
 *   setShowEntityDialog(true);
 *   setSelectedEntity(entity);
 *
 * After:
 *   const { openEntityDialog } = useShortcutStore();
 *   openEntityDialog(entity); // One call instead of two
 */

/**
 * Example 4: Card state from combined store
 *
 * Before (3 different contexts):
 *   const { lastCard } = usePartialCardContext();
 *   const { pinnedCard, isPinMode } = usePinContext();
 *   const { chatSidebarCard, isChatSidebarMode } = useChatSidebarContext();
 *
 * After (1 store):
 *   const { lastCard, pinnedCard, isPinMode, chatSidebarCard, isChatSidebarMode } = useCardStore();
 *
 * Or (optimized):
 *   const lastCard = useCardStore(s => s.lastCard);
 *   const { pinnedCard, isPinMode } = usePinnedCard(); // Custom hook
 */

/**
 * Example 5: Using Zustand outside React
 *
 * Zustand stores can be used outside React components!
 *
 * // In a utility function:
 * export function openQuickSearch() {
 *   useShortcutStore.getState().setShowQuickSearchWindow(true);
 * }
 *
 * // In a keyboard shortcut handler:
 * document.addEventListener('keydown', (e) => {
 *   if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
 *     useShortcutStore.getState().setShowQuickSearchWindow(true);
 *   }
 * });
 */

/**
 * Example 6: Testing
 *
 * Testing becomes easier - no provider wrappers needed!
 *
 * Before:
 *   render(
 *     <ShortcutProvider>
 *       <MyComponent />
 *     </ShortcutProvider>
 *   );
 *
 * After:
 *   render(<MyComponent />); // Just render the component
 *
 * // Or mock the store:
 *   useShortcutStore.setState({ showCreateTaskWindow: true });
 */
