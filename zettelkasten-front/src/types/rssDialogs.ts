import { RSSFeed, RSSFolder, RSSArticle } from "../api/rss";

/**
 * Unified dialog state using type discriminator pattern
 */
export type DialogState =
  | { type: 'none' }
  | { type: 'addFeed' }
  | { type: 'editFeed'; feed: RSSFeed }
  | { type: 'editFolder'; folder: RSSFolder }
  | { type: 'createFolder' }
  | { type: 'deleteConfirm'; itemType: 'feed' | 'folder'; item: RSSFeed | RSSFolder }
  | { type: 'convert'; article: RSSArticle }
  | { type: 'import' };

/**
 * Initial dialog state (no dialog open)
 */
export const initialDialogState: DialogState = { type: 'none' };

/**
 * Helper functions to create specific dialog states
 */
export const DialogStates = {
  none: (): DialogState => ({ type: 'none' }),
  addFeed: (): DialogState => ({ type: 'addFeed' }),
  editFeed: (feed: RSSFeed): DialogState => ({ type: 'editFeed', feed }),
  editFolder: (folder: RSSFolder): DialogState => ({ type: 'editFolder', folder }),
  createFolder: (): DialogState => ({ type: 'createFolder' }),
  deleteConfirm: (itemType: 'feed' | 'folder', item: RSSFeed | RSSFolder): DialogState => ({
    type: 'deleteConfirm',
    itemType,
    item,
  }),
  convert: (article: RSSArticle): DialogState => ({ type: 'convert', article }),
  import: (): DialogState => ({ type: 'import' }),
} as const;

/**
 * Type guard to check if dialog is open
 */
export function isDialogOpen(state: DialogState): boolean {
  return state.type !== 'none';
}

/**
 * Get dialog title based on state
 */
export function getDialogTitle(state: DialogState): string {
  switch (state.type) {
    case 'addFeed':
      return 'Add Feed';
    case 'editFeed':
      return 'Edit Feed';
    case 'editFolder':
      return 'Edit Folder';
    case 'createFolder':
      return 'Create Folder';
    case 'deleteConfirm':
      return state.itemType === 'feed' ? 'Delete Feed' : 'Delete Folder';
    case 'convert':
      return 'Convert to Card';
    case 'import':
      return 'Import OPML';
    case 'none':
      return '';
  }
}
