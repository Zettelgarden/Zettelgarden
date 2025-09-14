import { stringToArray } from "linkifyjs";
import {
  Card,
  CardChunk,
  PartialCard,
  NextIdResponse,
  getRatingValue,
  Entity,
  SearchResult,
  defaultPartialCard,
} from "../models/Card";
import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

interface SearchRequestParams {
  search_term: string;
  sort?: string;
  full_text?: boolean;
  show_entities?: boolean;
  show_facts?: boolean;
  search_type?: string; // classic or typesense
  rerank?: boolean;
}

export function semanticSearchCards(
  searchTerm = "",
  fullText = false,
  showEntities = false,
  showFacts = true,
  sortBy = "sortByRanking",
  searchType = "classic",
  rerank = true,
): Promise<SearchResult[]> {
  let token = localStorage.getItem("token");
  let url = base_url + "/search";

  const params: SearchRequestParams = {
    search_term: searchTerm,
    search_type: searchType,
    full_text: fullText,
    show_entities: showEntities,
    show_facts: showFacts,
    sort: sortBy,
    rerank: rerank,
  };

  return fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify(params),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((results: SearchResult[]) => {
          if (results === null) {
            return [];
          }
          return results.map((result) => ({
            ...result,
            created_at: new Date(result.created_at),
            updated_at: new Date(result.updated_at),
          }));
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function fetchCards(searchTerm = "", fullText = false): Promise<Card[]> {
  return semanticSearchCards(searchTerm, fullText).then((results) => {
    if (results === null) return [];
    return results.map(result => ({
      id: Number(result.metadata?.id) || 0,
      card_id: result.id,
      title: result.title,
      body: result.preview || "",
      link: "",
      is_deleted: false,
      created_at: result.created_at,
      updated_at: result.updated_at,
      parent_id: result.metadata?.parent_id || 0,
      user_id: 0,
      parent: defaultPartialCard,
      files: [],
      children: [],
      references: [],
      tags: [],
      tasks: [],
      entities: [],
    } as Card));
  });
}

export function getCard(id: string): Promise<Card> {
  let encoded = encodeURIComponent(id);
  const url = base_url + `/cards/${encoded}`;

  let token = localStorage.getItem("token");
  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((card: Card) => {
          let children =
            card.children !== null
              ? card.children.map((child) => {
                return {
                  ...child,
                  created_at: new Date(child.created_at),
                  updated_at: new Date(child.updated_at),
                };
              })
              : [];
          let references =
            card.references !== null
              ? card.references.map((ref) => {
                return {
                  ...ref,
                  created_at: new Date(ref.created_at),
                  updated_at: new Date(ref.updated_at),
                };
              })
              : [];
          let tasks =
            card.tasks !== null
              ? card.tasks.map((task) => {
                return {
                  ...task,
                  scheduled_date: task.scheduled_date
                    ? new Date(task.scheduled_date)
                    : null,
                  dueDate: task.dueDate ? new Date(task.dueDate) : null,
                  created_at: new Date(task.created_at),
                  updated_at: new Date(task.updated_at),
                  completed_at: task.completed_at
                    ? new Date(task.completed_at)
                    : null,
                };
              })
              : [];
          return {
            ...card,
            created_at: new Date(card.created_at),
            updated_at: new Date(card.updated_at),
            children: children,
            references: references,
            tasks: tasks,
          };
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function saveNewCard(card: Card): Promise<Card> {
  const url = base_url + `/cards`;
  const method = "POST";
  card.card_id = card.card_id.trim();
  return saveCard(url, method, card);
}

export function saveExistingCard(card: Card): Promise<Card> {
  const url = base_url + `/cards/${encodeURIComponent(card.id)}`;
  const method = "PUT";
  return saveCard(url, method, card);
}
export function saveCard(
  url: string,
  method: string,
  card: Card,
): Promise<Card> {
  let token = localStorage.getItem("token");
  return fetch(url, {
    method: method,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(card),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<Card>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function deleteCard(id: number): Promise<Card | null> {
  let encodedId = encodeURIComponent(id);
  const url = `${base_url}/cards/${encodedId}`;

  let token = localStorage.getItem("token");

  // Send a DELETE request to the URL
  return fetch(url, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        if (response.status === 204) {
          return null;
        }
        return response.json() as Promise<Card>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
export function getCardAuditEvents(cardId: string): Promise<any[]> {
  let token = localStorage.getItem("token");
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/audit`;

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((events: any[]) => {
          if (events === null) {
            return [];
          }
          return events.map(event => ({
            ...event,
            created_at: new Date(event.created_at),
            updated_at: new Date(event.updated_at),
          }));
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getCardFiles(cardId: string): Promise<any[]> {
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/files`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((files: any[]) => {
          if (files === null) {
            return [];
          }
          return files.map((file) => ({
            ...file,
            created_at: new Date(file.created_at),
            updated_at: new Date(file.updated_at),
          }));
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getCardChildren(cardId: string): Promise<PartialCard[]> {
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/children`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((children: PartialCard[]) => {
          if (children === null) {
            return [];
          }
          return children.map((child) => ({
            ...child,
            created_at: new Date(child.created_at),
            updated_at: new Date(child.updated_at),
          }));
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getCardTags(cardId: string): Promise<any[]> {
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/tags`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((tags: any[]) => {
          if (tags === null) {
            return [];
          }
          return tags;
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getLinkedEntitiesByCardPK(cardId: string | number): Promise<Entity[]> {
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/linked-entities`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then(async (response) => {
      if (!response) {
        return Promise.reject(new Error("Response is undefined"));
      }

      // Handle no content responses gracefully
      if (response.status === 204) {
        return [];
      }

      try {
        const entities: Entity[] | null = await response.json();
        if (!entities) {
          return [];
        }
        return entities;
      } catch (err) {
        // In case of empty body or invalid JSON
        return [];
      }
    });
}

export function getCardEntities(cardId: string | number): Promise<any[]> {
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/entities`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((entities: any[]) => {
          if (entities === null) {
            return [];
          }
          return entities;
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getCardTasks(cardId: string | number): Promise<any[]> {
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/tasks`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((tasks: any[]) => {
          if (tasks === null) {
            return [];
          }
          return tasks.map((task) => ({
            ...task,
            scheduled_date: task.scheduled_date ? new Date(task.scheduled_date) : null,
            dueDate: task.dueDate ? new Date(task.dueDate) : null,
            created_at: new Date(task.created_at),
            updated_at: new Date(task.updated_at),
            completed_at: task.completed_at ? new Date(task.completed_at) : null,
          }));
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getCardReferences(cardId: string): Promise<PartialCard[]> {
  const url = `${base_url}/cards/${encodeURIComponent(cardId)}/references`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((refs: PartialCard[]) => {
          if (refs === null) {
            return [];
          }
          return refs.map((ref) => ({
            ...ref,
            created_at: new Date(ref.created_at),
            updated_at: new Date(ref.updated_at),
          }));
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export async function getNextRootId(): Promise<NextIdResponse> {
  const url = `${base_url}/cards/next-root-id`;
  let token = localStorage.getItem("token");

  return await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<NextIdResponse>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

/**
 * Star a card to make it easily accessible
 * @param cardId The ID of the card to star
 * @returns A promise that resolves when the card is starred
 */
export function starCard(cardId: number): Promise<void> {
  const url = `${base_url}/cards/${cardId}/star`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then(() => {
      return;
    });
}

/**
 * Unstar a previously starred card
 * @param cardId The ID of the card to unstar
 * @returns A promise that resolves when the card is unstarred
 */
export function unstarCard(cardId: number): Promise<void> {
  const url = `${base_url}/cards/${cardId}/star`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then(() => {
      return;
    });
}

/**
 * Get all cards that have been starred by the current user
 * @returns A promise that resolves to an array of starred cards with their full data
 */
export function getStarredCards(): Promise<Card[]> {
  const url = `${base_url}/cards/starred`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((starredCards: any[]) => {
          if (starredCards === null) {
            return [];
          }

          // Transform the response into Card objects
          return starredCards.map((starredCard) => {
            const card = starredCard.card;

            // Process dates and nested objects
            return {
              ...card,
              created_at: new Date(card.created_at),
              updated_at: new Date(card.updated_at),
              children: card.children ? card.children.map((child: any) => ({
                ...child,
                created_at: new Date(child.created_at),
                updated_at: new Date(child.updated_at),
              })) : [],
              references: card.references ? card.references.map((ref: any) => ({
                ...ref,
                created_at: new Date(ref.created_at),
                updated_at: new Date(ref.updated_at),
              })) : [],
              tasks: card.tasks ? card.tasks.map((task: any) => ({
                ...task,
                scheduled_date: task.scheduled_date ? new Date(task.scheduled_date) : null,
                dueDate: task.dueDate ? new Date(task.dueDate) : null,
                created_at: new Date(task.created_at),
                updated_at: new Date(task.updated_at),
                completed_at: task.completed_at ? new Date(task.completed_at) : null,
              })) : [],
              is_pinned: true, // Mark as starred since it's coming from the starred cards endpoint
            };
          });
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
