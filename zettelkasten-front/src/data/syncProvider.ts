/**
 * SyncDataProvider: the desktop app's local-first data layer (epic
 * Zettelgarden-v5b, Phase 2b — issue fv3). Every card/task/tag read comes
 * from the engine's local mirror (instant, offline); every write goes to the
 * mirror + outbox atomically and reconciles on reconnect. The server is only
 * ever reached by the engine's HttpTransport.
 *
 * Identity: the UI keys rows by server int id, but offline-created rows have
 * none yet. This provider assigns stable NEGATIVE temp ids and persists an
 * alias map (tempId -> sync_uuid) in sync_meta, so URLs like /app/card/-5
 * keep resolving even after the row syncs and receives its real id.
 *
 * FK translation: task card_pk / parent_task_id are server ints; offline we
 * push card_pk_uuid / parent_task_uuid (resolved in-batch server-side) and
 * omit the ints, matching the Phase 0b push contract.
 */

import { SyncEngine } from '@zettelgarden/sync-engine/engine';
import type { Collection, MirrorRow } from '@zettelgarden/sync-engine/types';
import { Card, NextIdResponse, PartialCard } from '../models/Card';
import { Task, emptyTask } from '../models/Task';
import { Tag } from '../models/Tags';
import type {
  CreateTagParams,
  DataProvider,
  FetchTasksParams,
  UnsortedCardsResponse,
} from './provider';

const CARD_SEQ_KEY = 'local_card_id_seq';
const TASK_SEQ_KEY = 'local_task_id_seq';
const CARD_ALIAS_KEY = 'local_card_aliases';
const TASK_ALIAS_KEY = 'local_task_aliases';

type AliasMap = Record<string, string>; // tempId (string) -> rowUuid

/** SQLite/ISO date parsing: "2026-08-09 01:22:49" (no TZ) is treated as UTC. */
export function parseSyncDate(v: unknown): Date | null {
  if (v == null) return null;
  if (v instanceof Date) return isNaN(v.getTime()) ? null : v;
  const s = String(v);
  if (!s) return null;
  const m = s.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2}:\d{2})/);
  if (m) {
    const d = new Date(`${m[1]}T${m[2]}Z`);
    return isNaN(d.getTime()) ? null : d;
  }
  const d = new Date(s);
  return isNaN(d.getTime()) ? null : d;
}

/** Serializes a UI date (or raw string) into an ISO string for push. */
function toIso(v: Date | string | null | undefined): string | null {
  if (v == null) return null;
  if (v instanceof Date) return isNaN(v.getTime()) ? null : v.toISOString();
  const s = String(v);
  return s || null;
}

const NUMERIC_ID_RE = /^\d+$/;

export class SyncDataProvider implements DataProvider {
  private engine: SyncEngine;
  private cachedAliases: Map<string, AliasMap> = new Map();

  constructor(engine: SyncEngine) {
    this.engine = engine;
  }

  // ---- local id / uuid bookkeeping ---------------------------------------

  private async aliases(collection: 'cards' | 'tasks'): Promise<AliasMap> {
    const key = collection === 'cards' ? CARD_ALIAS_KEY : TASK_ALIAS_KEY;
    const cached = this.cachedAliases.get(key);
    if (cached) return cached;
    const raw = await this.engine.getMeta(key);
    let map: AliasMap = {};
    try {
      map = raw ? (JSON.parse(raw) as AliasMap) : {};
    } catch {
      map = {};
    }
    this.cachedAliases.set(key, map);
    return map;
  }

  private async saveAliases(
    collection: 'cards' | 'tasks',
    map: AliasMap,
  ): Promise<void> {
    const key = collection === 'cards' ? CARD_ALIAS_KEY : TASK_ALIAS_KEY;
    this.cachedAliases.set(key, map);
    await this.engine.setMeta(key, JSON.stringify(map));
  }

  /** Next temp id for a new local row (negative, stable across restarts). */
  private async nextLocalId(collection: 'cards' | 'tasks'): Promise<number> {
    const seqKey = collection === 'cards' ? CARD_SEQ_KEY : TASK_SEQ_KEY;
    const raw = await this.engine.getMeta(seqKey);
    const seq = raw ? parseInt(raw, 10) : 0;
    const id = -(seq + 1);
    await this.engine.setMeta(seqKey, String(seq + 1));
    return id;
  }

  /** Resolves a rowUuid for a UI card id (int temp/real, card_id, or uuid). */
  private async resolveCardUuid(
    id: string | number,
  ): Promise<string | undefined> {
    const s = String(id);
    // A uuid passed straight through.
    if (!NUMERIC_ID_RE.test(s) && s.length > 20) {
      const row = await this.engine.getRow('cards', s);
      if (row) return s;
    }
    if (/^-\d+$/.test(s)) {
      const map = await this.aliases('cards');
      return map[s];
    }
    if (NUMERIC_ID_RE.test(s)) {
      const rows = await this.engine.query('cards');
      const hit = rows.find((r) => Number(r.data.id) === Number(s));
      return hit?.rowUuid;
    }
    // card_id string
    const rows = await this.engine.query('cards');
    return rows.find((r) => r.data.card_id === s)?.rowUuid;
  }

  /** Resolves a rowUuid for a UI task id (int temp/real). */
  private async resolveTaskUuid(id: number): Promise<string | undefined> {
    const s = String(id);
    if (/^-\d+$/.test(s)) {
      const map = await this.aliases('tasks');
      return map[s];
    }
    const rows = await this.engine.query('tasks');
    return rows.find((r) => Number(r.data.id) === Number(s))?.rowUuid;
  }

  /** Canonical UI id for a row: real server id once synced, else temp id. */
  private async canonicalId(
    collection: 'cards' | 'tasks',
    rowUuid: string,
  ): Promise<number> {
    const row = await this.engine.getRow(collection, rowUuid);
    if (!row) return 0;
    const serverId = row.data.id;
    if (serverId != null && Number(serverId) > 0) return Number(serverId);
    const map = await this.aliases(collection);
    for (const [temp, uuid] of Object.entries(map)) {
      if (uuid === rowUuid) return Number(temp);
    }
    return 0;
  }

  // ---- card shaping ------------------------------------------------------

  private async cardFromRow(row: MirrorRow): Promise<Card> {
    const d = row.data;
    const id = await this.canonicalId('cards', row.rowUuid);
    const tags = await this.tagsForBody(
      d.body as string,
      d.parent_id as number,
    );
    const card: Card = {
      id,
      card_id: (d.card_id as string) ?? '',
      user_id: Number(d.user_id ?? 0),
      title: (d.title as string) ?? '',
      body: (d.body as string) ?? '',
      link: (d.link as string) ?? '',
      is_deleted: !!d.is_deleted,
      created_at: parseSyncDate(d.created_at) ?? new Date(0),
      updated_at: parseSyncDate(d.updated_at) ?? new Date(0),
      parent_id: Number(d.parent_id ?? 0),
      parent:
        (await this.parentFor(d.parent_id as number)) ??
        ({
          id: 0,
          card_id: '',
          user_id: 0,
          title: '',
          parent_id: 0,
          created_at: new Date(0),
          updated_at: new Date(0),
          tags: [],
        } as PartialCard),
      files: [],
      children: await this.childrenFor(id, row.rowUuid),
      references: [],
      tags,
      tasks: [],
      entities: [],
      schema_id: d.card_schema_id != null ? Number(d.card_schema_id) : null,
      structured_data: (d.structured_data as Record<string, any>) ?? null,
    };
    return card;
  }

  private async parentFor(parentId: number): Promise<PartialCard | undefined> {
    if (!parentId) return undefined;
    const rows = await this.engine.query('cards');
    const row = rows.find((r) => Number(r.data.id) === parentId);
    if (!row) return undefined;
    return this.partialFromRow(row);
  }

  private async partialFromRow(row: MirrorRow): Promise<PartialCard> {
    const d = row.data;
    return {
      id: await this.canonicalId('cards', row.rowUuid),
      card_id: (d.card_id as string) ?? '',
      user_id: Number(d.user_id ?? 0),
      title: (d.title as string) ?? '',
      parent_id: Number(d.parent_id ?? 0),
      created_at: parseSyncDate(d.created_at) ?? new Date(0),
      updated_at: parseSyncDate(d.updated_at) ?? new Date(0),
      tags: await this.tagsForBody(d.body as string, d.parent_id as number),
    };
  }

  /** Mirror cards whose parent_id points at this card (or temp-id children
   * that reference an offline parent by card_id prefix — not yet linked). */
  private async childrenFor(
    id: number,
    rowUuid: string,
  ): Promise<PartialCard[]> {
    const rows = await this.engine.query('cards');
    const kids = rows.filter(
      (r) => r.rowUuid !== rowUuid && Number(r.data.parent_id) === id,
    );
    const out: PartialCard[] = [];
    for (const row of kids) out.push(await this.partialFromRow(row));
    return out;
  }

  /** Offline tag list for a card: #tags from its body + parent chain,
   * resolved against the mirror tags collection for id/color. */
  private async tagsForBody(
    body: string | undefined,
    parentId: number,
  ): Promise<Tag[]> {
    const names = new Set<string>();
    const collect = async (
      text: string | undefined,
      depth: number,
    ): Promise<void> => {
      if (!text || depth > 10) return;
      for (const m of text.matchAll(/(^|\s)#([a-zA-Z0-9_-]+)/g)) {
        names.add(m[2]);
      }
    };
    await collect(body, 0);
    // Parent chain tags (server: IdentifyParentTags walks parents).
    let pid = parentId;
    let depth = 0;
    while (pid && depth < 10) {
      const rows = await this.engine.query('cards');
      const parent = rows.find((r) => Number(r.data.id) === pid);
      if (!parent) break;
      await collect(parent.data.body as string, depth + 1);
      pid = Number(parent.data.parent_id ?? 0);
      depth++;
    }
    if (names.size === 0) return [];
    const tagRows = await this.engine.query('tags');
    const byName = new Map<string, MirrorRow>();
    for (const t of tagRows) {
      const name = t.data.name as string;
      if (name && !t.data.is_deleted && !byName.has(name)) byName.set(name, t);
    }
    return [...names].map((name) => {
      const t = byName.get(name);
      return {
        id: t ? Number(t.data.id ?? 0) : 0,
        name,
        color: (t?.data.color as string) ?? 'black',
        user_id: Number(t?.data.user_id ?? 0),
      };
    });
  }

  private async taskFromRow(row: MirrorRow): Promise<Task> {
    const d = row.data;
    return {
      ...emptyTask,
      id: await this.canonicalId('tasks', row.rowUuid),
      card_pk: Number(d.card_pk ?? 0),
      user_id: Number(d.user_id ?? 0),
      scheduled_date: parseSyncDate(d.scheduled_date),
      due_date: parseSyncDate(d.due_date),
      created_at: parseSyncDate(d.created_at) ?? new Date(0),
      updated_at: parseSyncDate(d.updated_at) ?? new Date(0),
      completed_at: parseSyncDate(d.completed_at),
      title: (d.title as string) ?? '',
      description: (d.description as string) ?? null,
      priority: (d.priority as string) ?? null,
      status: (d.status as string) ?? 'todo',
      is_complete: !!d.is_complete,
      is_deleted: !!d.is_deleted,
      reminder_time: parseSyncDate(d.reminder_time),
      reminder_sent: !!d.reminder_sent,
      parent_task_id:
        d.parent_task_id != null ? Number(d.parent_task_id) : null,
      sort_order: d.sort_order != null ? Number(d.sort_order) : null,
    };
  }

  private tagFromRow(row: MirrorRow): Tag {
    const d = row.data;
    return {
      id: Number(d.id ?? 0),
      name: (d.name as string) ?? '',
      color: (d.color as string) ?? 'black',
      user_id: Number(d.user_id ?? 0),
    };
  }

  // ---- cards -------------------------------------------------------------

  async getCard(id: string): Promise<Card> {
    const uuid = await this.resolveCardUuid(id);
    if (!uuid) throw new Error('card not found');
    const row = await this.engine.getRow('cards', uuid);
    if (!row) throw new Error('card not found');
    if (row.data.is_deleted) throw new Error('card not found');
    const card = await this.cardFromRow(row);
    // Enrich tasks from the mirror.
    const taskRows = (await this.engine.query('tasks')).filter(
      (r) => Number(r.data.card_pk) === card.id,
    );
    card.tasks = await Promise.all(taskRows.map((r) => this.taskFromRow(r)));
    return card;
  }

  async saveNewCard(card: Card): Promise<Card> {
    const rowUuid = crypto.randomUUID();
    const tempId = await this.nextLocalId('cards');
    const data: Record<string, unknown> = this.cardPushData(card);
    await this.engine.mutate('cards', { rowUuid, data });
    const aliases = await this.aliases('cards');
    aliases[String(tempId)] = rowUuid;
    await this.saveAliases('cards', aliases);
    return { ...card, id: tempId };
  }

  async saveExistingCard(card: Card): Promise<Card> {
    const uuid = await this.resolveCardUuid(card.id);
    if (!uuid) throw new Error('card not found');
    const existing = await this.engine.getRow('cards', uuid);
    const pushed = this.cardPushData(card);
    // Merge over the mirror row so server-managed fields (created_at,
    // parent_id, version, sync_uuid) survive the local write.
    const data = { ...(existing?.data ?? {}), ...pushed, sync_uuid: uuid };
    delete (data as Record<string, unknown>).id;
    await this.engine.mutate('cards', { rowUuid: uuid, data });
    return card;
  }

  async deleteCard(id: number): Promise<Card | null> {
    const uuid = await this.resolveCardUuid(id);
    if (!uuid) return null;
    const existing = await this.engine.getRow('cards', uuid);
    await this.engine.deleteLocal('cards', uuid);
    // Keep the alias so stale URLs stay resolvable after the row is gone
    // (the mirror row is dropped; getCard will throw 'card not found').
    return existing ? this.cardFromRow(existing) : null;
  }

  async getCardChildren(cardId: string): Promise<PartialCard[]> {
    const uuid = await this.resolveCardUuid(cardId);
    if (!uuid) return [];
    const row = await this.engine.getRow('cards', uuid);
    if (!row) return [];
    const id = await this.canonicalId('cards', uuid);
    return this.childrenFor(id, uuid);
  }

  async getCardTags(cardId: string): Promise<any[]> {
    const uuid = await this.resolveCardUuid(cardId);
    if (!uuid) return [];
    const row = await this.engine.getRow('cards', uuid);
    if (!row) return [];
    return this.tagsForBody(
      row.data.body as string,
      Number(row.data.parent_id ?? 0),
    );
  }

  async getCardTasks(cardId: string | number): Promise<any[]> {
    const uuid = await this.resolveCardUuid(cardId);
    if (!uuid) return [];
    const id = await this.canonicalId('cards', uuid);
    const rows = (await this.engine.query('tasks')).filter(
      (r) => Number(r.data.card_pk) === id,
    );
    return Promise.all(rows.map((r) => this.taskFromRow(r)));
  }

  async getUnsortedCards(
    page = 1,
    perPage = 10,
  ): Promise<UnsortedCardsResponse> {
    const rows = (await this.engine.query('cards')).filter(
      (r) => !r.data.card_id && !r.data.is_deleted,
    );
    const cards: PartialCard[] = [];
    for (const row of rows) cards.push(await this.partialFromRow(row));
    cards.sort((a, b) => b.updated_at.getTime() - a.updated_at.getTime());
    const total = cards.length;
    const start = (page - 1) * perPage;
    return {
      cards: cards.slice(start, start + perPage),
      page,
      per_page: perPage,
      total,
      total_pages: Math.ceil(total / perPage),
    };
  }

  async getNextRootId(): Promise<NextIdResponse> {
    const rows = await this.engine.query('cards');
    let highest = 0;
    for (const r of rows) {
      const s = String(r.data.card_id ?? '');
      if (NUMERIC_ID_RE.test(s)) {
        const n = parseInt(s, 10);
        if (!isNaN(n) && n > highest) highest = n;
      }
    }
    return { error: false, message: 'ok', new_id: String(highest + 1) };
  }

  /** Server-shaped card payload (junctions/tags are server-derived). */
  private cardPushData(card: Card): Record<string, unknown> {
    const data: Record<string, unknown> = {
      card_id: (card.card_id ?? '').trim(),
      title: card.title ?? '',
      body: card.body ?? '',
      link: card.link ?? '',
      is_deleted: !!card.is_deleted,
    };
    if (card.schema_id != null) data.card_schema_id = Number(card.schema_id);
    if (card.structured_data != null)
      data.structured_data = card.structured_data;
    return data;
  }

  // ---- tasks -------------------------------------------------------------

  private async taskPushData(task: Task): Promise<Record<string, unknown>> {
    const data: Record<string, unknown> = {
      title: task.title ?? '',
      description: task.description ?? null,
      priority: task.priority ?? null,
      status: task.status ?? 'todo',
      is_complete: !!task.is_complete,
      is_deleted: !!task.is_deleted,
      reminder_sent: !!task.reminder_sent,
    };
    if (task.scheduled_date) data.scheduled_date = toIso(task.scheduled_date);
    if (task.due_date) data.due_date = toIso(task.due_date);
    if (task.completed_at) data.completed_at = toIso(task.completed_at);
    if (task.reminder_time) data.reminder_time = toIso(task.reminder_time);
    if (task.sort_order != null) data.sort_order = Number(task.sort_order);
    // FK translation: always push uuid refs, never raw ints (offline rows
    // have no server PKs; the server resolves in-batch).
    if (task.card_pk) {
      const cardUuid = await this.uuidForCardId(task.card_pk);
      if (cardUuid) data.card_pk_uuid = cardUuid;
    }
    if (task.parent_task_id) {
      const parentUuid = await this.resolveTaskUuid(task.parent_task_id);
      if (parentUuid) data.parent_task_uuid = parentUuid;
    }
    return data;
  }

  /** uuid for a card's UI id (temp or real) — for task FK resolution. */
  private async uuidForCardId(id: number): Promise<string | undefined> {
    return this.resolveCardUuid(id);
  }

  async fetchTasks(params: FetchTasksParams = {}): Promise<Task[]> {
    const {
      showCompleted = false,
      scheduledDate = null,
      completedDate = null,
      status = null,
    } = params;
    const rows = await this.engine.query('tasks');
    const out: Task[] = [];
    for (const row of rows) {
      const task = await this.taskFromRow(row);
      if (task.is_deleted) continue;
      if (!showCompleted && task.is_complete) continue;
      if (status && task.status !== status) continue;
      const day = (d: Date | null) =>
        d ? d.toISOString().split('T')[0] : null;
      if (scheduledDate && day(task.scheduled_date) !== day(scheduledDate))
        continue;
      if (completedDate && day(task.completed_at) !== day(completedDate))
        continue;
      out.push(task);
    }
    out.sort((a, b) => b.updated_at.getTime() - a.updated_at.getTime());
    return out;
  }

  async fetchTask(id: string): Promise<Task> {
    const uuid = await this.resolveTaskUuid(Number(id));
    if (!uuid) throw new Error('task not found');
    const row = await this.engine.getRow('tasks', uuid);
    if (!row) throw new Error('task not found');
    return this.taskFromRow(row);
  }

  async saveNewTask(task: Task): Promise<Task> {
    const rowUuid = crypto.randomUUID();
    const data = await this.taskPushData(task);
    await this.engine.mutate('tasks', { rowUuid, data });
    const tempId = await this.nextLocalId('tasks');
    const aliases = await this.aliases('tasks');
    aliases[String(tempId)] = rowUuid;
    await this.saveAliases('tasks', aliases);
    return { ...task, id: tempId };
  }

  async saveExistingTask(task: Task): Promise<Task> {
    const uuid = await this.resolveTaskUuid(task.id);
    if (!uuid) throw new Error('task not found');
    const existing = await this.engine.getRow('tasks', uuid);
    const pushed = await this.taskPushData(task);
    const data = { ...(existing?.data ?? {}), ...pushed, sync_uuid: uuid };
    delete (data as Record<string, unknown>).id;
    await this.engine.mutate('tasks', { rowUuid: uuid, data });
    return task;
  }

  async deleteTask(id: number): Promise<Task | null> {
    const uuid = await this.resolveTaskUuid(id);
    if (!uuid) return null;
    const existing = await this.engine.getRow('tasks', uuid);
    await this.engine.deleteLocal('tasks', uuid);
    return existing ? this.taskFromRow(existing) : null;
  }

  // ---- tags --------------------------------------------------------------

  async fetchUserTags(): Promise<Tag[]> {
    const rows = (await this.engine.query('tags')).filter(
      (r) => !r.data.is_deleted,
    );
    return rows
      .map((r) => this.tagFromRow(r))
      .sort((a, b) => a.name.localeCompare(b.name));
  }

  async createTag(params: CreateTagParams): Promise<Tag> {
    const rowUuid = crypto.randomUUID();
    const data: Record<string, unknown> = {
      name: params.name,
      color: params.color,
      is_deleted: false,
    };
    await this.engine.mutate('tags', { rowUuid, data });
    return { id: 0, name: params.name, color: params.color, user_id: 0 };
  }

  async deleteTag(id: number): Promise<Tag | null> {
    const rows = await this.engine.query('tags');
    const row = rows.find((r) => Number(r.data.id) === id);
    if (!row) return null;
    const tag = this.tagFromRow(row);
    await this.engine.deleteLocal('tags', row.rowUuid);
    return tag;
  }
}
