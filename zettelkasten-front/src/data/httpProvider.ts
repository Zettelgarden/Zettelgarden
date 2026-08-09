/**
 * HTTP DataProvider: the web thin-client implementation (unchanged behavior
 * from the original api/cards.ts / api/tasks.ts / api/tags.ts functions —
 * their bodies moved here so api/* can route through the provider seam
 * without a module cycle). In the desktop app the SyncDataProvider replaces
 * this; the web app always uses this.
 */

import { apiClient, getData } from '../api/client';
import { Card, PartialCard, NextIdResponse } from '../models/Card';
import { Task } from '../models/Task';
import { Tag } from '../models/Tags';
import { processTaskFromAPI } from '../utils/taskDataProcessing';
import type {
  DataProvider,
  FetchTasksParams,
  UnsortedCardsResponse,
  CreateTagParams,
} from './provider';

// ---- shared card helpers (also used by the online-only api/cards.ts) ----

export function processCardFromAPI(card: Card): Card {
  return {
    ...card,
    created_at: new Date(card.created_at),
    updated_at: new Date(card.updated_at),
    children:
      card.children?.map((child) => ({
        ...child,
        created_at: new Date(child.created_at),
        updated_at: new Date(child.updated_at),
      })) || [],
    references:
      card.references?.map((ref) => ({
        ...ref,
        created_at: new Date(ref.created_at),
        updated_at: new Date(ref.updated_at),
      })) || [],
    tasks:
      card.tasks?.map((task) => ({
        ...task,
        scheduled_date: task.scheduled_date
          ? new Date(task.scheduled_date)
          : null,
        due_date: task.due_date ? new Date(task.due_date) : null,
        created_at: new Date(task.created_at),
        updated_at: new Date(task.updated_at),
        completed_at: task.completed_at ? new Date(task.completed_at) : null,
      })) || [],
  };
}

function processPartialCards(cards: PartialCard[]): PartialCard[] {
  return cards.map((ref) => ({
    ...ref,
    created_at: new Date(ref.created_at),
    updated_at: new Date(ref.updated_at),
  }));
}

// ---- cards ----

async function getCard(id: string): Promise<Card> {
  const encoded = encodeURIComponent(id);
  const { data: card } = await apiClient.get<Card>(`/cards/${encoded}`);
  return processCardFromAPI(card);
}

async function saveNewCard(card: Card): Promise<Card> {
  card.card_id = card.card_id.trim();
  const { data } = await apiClient.post<Card>('/cards', card);
  return processCardFromAPI(data);
}

async function saveExistingCard(card: Card): Promise<Card> {
  const { data } = await apiClient.put<Card>(
    `/cards/${encodeURIComponent(card.id)}`,
    card,
  );
  return processCardFromAPI(data);
}

async function deleteCard(id: number): Promise<Card | null> {
  const encodedId = encodeURIComponent(id);
  const { response, data } = await apiClient.delete<Card>(
    `/cards/${encodedId}`,
  );
  if (response.status === 204) {
    return null;
  }
  return data;
}

async function getCardChildren(cardId: string): Promise<PartialCard[]> {
  const { data: children } = await apiClient.get<PartialCard[]>(
    `/cards/${encodeURIComponent(cardId)}/children`,
  );
  if (!children) {
    return [];
  }
  return children.map((child) => ({
    ...child,
    created_at: new Date(child.created_at),
    updated_at: new Date(child.updated_at),
  }));
}

async function getCardTags(cardId: string): Promise<any[]> {
  const { data: tags } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/tags`,
  );
  return tags ?? [];
}

async function getCardTasks(cardId: string | number): Promise<any[]> {
  const { data: tasks } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/tasks`,
  );
  if (!tasks) {
    return [];
  }
  return tasks.map((task) => ({
    ...task,
    scheduled_date: task.scheduled_date ? new Date(task.scheduled_date) : null,
    due_date: task.due_date ? new Date(task.due_date) : null,
    created_at: new Date(task.created_at),
    updated_at: new Date(task.updated_at),
    completed_at: task.completed_at ? new Date(task.completed_at) : null,
  }));
}

async function getUnsortedCards(
  page = 1,
  perPage = 10,
): Promise<UnsortedCardsResponse> {
  const { data } = await apiClient.get<UnsortedCardsResponse>(
    `/cards/unsorted?page=${page}&per_page=${perPage}`,
  );
  const cards = data.cards.map((card) => ({
    ...card,
    created_at: new Date(card.created_at),
    updated_at: new Date(card.updated_at),
  }));
  return { ...data, cards };
}

async function getNextRootId(): Promise<NextIdResponse> {
  return getData(apiClient.get<NextIdResponse>('/cards/next-root-id'));
}

// ---- tasks ----

async function fetchTasks(params: FetchTasksParams = {}): Promise<Task[]> {
  const {
    showCompleted = false,
    scheduledDate = null,
    completedDate = null,
    status = null,
  } = params;

  const fetchAllTasks = async (
    offset = 0,
    allTasks: Task[] = [],
  ): Promise<Task[]> => {
    const requestParams: Record<string, string | number | boolean | undefined> =
      {
        limit: 100,
        offset,
        completed: showCompleted,
      };
    if (scheduledDate) {
      requestParams.scheduled_date = scheduledDate.toISOString().split('T')[0];
    }
    if (completedDate) {
      requestParams.completed_date = completedDate.toISOString().split('T')[0];
    }
    if (status) {
      requestParams.status = status;
    }
    const { data: tasksResponse } = await apiClient.get<{
      tasks: Task[];
      limit: number;
    }>('/tasks', { params: requestParams });
    if (!tasksResponse.tasks) {
      return allTasks;
    }
    const formattedTasks = tasksResponse.tasks.map((task) =>
      processTaskFromAPI(task),
    );
    const combinedTasks = [...allTasks, ...formattedTasks];
    if (tasksResponse.tasks.length < tasksResponse.limit) {
      return combinedTasks;
    }
    return fetchAllTasks(offset + tasksResponse.limit, combinedTasks);
  };
  return fetchAllTasks();
}

async function fetchTask(id: string): Promise<Task> {
  const encoded = encodeURIComponent(id);
  const { data: rawTask } = await apiClient.get<Task>(`/tasks/${encoded}`);
  return processTaskFromAPI(rawTask);
}

async function saveNewTask(task: Task): Promise<Task> {
  const { data } = await apiClient.post<Task>('/tasks', task);
  return data;
}

async function saveExistingTask(task: Task): Promise<Task> {
  const { data } = await apiClient.put<Task>(
    `/tasks/${encodeURIComponent(task.id)}`,
    task,
  );
  return data;
}

async function deleteTask(id: number): Promise<Task | null> {
  const encodedId = encodeURIComponent(id);
  const { response, data } = await apiClient.delete<Task>(
    `/tasks/${encodedId}`,
  );
  if (response.status === 204) {
    return null;
  }
  return data;
}

// ---- tags ----

async function fetchUserTags(): Promise<Tag[]> {
  return getData(apiClient.get<Tag[]>('/tags'));
}

async function deleteTag(id: number): Promise<Tag | null> {
  return getData(apiClient.delete<Tag>(`/tags/id/${id}`));
}

async function createTag(params: CreateTagParams): Promise<Tag> {
  return getData(apiClient.post<Tag>('/tags', params));
}

export const httpDataProvider: DataProvider = {
  getCard,
  saveNewCard,
  saveExistingCard,
  deleteCard,
  getCardChildren,
  getCardTags,
  getCardTasks,
  getUnsortedCards,
  getNextRootId,
  fetchTasks,
  fetchTask,
  saveNewTask,
  saveExistingTask,
  deleteTask,
  fetchUserTags,
  createTag,
  deleteTag,
};
