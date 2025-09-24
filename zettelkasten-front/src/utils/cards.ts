import { PartialCard, Card, CardChunk, SearchResult } from "../models/Card";

// filter the card_id and title by searchText, return top 5 matches
export function quickFilterCards(
  cards: PartialCard[],
  searchText: string,
): PartialCard[] {
  let searchTextLower = searchText.toLowerCase().trim();
  const exactMatch = cards.find(
    (card) => card.card_id.toLowerCase() === searchText.toLowerCase(),
  );
  const searchResults = cards.filter((card) => {
    return (
      card.card_id.toLowerCase().startsWith(searchTextLower) ||
      card.title.toLowerCase().includes(searchTextLower)
    );
  });

  let filteredCards = exactMatch
    ? [exactMatch, ...searchResults]
    : searchResults;

  filteredCards = filteredCards.filter(
    (card, index, self) =>
      index === self.findIndex((t) => t.card_id === card.card_id),
  );
  let results = filteredCards.slice(0, 5);
  return results;
}

export function convertCardToPartialCard(card: Card): PartialCard {
  return {
    id: card.id,
    card_id: card.card_id,
    user_id: card.user_id,
    title: card.title,
    parent_id: card.parent_id,
    created_at: card.created_at,
    updated_at: card.updated_at,
    tags: card.tags,
  };
}

export function isCardIdUnique(
  cards: Card[] | PartialCard[],
  id: string,
): boolean {
  return !cards.some((card) => card.card_id === id);
}

export function sortCards(cards: SearchResult[], sortMethod: string): SearchResult[] {
  const sortedCards = [...cards];
  switch (sortMethod) {
    case "sortCreatedNewOld":
      sortedCards.sort((a, b) => b.created_at.getTime() - a.created_at.getTime());
      break;
    case "sortCreatedOldNew":
      sortedCards.sort((a, b) => a.created_at.getTime() - b.created_at.getTime());
      break;
    case "sortNewOld":
      sortedCards.sort((a, b) => b.updated_at.getTime() - a.updated_at.getTime());
      break;
    case "sortOldNew":
      sortedCards.sort((a, b) => a.updated_at.getTime() - b.updated_at.getTime());
      break;
    case "sortBigSmall":
      sortedCards.sort((a, b) => a.title.localeCompare(b.title));
      break;
    case "sortSmallBig":
      sortedCards.sort((a, b) => b.title.localeCompare(a.title));
      break;
    case "sortByRanking":
      sortedCards.sort((a, b) => b.score - a.score);
      break;
    default:
      break;
  }
  return sortedCards;
}

export function compareCardIds(a: string, b: string): number {
  const parseCardId = (cardId: string): (string | number)[] => {
    // Split ID on separators '/' and '.' and parse numbers
    return cardId.split("/").flatMap((part) =>
      part.split(".").map((segment, index) => {
        return isNaN(Number(segment)) ? segment : Number(segment);
      }),
    );
  };

  const aParts = parseCardId(a);
  const bParts = parseCardId(b);

  const length = Math.min(aParts.length, bParts.length);

  for (let i = 0; i < length; i++) {
    const aPart = aParts[i];
    const bPart = bParts[i];

    // Compare numbers and strings accordingly
    if (typeof aPart === "number" && typeof bPart === "number") {
      if (aPart !== bPart) {
        return aPart - bPart;
      }
    } else if (typeof aPart === "string" && typeof bPart === "string") {
      if (aPart !== bPart) {
        return aPart.localeCompare(bPart);
      }
    } else {
      // Unequal types should never occur if input format is valid, but handle gracefully
      return typeof aPart === "number" ? -1 : 1;
    }
  }

  // If all parts match, the shorter ID comes first
  return aParts.length - bParts.length;
}

export function sortCardIds(input: string[]): string[] {
  let results = input.sort(compareCardIds);
  console.log(results);
  return results;
}

export function findNextChildId(parentId: string, existingChildren: PartialCard[]): string {
  if (existingChildren.length === 0) {
    return `${parentId}.1`;
  }

  const childIds = existingChildren
    .map(child => child.card_id)
    .filter(childId => {
      const childPrefix = childId.substring(0, parentId.length);
      return childPrefix === parentId && childId.length > parentId.length;
    });

  let maxChildNumber = 0;

  for (const childId of childIds) {
    const childSuffix = childId.substring(parentId.length + 1);
    const firstSegment = childSuffix.split(/[/.\-]/)[0];
    const num = parseInt(firstSegment, 10);
    if (!isNaN(num) && num > maxChildNumber) {
      maxChildNumber = num;
    }
  }

  return `${parentId}.${maxChildNumber + 1}`;
}
