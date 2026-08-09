/**
 * DataProvider: the seam between the UI and the data layer (epic
 * Zettelgarden-v5b, Phase 2b — issue fv3).
 *
 * The offline-writable scope is cards, tasks, and tags (v1 pin): every
 * read/write path for those collections goes through a DataProvider. In the
 * web SPA the provider is the HTTP apiClient (unchanged thin client); in the
 * desktop app it is the sync engine's local mirror + outbox, so reads are
 * instant/offline and writes queue and reconcile on reconnect. Everything
 * else (entities, files, RSS, search, starring…) stays on apiClient.
 */

import type { Card, NextIdResponse, PartialCard } from '../models/Card';
import type { Task } from '../models/Task';
import type { Tag } from '../models/Tags';
import { isDesktopApp } from './tauriStorageAdapter';
import { httpDataProvider } from './httpProvider';
import type { SyncDataProvider } from './syncProvider';

export interface FetchTasksParams {
  showCompleted?: boolean;
  scheduledDate?: Date | null;
  completedDate?: Date | null;
  status?: string | null;
}

export interface UnsortedCardsResponse {
  cards: PartialCard[];
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface CreateTagParams {
  name: string;
  color: string;
}

export interface DataProvider {
  // ---- cards (offline-writable) ----
  getCard(id: string): Promise<Card>;
  saveNewCard(card: Card): Promise<Card>;
  saveExistingCard(card: Card): Promise<Card>;
  deleteCard(id: number): Promise<Card | null>;
  getCardChildren(cardId: string): Promise<PartialCard[]>;
  getCardTags(cardId: string): Promise<any[]>;
  getCardTasks(cardId: string | number): Promise<any[]>;
  getUnsortedCards(page: number, perPage: number): Promise<UnsortedCardsResponse>;
  getNextRootId(): Promise<NextIdResponse>;

  // ---- tasks (offline-writable) ----
  fetchTasks(params: FetchTasksParams): Promise<Task[]>;
  fetchTask(id: string): Promise<Task>;
  saveNewTask(task: Task): Promise<Task>;
  saveExistingTask(task: Task): Promise<Task>;
  deleteTask(id: number): Promise<Task | null>;

  // ---- tags (offline-writable) ----
  fetchUserTags(): Promise<Tag[]>;
  createTag(params: CreateTagParams): Promise<Tag>;
  deleteTag(id: number): Promise<Tag | null>;
}

let syncProvider: SyncDataProvider | null = null;

/**
 * Registers the desktop sync provider once the engine is initialized
 * (SyncProvider mounts before any data query). Until then — and always in
 * the web app — the HTTP provider serves.
 */
export function registerSyncProvider(provider: SyncDataProvider): void {
  syncProvider = provider;
}

export function getSyncProvider(): SyncDataProvider | null {
  return syncProvider;
}

/**
 * Returns the active provider: the local-first sync provider in the desktop
 * app once it is initialized, otherwise the HTTP provider (unchanged web
 * behavior — the thin client).
 */
export function getDataProvider(): DataProvider {
  if (syncProvider) return syncProvider;
  return httpDataProvider;
}

export { isDesktopApp };
