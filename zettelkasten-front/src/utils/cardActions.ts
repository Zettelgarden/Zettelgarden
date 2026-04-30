import { Card, PartialCard } from "../models/Card";
import { saveExistingCard, starCard, unstarCard } from "../api/cards";
import { findNextChildId } from "./cards";

/**
 * Adds a tag to the card body by appending it with a newline prefix
 * @param card - The card to modify
 * @param tagName - The tag name to add (without #)
 * @returns The modified card with the tag added to the body
 */
export function addTagToBody(card: Card, tagName: string): Card {
  return {
    ...card,
    body: card.body + "\n\n#" + tagName,
  };
}

/**
 * Removes all occurrences of a tag from the card body
 * @param card - The card to modify
 * @param tagName - The tag name to remove (without #)
 * @returns The modified card with the tag removed from the body
 */
export function removeTagFromBody(card: Card, tagName: string): Card {
  const tagRegex = new RegExp(`\\n*#${tagName}\\b`, 'g');
  return {
    ...card,
    body: card.body.replace(tagRegex, ''),
  };
}

/**
 * Adds a backlink reference to the card body
 * @param card - The card to modify
 * @param selectedCard - The card to reference
 * @returns The modified card with the backlink added to the body
 */
export function addBacklinkToBody(card: Card, selectedCard: PartialCard): Card {
  const backlinkText = "\n\n[[" + selectedCard.card_id + "|" + selectedCard.title + "]]";
  return {
    ...card,
    body: card.body + backlinkText,
  };
}

/**
 * Toggles the star status of a card by calling the appropriate API
 * @param card - The card to star/unstar
 * @param refreshCallback - Callback function to refresh card data after API call
 */
export async function toggleCardStar(
  card: Card,
  refreshCallback?: () => void
): Promise<void> {
  try {
    if (card.is_starred) {
      await unstarCard(card.id);
    } else {
      await starCard(card.id);
    }

    // Refresh the card data after the mutation if callback provided
    if (refreshCallback) {
      refreshCallback();
    }
  } catch (error) {
    console.error("Error toggling star status:", error);
    throw error; // Re-throw to allow caller to handle
  }
}

/**
 * Toggles the star status of a partial card by calling the appropriate API
 * @param card - The partial card to star/unstar
 * @param refreshCallback - Callback function to refresh card data after API call
 */
export async function togglePartialCardStar(
  card: PartialCard,
  refreshCallback?: () => void
): Promise<void> {
  try {
    if (card.is_starred) {
      await unstarCard(card.id);
    } else {
      await starCard(card.id);
    }

    // Refresh the card data after the mutation if callback provided
    if (refreshCallback) {
      refreshCallback();
    }
  } catch (error) {
    console.error("Error toggling star status:", error);
    throw error; // Re-throw to allow caller to handle
  }
}

/**
 * Saves a modified card to the API
 * @param card - The card to save
 * @param refreshCallback - Optional callback to refresh data after save
 */
export async function saveCard(
  card: Card,
  refreshCallback?: () => void
): Promise<void> {
  await saveExistingCard(card);

  // Refresh the card data after the mutation if callback provided
  if (refreshCallback) {
    refreshCallback();
  }
}

/**
 * Triggers card reprocessing by setting process_entities_and_facts flag and saving
 * @param card - The card to resummarize
 * @param refreshCallback - Optional callback to refresh data after save
 */
export async function resummarizeCard(
  card: Card,
  refreshCallback?: () => void
): Promise<void> {
  const updatedCard = {
    ...card,
    process_entities_and_facts: true
  };

  await saveCard(updatedCard, refreshCallback);
}

/**
 * Adds a tag to the card and saves it
 * @param card - The card to modify
 * @param tagName - The tag name to add (without #)
 * @param refreshCallback - Optional callback to refresh data after save
 */
export async function addTagToCard(
  card: Card,
  tagName: string,
  refreshCallback?: () => void
): Promise<void> {
  const editedCard = addTagToBody(card, tagName);
  await saveCard(editedCard, refreshCallback);
}

/**
 * Removes a tag from the card and saves it
 * @param card - The card to modify
 * @param tagName - The tag name to remove (without #)
 * @param refreshCallback - Optional callback to refresh data after save
 */
export async function removeTagFromCard(
  card: Card,
  tagName: string,
  refreshCallback?: () => void
): Promise<void> {
  const editedCard = removeTagFromBody(card, tagName);
  await saveCard(editedCard, refreshCallback);
}

/**
 * Adds a backlink to the card and saves it
 * @param card - The card to modify
 * @param selectedCard - The card to reference
 * @param refreshCallback - Optional callback to refresh data after save
 */
export async function addBacklinkToCard(
  card: Card,
  selectedCard: PartialCard,
  refreshCallback?: () => void
): Promise<void> {
  const editedCard = addBacklinkToBody(card, selectedCard);
  await saveCard(editedCard, refreshCallback);
}

/**
 * Calculates the next available child ID for creating a new child card
 * @param parentCardId - The ID of the parent card
 * @param existingChildren - Array of existing child cards
 * @returns The next available child ID
 */
export function calculateNextChildId(
  parentCardId: string,
  existingChildren: PartialCard[]
): string {
  return findNextChildId(parentCardId, existingChildren);
}