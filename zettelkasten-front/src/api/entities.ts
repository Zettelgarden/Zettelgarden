import { Entity, EntityWithScore, EntityCard } from '../models/Card';
import { FactWithCard } from '../models/Fact';
import { apiClient, getData } from './client';

export interface EntityListResponse {
  entities: Entity[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface EntityQueryParams {
  search?: string;
  page?: number;
  per_page?: number;
  sort_by?: 'name' | 'cards' | 'created_at';
  sort_direction?: 'asc' | 'desc';
}

function parseEntityDates(entity: Entity): Entity {
  return {
    ...entity,
    created_at: new Date(entity.created_at),
    updated_at: new Date(entity.updated_at),
  };
}

export function fetchEntities(
  params?: EntityQueryParams,
): Promise<EntityListResponse> {
  const requestParams: Record<string, string | number | boolean | undefined> =
    {};
  if (params?.search) requestParams.search = params.search;
  if (params?.page) requestParams.page = params.page;
  if (params?.per_page) requestParams.per_page = params.per_page;
  if (params?.sort_by) requestParams.sort_by = params.sort_by;
  if (params?.sort_direction)
    requestParams.sort_direction = params.sort_direction;

  return getData(
    apiClient.get<EntityListResponse>('/entities', { params: requestParams }),
  ).then((data) => ({
    ...data,
    entities: data.entities?.map(parseEntityDates) || [],
  }));
}

export function mergeEntities(
  entity1Id: number,
  entity2Id: number,
): Promise<void> {
  return getData(
    apiClient.post<void>('/entities/merge', {
      entity1_id: entity1Id,
      entity2_id: entity2Id,
    }),
  );
}

export function deleteEntity(entityId: number): Promise<void> {
  return getData(apiClient.delete<void>(`/entities/id/${entityId}`));
}

export interface UpdateEntityRequest {
  name: string;
  description: string;
  type: string;
  card_pk: number | null;
}

export function updateEntity(
  entityId: number,
  data: UpdateEntityRequest,
): Promise<void> {
  return getData(apiClient.put<void>(`/entities/id/${entityId}`, data));
}

export function removeEntityFromCard(
  entityId: number,
  cardId: number,
): Promise<void> {
  return getData(
    apiClient.delete<void>(`/entities/${entityId}/cards/${cardId}`),
  );
}

export function addEntityToCard(
  entityId: number,
  cardId: number,
): Promise<void> {
  return getData(apiClient.post<void>(`/entities/${entityId}/cards/${cardId}`));
}

// Fetch entity by ID (new API to avoid case-sensitivity issues)
export function fetchEntityById(id: number): Promise<Entity> {
  return getData(apiClient.get<Entity>(`/entities/id/${id}`)).then(
    parseEntityDates,
  );
}

// Fetch entity by name
export function fetchEntityByName(name: string): Promise<Entity> {
  return getData(
    apiClient.get<Entity>(`/entities/name/${encodeURIComponent(name)}`),
  ).then(parseEntityDates);
}

// Fetch all cards linked to an entity via the junction, each with the card's
// total entity count. Search-independent (works without Typesense).
export async function getEntityCards(entityId: number): Promise<EntityCard[]> {
  const { data } = await apiClient.get<EntityCard[]>(
    `/entities/id/${entityId}/cards`,
  );
  return data;
}

export function getSimilarEntities(
  entityId: number,
): Promise<EntityWithScore[]> {
  return getData(
    apiClient.get<EntityWithScore[]>(`/entities/${entityId}/similar`),
  )
    .then((entities) => entities || [])
    .then((entities) =>
      entities.map((entity) => ({
        ...entity,
        created_at: new Date(entity.created_at),
        updated_at: new Date(entity.updated_at),
      })),
    );
}

// Fetch facts for a given entity
export function getEntityFacts(entityId: number): Promise<FactWithCard[]> {
  return getData(apiClient.get<FactWithCard[]>(`/entities/${entityId}/facts`));
}

// Fetch entities for a given fact
export function getFactEntities(factId: number): Promise<Entity[]> {
  return getData(apiClient.get<Entity[]>(`/facts/${factId}/entities`))
    .then((entities) => entities || [])
    .then((entities) => entities.map(parseEntityDates));
}
